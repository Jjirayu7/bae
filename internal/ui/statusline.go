package ui

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type StatusInfo struct {
	Model      string
	Provider   string
	State      string // "idle", "thinking", "working"
	ToolName   string
	Cwd        string
	GitBranch  string
	GitDirty   bool
	TurnCount  int
	TotalTurns int
}

// GetGitInfo returns the current git branch and dirty status
func GetGitInfo(dir string) (string, bool) {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", false
	}
	branch := strings.TrimSpace(out.String())
	if branch == "" {
		cmdHead := exec.Command("git", "rev-parse", "--short", "HEAD")
		cmdHead.Dir = dir
		var outHead bytes.Buffer
		cmdHead.Stdout = &outHead
		if err := cmdHead.Run(); err == nil {
			branch = strings.TrimSpace(outHead.String())
		}
	}
	if branch == "" {
		return "", false
	}

	cmdStatus := exec.Command("git", "status", "--porcelain")
	cmdStatus.Dir = dir
	var outStatus bytes.Buffer
	cmdStatus.Stdout = &outStatus
	isDirty := false
	if err := cmdStatus.Run(); err == nil {
		isDirty = len(strings.TrimSpace(outStatus.String())) > 0
	}

	return branch, isDirty
}

// RenderStatusline renders a clean ASCII terminal statusline
func RenderStatusline(info StatusInfo, width int) string {
	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("#45475A")).Render(" | ")

	// 1. Model Badge
	modelStr := info.Model
	if modelStr == "" {
		modelStr = "Gemini"
	}
	modelBadge := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00D1FF")).
		Render(fmt.Sprintf("[MODEL: %s]", modelStr))

	// 2. State Badge
	var stateBadge string
	switch strings.ToLower(info.State) {
	case "thinking":
		stateBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F9E2AF")).
			Render("[THINKING]")
	case "working", "tool":
		toolLabel := "WORKING"
		if info.ToolName != "" {
			toolLabel = "TOOL: " + info.ToolName
		}
		stateBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#CBA6F7")).
			Render(fmt.Sprintf("[%s]", toolLabel))
	default:
		stateBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A6E3A1")).
			Render("[IDLE]")
	}

	// 3. Working Directory Badge
	cwd := info.Cwd
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(cwd, home) {
		cwd = "~" + cwd[len(home):]
	}
	parts := strings.Split(cwd, "/")
	if len(parts) > 4 {
		cwd = strings.Join(parts[:2], "/") + "/.../" + strings.Join(parts[len(parts)-2:], "/")
	}
	cwdBadge := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#89B4FA")).
		Render(fmt.Sprintf("[DIR: %s]", cwd))

	// 4. Git Badge
	var gitBadge string
	branch, dirty := GetGitInfo(info.Cwd)
	if branch != "" {
		dirtyMark := ""
		if dirty {
			dirtyMark = "*"
		}
		gitBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A6E3A1")).
			Render(fmt.Sprintf("[GIT: %s%s]", branch, dirtyMark))
	}

	// 5. Time Badge
	nowStr := time.Now().Format("15:04")
	timeBadge := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6C7086")).
		Render(fmt.Sprintf("[%s]", nowStr))

	var segments []string
	if modelBadge != "" {
		segments = append(segments, modelBadge)
	}
	if stateBadge != "" {
		segments = append(segments, stateBadge)
	}
	if cwdBadge != "" {
		segments = append(segments, cwdBadge)
	}
	if gitBadge != "" {
		segments = append(segments, gitBadge)
	}
	if timeBadge != "" {
		segments = append(segments, timeBadge)
	}

	line := strings.Join(segments, sep)
	return lipgloss.NewStyle().
		Background(lipgloss.Color("#181825")).
		Padding(0, 1).
		Width(width).
		Render(line)
}
