// Package render turns Markdown into styled ANSI text for the terminal. It
// parses with goldmark and walks the AST, applying a theme palette and wrapping
// prose to the terminal width. Code blocks are highlighted with chroma.
package render

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"

	"github.com/dhitalkamal/mdo/internal/term"
	"github.com/dhitalkamal/mdo/internal/theme"
)

// Options controls a render.
type Options struct {
	Level term.Level // color capability
	Width int        // wrap width in columns (<= 0 disables wrapping)
}

// Render returns src rendered as ANSI text.
func Render(src []byte, opts Options) string {
	md := goldmark.New(goldmark.WithExtensions(extension.GFM))
	doc := md.Parser().Parse(text.NewReader(src))
	r := &renderer{src: src, pal: theme.For(opts.Level), opts: opts}
	r.blocks(doc, "")
	return strings.TrimRight(r.b.String(), "\n") + "\n"
}

type renderer struct {
	src  []byte
	pal  theme.Palette
	opts Options
	b    strings.Builder
}

func (r *renderer) writeLine(s string) { r.b.WriteString(s); r.b.WriteByte('\n') }

// blank writes a separating blank line, collapsing consecutive blanks.
func (r *renderer) blank() {
	s := r.b.String()
	if s == "" || strings.HasSuffix(s, "\n\n") {
		return
	}
	r.b.WriteByte('\n')
}

// renderToString renders n's block children into a standalone string.
func (r *renderer) renderToString(n ast.Node, indent string) string {
	saved := r.b
	r.b = strings.Builder{}
	r.blocks(n, indent)
	out := r.b.String()
	r.b = saved
	return out
}

func (r *renderer) blocks(n ast.Node, indent string) {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		r.block(c, indent)
	}
}

func (r *renderer) block(n ast.Node, indent string) {
	switch t := n.(type) {
	case *ast.Heading:
		r.writeLine(indent + r.headingStyle(t.Level).Apply(r.inline(n)))
		r.blank()
	case *ast.Paragraph:
		r.paragraph(n, indent)
		r.blank()
	case *ast.TextBlock: // tight list item content: no trailing blank
		r.paragraph(n, indent)
	case *ast.FencedCodeBlock:
		r.codeBlock(fencedLang(t, r.src), r.codeText(n), indent)
		r.blank()
	case *ast.CodeBlock:
		r.codeBlock("", r.codeText(n), indent)
		r.blank()
	case *ast.Blockquote:
		r.blockquote(n, indent)
		r.blank()
	case *ast.List:
		r.list(t, indent)
		r.blank()
	case *ast.ThematicBreak:
		w := max(1, r.opts.Width-visibleLen(indent))
		r.writeLine(indent + r.pal.Rule.Apply(strings.Repeat("-", w)))
		r.blank()
	case *extast.Table:
		r.table(n, indent)
		r.blank()
	default:
		r.blocks(n, indent)
	}
}

func (r *renderer) paragraph(n ast.Node, indent string) {
	txt := r.inline(n)
	wrapped := wrap(txt, r.opts.Width-visibleLen(indent))
	for _, ln := range strings.Split(wrapped, "\n") {
		r.writeLine(indent + ln)
	}
}

func (r *renderer) blockquote(n ast.Node, indent string) {
	content := strings.TrimRight(r.renderToString(n, ""), "\n")
	gutter := r.pal.Quote.Apply("|") + " "
	for _, ln := range strings.Split(content, "\n") {
		r.writeLine(indent + gutter + ln)
	}
}

func (r *renderer) list(l *ast.List, indent string) {
	num := l.Start
	if num == 0 {
		num = 1
	}
	for item := l.FirstChild(); item != nil; item = item.NextSibling() {
		marker := r.pal.Bullet.Apply("-") + " "
		if l.IsOrdered() {
			marker = r.pal.Bullet.Apply(fmt.Sprintf("%d.", num)) + " "
			num++
		}
		content := strings.TrimRight(r.renderToString(item, ""), "\n")
		pad := strings.Repeat(" ", visibleLen(marker))
		for j, ln := range strings.Split(content, "\n") {
			if j == 0 {
				r.writeLine(indent + marker + ln)
			} else {
				r.writeLine(indent + pad + ln)
			}
		}
	}
}

