package tools

import (
	"context"
	"path/filepath"
	"testing"
)

func TestResolveWorkspacePathRejectsOutsidePath(t *testing.T) {
	if _, err := resolveWorkspacePath(filepath.Join(workspaceRoot, "..", "outside")); err == nil {
		t.Fatal("expected path outside workspace to be rejected")
	}
}

func TestRunBashHonorsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := RunBash(ctx, "printf hello"); err == nil {
		t.Fatal("expected cancelled command to fail")
	}
}
