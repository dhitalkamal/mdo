//go:build release

package release

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	// run key, with or without the leading "- " of a list item
	runKeyRe = regexp.MustCompile(`^(\s*)(?:-\s+)?run:\s*(.*)$`)
	// block scalar indicators: |, >, with optional chomping or indent digits
	blockIndRe = regexp.MustCompile(`^[|>][+-]?[0-9]?\s*(#.*)?$`)
	goBuildRe  = regexp.MustCompile(`(^|[^A-Za-z0-9_./-])go\s+build(\s|$)`)
	scriptRe   = regexp.MustCompile(`scripts/build-release\.sh(\s.*)?$`)
	tagArgRe   = regexp.MustCompile(`^["']?(\$\{\{\s*github\.ref_name\s*\}\}|\$GITHUB_REF_NAME|\$\{GITHUB_REF_NAME\})["']?$`)
)

// readWorkflow loads the release workflow from the repo root.
func readWorkflow(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "release.yml"))
	if err != nil {
		t.Fatalf("read release workflow: %v", err)
	}
	return string(data)
}

// runCommands returns every shell command line from the workflow's run steps,
// with backslash continuations joined and comment-only lines dropped. Text
// inspection is used because no yaml library is in go.mod.
func runCommands(workflow string) []string {
	lines := strings.Split(workflow, "\n")
	var cmds []string
	for i := 0; i < len(lines); i++ {
		m := runKeyRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		keyIndent := len(m[1])
		val := strings.TrimSpace(m[2])
		var body []string
		if blockIndRe.MatchString(val) {
			// block scalar: every following line indented deeper than the key
			for i+1 < len(lines) {
				next := lines[i+1]
				if strings.TrimSpace(next) != "" && len(next)-len(strings.TrimLeft(next, " ")) <= keyIndent {
					break
				}
				body = append(body, next)
				i++
			}
		} else {
			body = []string{val}
		}
		var joined string
		for _, l := range body {
			l = strings.TrimSpace(l)
			if strings.HasSuffix(l, `\`) {
				joined += strings.TrimSuffix(l, `\`) + " "
				continue
			}
			joined += l
			if joined != "" && !strings.HasPrefix(joined, "#") {
				cmds = append(cmds, joined)
			}
			joined = ""
		}
	}
	return cmds
}

// A tag push must build through the script so the static flags live in one
// place, and the script must get the tag as its version argument.
func TestWorkflowBuildsThroughTheScriptWithTheTagAsVersion(t *testing.T) {
	cmds := runCommands(readWorkflow(t))
	for _, c := range cmds {
		m := scriptRe.FindStringSubmatch(c)
		if m == nil {
			continue
		}
		args := strings.Fields(m[1])
		if len(args) > 0 && tagArgRe.MatchString(args[0]) {
			return
		}
	}
	t.Fatalf("no run step invokes scripts/build-release.sh with the tag (github.ref_name or GITHUB_REF_NAME) as first argument; run commands: %q", cmds)
}

// An inline go build would bypass CGO_ENABLED=0 and ship a dynamic binary.
func TestWorkflowRunStepsNeverCallGoBuildDirectly(t *testing.T) {
	for _, c := range runCommands(readWorkflow(t)) {
		if goBuildRe.MatchString(c) {
			t.Errorf("run step calls go build directly: %q", c)
		}
	}
}

// Publishing must keep shipping the tarballs the script writes into dist/.
func TestWorkflowPublishesDistTarballs(t *testing.T) {
	if !strings.Contains(readWorkflow(t), "dist/*.tar.gz") {
		t.Error("workflow does not upload dist/*.tar.gz")
	}
}
