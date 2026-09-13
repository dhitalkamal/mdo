// Package tui is a full-screen pager for rendered markdown: scroll, in-document
// search, and a table-of-contents jump. Built on bubbletea's viewport.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Heading is a table-of-contents entry: display text and the content line it is on.
type Heading struct {
	Text string
	Line int
}

// stripANSI removes escape sequences so text can be matched/measured.
func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if inEsc {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		if r == '\x1b' {
			inEsc = true
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// TOC maps each heading text to the first content line that contains it.
func TOC(lines, texts []string) []Heading {
	var hs []Heading
	start := 0
	for _, t := range texts {
		for i := start; i < len(lines); i++ {
			if t != "" && strings.Contains(stripANSI(lines[i]), t) {
				hs = append(hs, Heading{Text: t, Line: i})
				start = i + 1
				break
			}
		}
	}
	return hs
}

// findMatches returns the indices of lines containing query (case-insensitive).
func findMatches(lines []string, query string) []int {
	if query == "" {
		return nil
	}
	q := strings.ToLower(query)
	var out []int
	for i, ln := range lines {
		if strings.Contains(strings.ToLower(stripANSI(ln)), q) {
			out = append(out, i)
		}
	}
	return out
}

type mode int

const (
	modeNormal mode = iota
	modeSearch
	modeTOC
)

type model struct {
	vp       viewport.Model
	lines    []string
	content  string
	headings []Heading
	ready    bool
	mode     mode
	search   string
	matches  []int
	matchIdx int
	tocIdx   int
}

var statusStyle = lipgloss.NewStyle().Reverse(true)

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if !m.ready {
			m.vp = viewport.New(msg.Width, msg.Height-1)
			m.vp.SetContent(m.content)
			m.ready = true
		} else {
			m.vp.Width = msg.Width
			m.vp.Height = msg.Height - 1
		}
		return m, nil
	case tea.KeyMsg:
		switch m.mode {
		case modeSearch:
			return m.updateSearch(msg)
		case modeTOC:
			return m.updateTOC(msg)
		default:
			return m.updateNormal(msg)
		}
	}
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func (m model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "/":
		m.mode = modeSearch
		m.search = ""
		return m, nil
	case "t":
		if len(m.headings) > 0 {
			m.mode = modeTOC
			m.tocIdx = 0
		}
		return m, nil
	case "n":
		return m.jumpMatch(1), nil
	case "N":
		return m.jumpMatch(-1), nil
	}
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func (m model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.matches = findMatches(m.lines, m.search)
		m.matchIdx = 0
		m.mode = modeNormal
		if len(m.matches) > 0 {
			m.vp.SetYOffset(m.matches[0])
		}
		return m, nil
	case "esc":
		m.mode = modeNormal
		return m, nil
	case "backspace":
		if m.search != "" {
			m.search = m.search[:len(m.search)-1]
		}
		return m, nil
	default:
		if len(msg.Runes) > 0 {
			m.search += string(msg.Runes)
		}
		return m, nil
	}
}

func (m model) updateTOC(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.tocIdx < len(m.headings)-1 {
			m.tocIdx++
		}
	case "k", "up":
		if m.tocIdx > 0 {
			m.tocIdx--
		}
	case "enter":
		m.vp.SetYOffset(m.headings[m.tocIdx].Line)
		m.mode = modeNormal
	case "esc", "t", "q":
		m.mode = modeNormal
	}
	return m, nil
}

func (m model) jumpMatch(dir int) model {
	if len(m.matches) == 0 {
		return m
	}
	m.matchIdx = (m.matchIdx + dir + len(m.matches)) % len(m.matches)
	m.vp.SetYOffset(m.matches[m.matchIdx])
	return m
}

func (m model) View() string {
	if !m.ready {
		return "loading..."
	}
	if m.mode == modeTOC {
		return m.tocView()
	}
	var status string
	switch m.mode {
	case modeSearch:
		status = "/" + m.search
	default:
		status = fmt.Sprintf(" %3.0f%%  j/k scroll  / search  n/N next  t contents  q quit", m.vp.ScrollPercent()*100)
	}
	return m.vp.View() + "\n" + statusStyle.Width(m.vp.Width).Render(status)
}

func (m model) tocView() string {
	var b strings.Builder
	b.WriteString("Contents  (j/k move, enter jump, esc back)\n\n")
	for i, h := range m.headings {
		prefix := "  "
		if i == m.tocIdx {
			prefix = "> "
		}
		b.WriteString(prefix + h.Text + "\n")
	}
	return b.String()
}

// Run opens the pager on content (rendered markdown). tocTexts are heading
// strings used to build the jump list.
func Run(content string, tocTexts []string) error {
	lines := strings.Split(content, "\n")
	m := model{
		lines:    lines,
		content:  content,
		headings: TOC(lines, tocTexts),
	}
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}