func (r *renderer) table(n ast.Node, indent string) {
	var rows [][]string
	for row := n.FirstChild(); row != nil; row = row.NextSibling() {
		var cells []string
		for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
			cells = append(cells, r.inline(cell))
		}
		rows = append(rows, cells)
	}
	if len(rows) == 0 {
		return
	}
	ncol := 0
	for _, row := range rows {
		if len(row) > ncol {
			ncol = len(row)
		}
	}
	widths := make([]int, ncol)
	for _, row := range rows {
		for i, c := range row {
			if w := visibleLen(c); w > widths[i] {
				widths[i] = w
			}
		}
	}
	for ri, row := range rows {
		var b strings.Builder
		for i := 0; i < ncol; i++ {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			b.WriteString(cell)
			b.WriteString(strings.Repeat(" ", widths[i]-visibleLen(cell)))
			if i < ncol-1 {
				b.WriteString("  ")
			}
		}
		r.writeLine(indent + strings.TrimRight(b.String(), " "))
		if ri == 0 { // separator under the header row
			var s strings.Builder
			for i := 0; i < ncol; i++ {
				s.WriteString(strings.Repeat("-", widths[i]))
				if i < ncol-1 {
					s.WriteString("  ")
				}
			}
			r.writeLine(indent + r.pal.Rule.Apply(s.String()))
		}
	}
}

func (r *renderer) codeBlock(lang, code, indent string) {
	code = strings.TrimRight(code, "\n")
	emit := func(s string) {
		for _, ln := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
			r.writeLine(indent + "    " + ln)
		}
	}
	if r.opts.Level == term.LevelNone {
		emit(code)
		return
	}
	var buf bytes.Buffer
	if err := quick.Highlight(&buf, code, lang, "terminal256", "monokai"); err != nil {
		emit(code)
		return
	}
	emit(buf.String())
}

// inline renders a node's inline children to a styled string.
func (r *renderer) inline(n ast.Node) string {
	var b strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch t := c.(type) {
		case *ast.Text:
			seg := t.Segment
			b.Write(seg.Value(r.src))
			if t.SoftLineBreak() {
				b.WriteByte(' ')
			}
			if t.HardLineBreak() {
				b.WriteByte('\n')
			}
		case *ast.String:
			b.Write(t.Value)
		case *ast.CodeSpan:
			b.WriteString(r.pal.Code.Apply(r.textOf(t)))
		case *extast.Strikethrough:
			b.WriteString(r.pal.Strike.Apply(r.inline(t)))
		case *extast.TaskCheckBox:
			if t.IsChecked {
				b.WriteString("[x] ")
			} else {
				b.WriteString("[ ] ")
			}
		case *ast.Emphasis:
			if t.Level >= 2 {
				b.WriteString(r.pal.Bold.Apply(r.inline(t)))
			} else {
				b.WriteString(r.pal.Italic.Apply(r.inline(t)))
			}
		case *ast.Link:
			txt := r.inline(t)
			b.WriteString(r.pal.Link.Apply(txt))
			if dest := string(t.Destination); dest != "" && dest != txt {
				b.WriteString(" (" + dest + ")")
			}
		case *ast.AutoLink:
			b.WriteString(r.pal.Link.Apply(string(t.URL(r.src))))
		case *ast.Image:
			b.WriteString("[image: " + r.inline(t) + "]")
		case *ast.RawHTML, *ast.HTMLBlock:
			// skip raw HTML for clean terminal output
		default:
			b.WriteString(r.inline(c))
		}
	}
	return b.String()
}

// textOf returns the concatenated literal text of a node's Text children.
func (r *renderer) textOf(n ast.Node) string {
	var b strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			seg := t.Segment
			b.Write(seg.Value(r.src))
		}
	}
	return b.String()
}

func (r *renderer) headingStyle(level int) theme.Style {
	switch level {
	case 1:
		return r.pal.H1
	case 2:
		return r.pal.H2
	default:
		return r.pal.H3
	}
}

func (r *renderer) codeText(n ast.Node) string {
	var b strings.Builder
	l := n.Lines()
	for i := 0; i < l.Len(); i++ {
		seg := l.At(i)
		b.Write(seg.Value(r.src))
	}
	return b.String()
}

func fencedLang(fc *ast.FencedCodeBlock, src []byte) string {
	if fc.Info == nil {
		return ""
	}
	seg := fc.Info.Segment
	info := strings.TrimSpace(string(seg.Value(src)))
	if i := strings.IndexAny(info, " \t"); i >= 0 {
		return info[:i]
	}
	return info
}
