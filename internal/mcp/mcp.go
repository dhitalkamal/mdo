// Package mcp serves mdo's capabilities to AI agents over the Model Context
// Protocol (JSON-RPC 2.0 on stdio). Agents can render markdown and inspect a
// document's code blocks without shelling out - the "AI drives mdo" surface.
package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/dhitalkamal/mdo/internal/blocks"
	"github.com/dhitalkamal/mdo/internal/render"
	"github.com/dhitalkamal/mdo/internal/term"
)

const protocolVersion = "2024-11-05"

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
	handler     func(args map[string]any) (string, error)
}

// Serve reads newline-delimited JSON-RPC from in and writes responses to out
// until in is exhausted. version is reported to the client on initialize.
func Serve(in io.Reader, out io.Writer, version string) error {
	tools := buildTools()
	dec := json.NewDecoder(in)
	enc := json.NewEncoder(out)
	for {
		var req rpcRequest
		if err := dec.Decode(&req); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if resp := handle(&req, tools, version); resp != nil {
			if err := enc.Encode(resp); err != nil {
				return err
			}
		}
	}
}

func handle(req *rpcRequest, tools []tool, version string) *rpcResponse {
	// notifications (no id) get no response
	notification := len(req.ID) == 0

	switch req.Method {
	case "initialize":
		return &rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "mdo", "version": version},
		}}
	case "notifications/initialized", "notifications/cancelled":
		return nil
	case "tools/list":
		return &rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"tools": tools}}
	case "tools/call":
		if notification {
			return nil
		}
		return callTool(req, tools)
	default:
		if notification {
			return nil
		}
		return &rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32601, Message: "method not found: " + req.Method}}
	}
}

func callTool(req *rpcRequest, tools []tool) *rpcResponse {
	var p struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	_ = json.Unmarshal(req.Params, &p)
	for _, t := range tools {
		if t.Name != p.Name {
			continue
		}
		text, err := t.handler(p.Arguments)
		if err != nil {
			return toolResult(req.ID, "error: "+err.Error(), true)
		}
		return toolResult(req.ID, text, false)
	}
	return &rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32602, Message: "unknown tool: " + p.Name}}
}

func toolResult(id json.RawMessage, text string, isErr bool) *rpcResponse {
	return &rpcResponse{JSONRPC: "2.0", ID: id, Result: map[string]any{
		"content": []map[string]any{{"type": "text", "text": text}},
		"isError": isErr,
	}}
}

func buildTools() []tool {
	srcArgs := `{"type":"object","properties":{"path":{"type":"string","description":"path to a markdown file"},"content":{"type":"string","description":"markdown content (use instead of path)"}}}`
	return []tool{
		{
			Name:        "render_markdown",
			Description: "Render a markdown document to plain, readable terminal text. Provide 'path' or 'content'.",
			InputSchema: json.RawMessage(srcArgs),
			handler: func(a map[string]any) (string, error) {
				src, err := loadSource(a)
				if err != nil {
					return "", err
				}
				return render.Render(src, render.Options{Level: term.LevelNone, Width: 100}), nil
			},
		},
		{
			Name:        "list_code_blocks",
			Description: "List the fenced code blocks in a markdown document as JSON ({index, lang, code}). Provide 'path' or 'content'.",
			InputSchema: json.RawMessage(srcArgs),
			handler: func(a map[string]any) (string, error) {
				src, err := loadSource(a)
				if err != nil {
					return "", err
				}
				b, _ := json.MarshalIndent(blocks.Extract(src), "", "  ")
				return string(b), nil
			},
		},
		{
			Name:        "get_code_block",
			Description: "Return the code of the Nth (1-based) fenced code block. Args: 'index' plus 'path' or 'content'.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"},"content":{"type":"string"},"index":{"type":"integer","description":"1-based block index"}},"required":["index"]}`),
			handler: func(a map[string]any) (string, error) {
				src, err := loadSource(a)
				if err != nil {
					return "", err
				}
				idx := 0
				if f, ok := a["index"].(float64); ok {
					idx = int(f)
				}
				bs := blocks.Extract(src)
				if idx < 1 || idx > len(bs) {
					return "", fmt.Errorf("no code block %d (found %d)", idx, len(bs))
				}
				return bs[idx-1].Code, nil
			},
		},
	}
}

func loadSource(a map[string]any) ([]byte, error) {
	if c, ok := a["content"].(string); ok && c != "" {
		return []byte(c), nil
	}
	if p, ok := a["path"].(string); ok && p != "" {
		return os.ReadFile(p)
	}
	return nil, fmt.Errorf("provide 'path' or 'content'")
}
