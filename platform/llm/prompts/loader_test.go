package prompts

import (
	"errors"
	"strings"
	"testing"
)

func TestSealAndParseRoundTrip(t *testing.T) {
	body := "Hello {{.Name}}!\n"
	raw := []byte("---\nmodel_default: claude-haiku-4-5\nsha256: \"\"\n---\n" + body)
	sealed, err := Seal(raw)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	p, err := Parse("test", "0.1.0", sealed)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if p.Body != body {
		t.Errorf("body round-trip failed: got %q want %q", p.Body, body)
	}
	if p.Frontmatter.ModelDefault != "claude-haiku-4-5" {
		t.Errorf("model_default: got %q", p.Frontmatter.ModelDefault)
	}
	if p.Frontmatter.SHA256 == "" {
		t.Error("sha256 frontmatter empty after seal")
	}
}

func TestSealMismatchRejected(t *testing.T) {
	original := []byte("---\nmodel_default: foo\nsha256: \"\"\n---\nbody one\n")
	sealed, err := Seal(original)
	if err != nil {
		t.Fatal(err)
	}
	mutated := strings.Replace(string(sealed), "body one", "body two", 1)
	if _, err := Parse("test", "0.1.0", []byte(mutated)); err == nil {
		t.Fatal("Parse accepted mutated body without seal mismatch")
	} else if !errors.Is(err, ErrSealMismatch) {
		t.Errorf("got error %v, want ErrSealMismatch", err)
	}
}
