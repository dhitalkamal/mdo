// Package blocks extracts fenced code blocks from markdown so the CLI can copy
// or (later) run them individually.
package blocks

import (
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Block is one fenced code block in a document.
type Block struct {
	Index int    // 1-based position in document order
	Lang  string // info-string language (may be empty)
	Code  string // the code content, including its trailing newline
}

// Extract returns every fenced code block in src, in document order.
func Extract(src []byte) []Block {
	doc := goldmark.New().Parser().Parse(text.NewReader(src))
	var out []Block
	i := 0
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		fc, ok := n.(*ast.FencedCodeBlock)
		if !ok {
			return ast.WalkContinue, nil
		}
		i++
		lang := ""
		if fc.Info != nil {
			seg := fc.Info.Segment
			lang = firstWord(string(seg.Value(src)))
		}
		out = append(out, Block{Index: i, Lang: lang, Code: linesText(fc, src)})
		return ast.WalkContinue, nil
	})
	return out
}

func linesText(n ast.Node, src []byte) string {
	var b strings.Builder
	l := n.Lines()
	for i := 0; i < l.Len(); i++ {
		seg := l.At(i)
		b.Write(seg.Value(src))
	}
	return b.String()
}

func firstWord(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		return s[:i]
	}
	return s
}
