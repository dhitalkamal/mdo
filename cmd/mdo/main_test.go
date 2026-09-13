package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunRenders(t *testing.T) {
	var out, errb bytes.Buffer
	in := strings.NewReader("# Title\n\nhello world\n")
	if err := run([]string{"-"}, in, &out, &errb); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out.String(), "Title") || !strings.Contains(out.String(), "hello world") {
		t.Errorf("render output missing content: %q", out.String())
	}
}

func TestRunListsBlocks(t *testing.T) {
	var out, errb bytes.Buffer
	in := strings.NewReader("```go\nfmt.Println(1)\n```\n")
	if err := run([]string{"-l", "-"}, in, &out, &errb); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out.String(), "go") || !strings.Contains(out.String(), "fmt.Println(1)") {
		t.Errorf("list output = %q", out.String())
	}
}

func TestRunCopiesBlockAsOSC52(t *testing.T) {
	var out, errb bytes.Buffer
	in := strings.NewReader("```sh\nls -la\n```\n")
	if err := run([]string{"-c", "1", "-"}, in, &out, &errb); err != nil {
		t.Fatalf("run: %v", err)
	}
	// trailing newline is stripped so the last line waits for Enter on paste.
	want := "\x1b]52;c;bHMgLWxh\x1b\\" // base64("ls -la") = bHMgLWxh
	if out.String() != want {
		t.Errorf("copy output = %q, want %q", out.String(), want)
	}
}

func TestRunVersion(t *testing.T) {
	var out, errb bytes.Buffer
	if err := run([]string{"--version"}, strings.NewReader(""), &out, &errb); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), version) {
		t.Errorf("version output = %q, want to contain %q", out.String(), version)
	}
}

func TestRunCopyOutOfRange(t *testing.T) {
	var out, errb bytes.Buffer
	in := strings.NewReader("no blocks here\n")
	if err := run([]string{"-c", "1", "-"}, in, &out, &errb); err == nil {
		t.Error("expected error for out-of-range copy")
	}
}
