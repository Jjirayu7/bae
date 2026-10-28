package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEditFileRejectsAmbiguousTarget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(path, []byte("x\nx\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := EditFile(path, "x", "y"); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguous edit error, got %v", err)
	}
}

func TestEditFileReplacesSingleTarget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(path, []byte("before"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := EditFile(path, "before", "after"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "after" {
		t.Fatalf("got %q", b)
	}
}

func TestRunBashHonorsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := RunBash(ctx, "printf hello"); err == nil {
		t.Fatal("expected cancelled command to fail")
	}
}
