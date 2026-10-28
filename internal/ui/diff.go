package ui

import (
	"strings"

	"github.com/aymanbagabas/go-udiff"
	"github.com/charmbracelet/lipgloss"
)

var (
	diffHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#89DCEB"))

	diffLineHunkStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#89B4FA")).
				Bold(true)

	diffLineDeleteStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F38BA8")).
				Background(lipgloss.Color("#311E28"))

	diffLineAddStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#A6E3A1")).
				Background(lipgloss.Color("#1E3128"))

	diffBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#74C7EC")).
			Padding(0, 1).
			MarginBottom(1)
)

func FormatDiffText(diffText string) string {
	lines := strings.Split(diffText, "\n")
	var styledLines []string

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "diff "):
			styledLines = append(styledLines, diffHeaderStyle.Render(line))
		case strings.HasPrefix(line, "@@"):
			styledLines = append(styledLines, diffLineHunkStyle.Render(line))
		case strings.HasPrefix(line, "-"):
			styledLines = append(styledLines, diffLineDeleteStyle.Render(line))
		case strings.HasPrefix(line, "+"):
			styledLines = append(styledLines, diffLineAddStyle.Render(line))
		default:
			styledLines = append(styledLines, lipgloss.NewStyle().Foreground(lipgloss.Color("#CDD6F4")).Render(line))
		}
	}
	return strings.Join(styledLines, "\n")
}

func RenderCodeChange(filePath, target, replacement string, width int) string {
	rawDiff := udiff.Unified("a/"+filePath, "b/"+filePath, target, replacement)
	if strings.TrimSpace(rawDiff) == "" {
		var sb strings.Builder
		sb.WriteString("--- a/" + filePath + "\n")
		sb.WriteString("+++ b/" + filePath + "\n")
		for _, l := range strings.Split(target, "\n") {
			sb.WriteString("-" + l + "\n")
		}
		for _, l := range strings.Split(replacement, "\n") {
			sb.WriteString("+" + l + "\n")
		}
		rawDiff = sb.String()
	}

	colored := FormatDiffText(rawDiff)
	box := diffBoxStyle.Width(width)
	return box.Render(colored)
}
