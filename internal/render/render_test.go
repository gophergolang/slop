package render

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/vibeguard/vibeguard/internal/ir"
)

type fakeBackend struct {
	files FileSet
}

func (fakeBackend) Name() string { return "fake" }
func (b fakeBackend) Plan(_ *ir.Application) (FileSet, error) {
	return b.files, nil
}

func TestEngineWritesFiles(t *testing.T) {
	dir := t.TempDir()
	engine := &Engine{
		Root: dir,
		Mode: ModeWrite,
		Backends: []Backend{
			fakeBackend{files: FileSet{
				{Path: "a.txt", Content: []byte("hello\n")},
				{Path: "sub/b.txt", Content: []byte("world\n"), Header: "// gen"},
			}},
		},
	}
	report, err := engine.Run(&ir.Application{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if got := len(report.FilesWritten); got != 2 {
		t.Errorf("FilesWritten: got %d", got)
	}
	got, err := os.ReadFile(filepath.Join(dir, "sub/b.txt"))
	if err != nil {
		t.Fatal(err)
	}
	want := "// gen\n\nworld\n"
	if string(got) != want {
		t.Errorf("b.txt: got %q want %q", got, want)
	}
}

func TestEngineKeepIfExistsSkipsExistingFile(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "stub.go")
	if err := os.WriteFile(existing, []byte("// developer wrote this\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	engine := &Engine{
		Root: dir,
		Mode: ModeWrite,
		Backends: []Backend{
			fakeBackend{files: FileSet{
				{Path: "stub.go", Content: []byte("// generator stub\n"), KeepIfExists: true},
				{Path: "wrapper.go", Content: []byte("// wrapper\n")},
			}},
		},
	}
	report, err := engine.Run(&ir.Application{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(existing)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "// developer wrote this\n" {
		t.Errorf("stub overwritten: %q", got)
	}
	if len(report.Skipped) != 1 || report.Skipped[0] != "stub.go" {
		t.Errorf("Skipped: got %v want [stub.go]", report.Skipped)
	}
	wrapped, err := os.ReadFile(filepath.Join(dir, "wrapper.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(wrapped) != "// wrapper\n" {
		t.Errorf("wrapper not written: %q", wrapped)
	}
}

func TestEngineDryRunDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	engine := &Engine{
		Root: dir,
		Mode: ModeDryRun,
		Out:  &buf,
		Backends: []Backend{
			fakeBackend{files: FileSet{{Path: "a.txt", Content: []byte("x")}}},
		},
	}
	if _, err := engine.Run(&ir.Application{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "a.txt")); !os.IsNotExist(err) {
		t.Errorf("dry-run wrote a file: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("would write")) {
		t.Errorf("dry-run output didn't say 'would write': %s", buf.String())
	}
}
