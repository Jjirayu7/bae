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

func TestResolveWorkspacePathRejectsOutsidePath(t *testing.T) {
	if _, err := resolveWorkspacePath(filepath.Join(workspaceRoot, "..", "outside")); err == nil {
		t.Fatal("expected path outside workspace to be rejected")
	}
}

func TestResolveWorkspacePathRejectsSymlinkEscape(t *testing.T) {
	tmp := t.TempDir()
	link := filepath.Join(workspaceRoot, ".test-workspace-link")
	if err := os.Symlink(tmp, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(link) })
	if _, err := resolveWorkspacePath(filepath.Join(link, "secret.txt")); err == nil {
		t.Fatal("expected symlink escape to be rejected")
	}
}

func TestResolveWorkspacePathAllowsNewFileInWorkspace(t *testing.T) {
	path := filepath.Join(workspaceRoot, ".test-new-dir", "file.txt")
	resolved, err := resolveWorkspacePath(path)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != path {
		t.Fatalf("got %q, want %q", resolved, path)
	}
}

func TestRunBashHonorsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := RunBash(ctx, "printf hello"); err == nil {
		t.Fatal("expected cancelled command to fail")
	}
}

func TestRunBashBlocksDangerousCommands(t *testing.T) {
	if _, err := RunBash(context.Background(), "rm -rf ./tmp"); err == nil {
		t.Fatal("expected dangerous command to be blocked")
	}
	if _, err := RunBash(context.Background(), "go test ./... | cat"); err == nil {
		t.Fatal("expected shell pipeline to be blocked")
	}
}
