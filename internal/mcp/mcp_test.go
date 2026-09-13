package mcp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func serveWith(t *testing.T, input string) []rpcResponse {
	t.Helper()
	var out bytes.Buffer
	if err := Serve(strings.NewReader(input), &out, "test"); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	var resps []rpcResponse
	dec := json.NewDecoder(&out)
	for dec.More() {
		var r rpcResponse
		if err := dec.Decode(&r); err != nil {
			t.Fatalf("decode: %v", err)
		}
		resps = append(resps, r)
	}
	return resps
}

func toolText(t *testing.T, r rpcResponse) string {
	t.Helper()
	m, ok := r.Result.(map[string]any)
	if !ok {
		t.Fatalf("result not a map: %#v", r.Result)
	}
	content, _ := m["content"].([]any)
	if len(content) == 0 {
		t.Fatalf("no content: %#v", m)
	}
	first, _ := content[0].(map[string]any)
	s, _ := first["text"].(string)
	return s
}

func TestInitialize(t *testing.T) {
	resps := serveWith(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	if len(resps) != 1 {
		t.Fatalf("got %d responses, want 1", len(resps))
	}
	m, _ := resps[0].Result.(map[string]any)
	if m["protocolVersion"] != protocolVersion {
		t.Errorf("protocolVersion = %v, want %v", m["protocolVersion"], protocolVersion)
	}
}

func TestNotificationNoResponse(t *testing.T) {
	if resps := serveWith(t, `{"jsonrpc":"2.0","method":"notifications/initialized"}`); len(resps) != 0 {
		t.Errorf("notification produced %d responses, want 0", len(resps))
	}
}

func TestToolsList(t *testing.T) {
	resps := serveWith(t, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	m, _ := resps[0].Result.(map[string]any)
	tools, _ := m["tools"].([]any)
	if len(tools) != 3 {
		t.Fatalf("got %d tools, want 3", len(tools))
	}
}

func TestToolCallRender(t *testing.T) {
	resps := serveWith(t, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"render_markdown","arguments":{"content":"# Hi there"}}}`)
	if txt := toolText(t, resps[0]); !strings.Contains(txt, "Hi there") {
		t.Errorf("render result = %q, want 'Hi there'", txt)
	}
}

func TestToolCallListBlocks(t *testing.T) {
	req := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"list_code_blocks","arguments":{"content":"` + "```sh\\nls\\n```" + `"}}}`
	resps := serveWith(t, req)
	if txt := toolText(t, resps[0]); !strings.Contains(txt, "sh") || !strings.Contains(txt, "ls") {
		t.Errorf("list result = %q", txt)
	}
}

func TestToolCallGetBlockOutOfRange(t *testing.T) {
	resps := serveWith(t, `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"get_code_block","arguments":{"content":"no blocks","index":1}}}`)
	m, _ := resps[0].Result.(map[string]any)
	if m["isError"] != true {
		t.Errorf("expected isError true, got %v", m)
	}
}

func TestUnknownMethod(t *testing.T) {
	resps := serveWith(t, `{"jsonrpc":"2.0","id":6,"method":"bogus"}`)
	if resps[0].Error == nil {
		t.Errorf("expected error for unknown method")
	}
}
