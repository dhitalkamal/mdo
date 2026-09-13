package blocks

import "testing"

func TestExtract(t *testing.T) {
	src := []byte("# Title\n\n```go\nfmt.Println(\"hi\")\n```\n\nsome text\n\n```sh\nls -la\necho done\n```\n")
	got := Extract(src)
	if len(got) != 2 {
		t.Fatalf("got %d blocks, want 2", len(got))
	}
	if got[0].Index != 1 || got[0].Lang != "go" {
		t.Errorf("block 0 = %+v, want index 1 lang go", got[0])
	}
	if got[0].Code != "fmt.Println(\"hi\")\n" {
		t.Errorf("block 0 code = %q", got[0].Code)
	}
	if got[1].Index != 2 || got[1].Lang != "sh" {
		t.Errorf("block 1 = %+v, want index 2 lang sh", got[1])
	}
	if got[1].Code != "ls -la\necho done\n" {
		t.Errorf("block 1 code = %q", got[1].Code)
	}
}

func TestExtractNone(t *testing.T) {
	if got := Extract([]byte("# just text\n\nno code here\n")); len(got) != 0 {
		t.Errorf("got %d blocks, want 0", len(got))
	}
}
