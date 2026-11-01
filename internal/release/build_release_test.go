//go:build release

// Package release tests scripts/build-release.sh end to end. It sits behind
// the release tag because it cross-compiles four binaries and is slow.
package release

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"debug/elf"
	"debug/macho"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const testVersion = "v9.8.7-reltest"

// repoRoot walks up from the package dir to the directory holding go.mod.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working dir: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above test dir")
		}
		dir = parent
	}
}

// copyRepo copies the repo minus .git and dist so the script's dist/ output
// lands in a temp dir and never in the real checkout.
func copyRepo(t *testing.T, src string) string {
	t.Helper()
	dst := t.TempDir()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if d.IsDir() && (rel == ".git" || rel == "dist") {
			return filepath.SkipDir
		}
		target := filepath.Join(dst, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		switch {
		case d.IsDir():
			return os.MkdirAll(target, 0o755)
		case info.Mode().IsRegular():
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(target, data, info.Mode().Perm())
		}
		return nil // skip symlinks and specials, the build does not need them
	})
	if err != nil {
		t.Fatalf("copy repo: %v", err)
	}
	return dst
}

// runScript builds the release in a repo copy and returns that copy's dist dir.
func runScript(t *testing.T) string {
	t.Helper()
	work := copyRepo(t, repoRoot(t))
	script := filepath.Join(work, "scripts", "build-release.sh")
	if _, err := os.Stat(script); err != nil {
		t.Fatalf("scripts/build-release.sh is missing: %v", err)
	}
	cmd := exec.Command("scripts/build-release.sh", testVersion)
	cmd.Dir = work
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run build-release.sh %s: %v\n%s", testVersion, err, out)
	}
	return filepath.Join(work, "dist")
}

// readSingleBinary returns the contents of the only file in a tarball,
// failing if the archive holds anything else or the name is not mdo.
func readSingleBinary(t *testing.T, tarball string) []byte {
	t.Helper()
	f, err := os.Open(tarball)
	if err != nil {
		t.Fatalf("open tarball: %v", err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gunzip %s: %v", tarball, err)
	}
	tr := tar.NewReader(gz)
	var names []string
	var data []byte
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read tar %s: %v", tarball, err)
		}
		names = append(names, hdr.Name)
		if data, err = io.ReadAll(tr); err != nil {
			t.Fatalf("read entry %s: %v", hdr.Name, err)
		}
	}
	if len(names) != 1 || names[0] != "mdo" {
		t.Fatalf("%s entries = %q, want exactly [\"mdo\"]", filepath.Base(tarball), names)
	}
	return data
}

func TestBuildReleaseWritesFourPlatformTarballsEachWithAVersionedMdoBinary(t *testing.T) {
	dist := runScript(t)

	entries, err := os.ReadDir(dist)
	if err != nil {
		t.Fatalf("read dist: %v", err)
	}
	var got []string
	for _, e := range entries {
		got = append(got, e.Name())
	}
	sort.Strings(got)
	want := []string{
		"mdo_" + testVersion + "_darwin_amd64.tar.gz",
		"mdo_" + testVersion + "_darwin_arm64.tar.gz",
		"mdo_" + testVersion + "_linux_amd64.tar.gz",
		"mdo_" + testVersion + "_linux_arm64.tar.gz",
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("dist contents = %v, want %v", got, want)
	}

	checks := map[string]func(t *testing.T, bin []byte){
		"linux_amd64": func(t *testing.T, bin []byte) {
			f, err := elf.NewFile(bytes.NewReader(bin))
			if err != nil {
				t.Fatalf("not an ELF binary: %v", err)
			}
			if f.Machine != elf.EM_X86_64 {
				t.Errorf("elf machine = %v, want %v", f.Machine, elf.EM_X86_64)
			}
		},
		"linux_arm64": func(t *testing.T, bin []byte) {
			f, err := elf.NewFile(bytes.NewReader(bin))
			if err != nil {
				t.Fatalf("not an ELF binary: %v", err)
			}
			if f.Machine != elf.EM_AARCH64 {
				t.Errorf("elf machine = %v, want %v", f.Machine, elf.EM_AARCH64)
			}
		},
		"darwin_amd64": func(t *testing.T, bin []byte) {
			f, err := macho.NewFile(bytes.NewReader(bin))
			if err != nil {
				t.Fatalf("not a Mach-O binary: %v", err)
			}
			if f.Cpu != macho.CpuAmd64 {
				t.Errorf("macho cpu = %v, want %v", f.Cpu, macho.CpuAmd64)
			}
		},
		"darwin_arm64": func(t *testing.T, bin []byte) {
			f, err := macho.NewFile(bytes.NewReader(bin))
			if err != nil {
				t.Fatalf("not a Mach-O binary: %v", err)
			}
			if f.Cpu != macho.CpuArm64 {
				t.Errorf("macho cpu = %v, want %v", f.Cpu, macho.CpuArm64)
			}
		},
	}
	for platform, check := range checks {
		t.Run(platform, func(t *testing.T) {
			bin := readSingleBinary(t, filepath.Join(dist, "mdo_"+testVersion+"_"+platform+".tar.gz"))
			check(t, bin)
			if !bytes.Contains(bin, []byte(testVersion)) {
				t.Errorf("binary does not embed version %q", testVersion)
			}
		})
	}
}
