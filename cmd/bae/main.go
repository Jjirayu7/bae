package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"bae/internal/agent"
	"bae/internal/tools"
	"bae/internal/ui"
)

const Version = "0.3.0"

func main() {
	modelFlag := flag.String("model", "", "Gemini model to use (leave empty for default)")
	providerFlag := flag.String("provider", "", "Provider to use: 'agy' (default, uses local Antigravity OAuth) or 'gemini' (uses GEMINI_API_KEY)")
	promptFlag := flag.String("p", "", "Run in one-shot mode with a prompt and exit")
	classicFlag := flag.Bool("classic", false, "Use simple command-line prompt instead of full-screen Chat UI")
	flag.Parse()

	ctx := context.Background()

	ag, err := agent.NewAgent(ctx, *providerFlag, *modelFlag)
	if err != nil {
		ui.PrintError(err)
		os.Exit(1)
	}

	if *promptFlag != "" {
		reply, err := ag.Send(*promptFlag)
		if err != nil {
			ui.PrintError(err)
			os.Exit(1)
		}
		ui.PrintAgentResponse(reply)
		return
	}

	if !*classicFlag {
		if err := ui.RunChatUI(ag); err != nil {
			ui.PrintError(err)
			os.Exit(1)
		}
		return
	}

	ui.PrintBanner(Version)
	fmt.Printf("%s\n\n", ui.DimStyle.Render("Provider: "+ag.ProviderName()))

	scanner := bufio.NewScanner(os.Stdin)

	for {
		ui.PrintUserPrompt()
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		switch strings.ToLower(input) {
		case "/exit", "/quit", "exit", "quit":
			fmt.Println("Goodbye!")
			return
		case "/clear", "clear":
			fmt.Print("\033[H\033[2J")
			ui.PrintBanner(Version)
			fmt.Printf("%s\n\n", ui.DimStyle.Render("Provider: "+ag.ProviderName()))
			continue
		case "/provider":
			fmt.Printf("\nActive Provider: %s\n\n", ag.ProviderName())
			continue
		case "/tools":
			fmt.Println("\nAvailable Tools:")
			for _, t := range tools.GetToolDeclarations() {
				fmt.Printf(" - %s: %s\n", t.Name, t.Description)
			}
			fmt.Println()
			continue
		case "/help":
			fmt.Println("\nCommands:")
			fmt.Println("  /provider - Show active LLM provider")
			fmt.Println("  /tools    - List all registered tools")
			fmt.Println("  /clear    - Clear the terminal screen")
			fmt.Println("  /exit     - Exit Bae Agent")
			fmt.Println()
			continue
		}

		reply, err := ag.Send(input)
		if err != nil {
			ui.PrintError(err)
			continue
		}

		ui.PrintAgentResponse(reply)
		fmt.Println()
	}

	if err := scanner.Err(); err != nil {
		ui.PrintError(err)
	}
}
