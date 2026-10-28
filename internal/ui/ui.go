package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	primaryColor   = lipgloss.Color("#00D1FF")
	secondaryColor = lipgloss.Color("#7D56F4")
	toolColor      = lipgloss.Color("#F38BA8")
	successColor   = lipgloss.Color("#A6E3A1")
	grayColor      = lipgloss.Color("#6C7086")

	BannerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(secondaryColor).
			Padding(0, 2).
			MarginBottom(1)

	UserPromptStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(secondaryColor)

	AgentBadgeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor)

	ToolBadgeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(toolColor)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(successColor)

	DimStyle = lipgloss.NewStyle().
			Foreground(grayColor)

	ErrorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF5555"))
)

func PrintBanner(version string) {
	content := fmt.Sprintf("BAE Agent CLI (v%s)\nAutonomous Developer Agent powered by Gemini", version)
	fmt.Println(BannerStyle.Render(content))
	fmt.Println(DimStyle.Render("Commands: /exit to quit | /clear to clear screen | /tools to see tools"))
	fmt.Println()
}

func PrintUserPrompt() {
	fmt.Print(UserPromptStyle.Render("user > "))
}

func PrintAgentResponse(text string) {
	fmt.Println()
	fmt.Println(AgentBadgeStyle.Render("bae > ") + text)
}

func PrintToolCall(name string, args map[string]any) {
	var argStrings []string
	for k, v := range args {
		argStrings = append(argStrings, fmt.Sprintf("%s=%v", k, v))
	}
	fmt.Printf("\n%s %s(%s)\n",
		ToolBadgeStyle.Render("[TOOL CALL]"),
		lipgloss.NewStyle().Bold(true).Render(name),
		DimStyle.Render(strings.Join(argStrings, ", ")),
	)
}

func PrintToolResult(output string, err error) {
	if err != nil {
		fmt.Printf("%s %v\n", ErrorStyle.Render("  [ERROR] Tool Error:"), err)
		return
	}
	lines := strings.Split(output, "\n")
	limit := 8
	if len(lines) > limit {
		preview := strings.Join(lines[:limit], "\n")
		fmt.Printf("%s\n%s\n%s\n",
			SuccessStyle.Render("  Result:"),
			DimStyle.Render(preview),
			DimStyle.Render(fmt.Sprintf("  ... (+%d more lines)", len(lines)-limit)),
		)
	} else {
		fmt.Printf("%s\n%s\n",
			SuccessStyle.Render("  Result:"),
			DimStyle.Render(output),
		)
	}
}

func PrintError(err error) {
	fmt.Println(ErrorStyle.Render(fmt.Sprintf("\n[Error]: %v\n", err)))
}
