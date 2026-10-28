package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var SlashCommands = []string{
	"/skill",
	"/learn",
	"/remember",
	"/rewind",
	"/diff",
	"/status",
	"/compact",
	"/tools",
	"/doctor",
	"/clear",
	"/help",
	"/exit",
}

func FindSkills() map[string]string {
	skills := make(map[string]string)
	home, _ := os.UserHomeDir()
	dirs := []string{
		filepath.Join(home, ".agents", "skills"),
		filepath.Join(home, ".gemini", "config", "skills"),
		filepath.Join(home, ".gemini", "antigravity-cli", "builtin", "skills"),
	}

	for _, d := range dirs {
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				skillMD := filepath.Join(d, e.Name(), "SKILL.md")
				if _, err := os.Stat(skillMD); err == nil {
					skills[e.Name()] = skillMD
				}
			}
		}
	}
	return skills
}

func MemoryFilePath() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".gemini", "bae")
	_ = os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "preferences.md")
}

func HandleCommand(input string, m *ChatModel) (bool, tea.Cmd) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return true, nil
	}
	cmd := strings.ToLower(parts[0])
	args := strings.TrimSpace(strings.TrimPrefix(input, parts[0]))

	switch cmd {
	case "/skill":
		skills := FindSkills()
		if args == "" {
			var list []string
			for name := range skills {
				list = append(list, name)
			}
			sort.Strings(list)
			msg := fmt.Sprintf("Available Skills (%d found):\n  Use '/skill <name>' to activate.\n\n  %s", len(list), strings.Join(list, ", "))
			m.entries = append(m.entries, ChatEntry{
				Type:    MsgSystem,
				Sender:  "Skills",
				Content: msg,
			})
			return true, nil
		}

		skillMD, found := skills[args]
		if !found {
			m.entries = append(m.entries, ChatEntry{
				Type:    MsgError,
				Sender:  "Skill",
				Content: fmt.Sprintf("Skill '%s' not found. Type '/skill' to see available skills.", args),
			})
			return true, nil
		}

		content, err := os.ReadFile(skillMD)
		if err != nil {
			m.entries = append(m.entries, ChatEntry{
				Type:    MsgError,
				Sender:  "Skill",
				Content: fmt.Sprintf("Failed to read skill: %v", err),
			})
			return true, nil
		}
		m.agent.AddContext(string(content))
		m.entries = append(m.entries, ChatEntry{
			Type:    MsgSystem,
			Sender:  "Skill: " + args,
			Content: fmt.Sprintf("[OK] Skill '%s' loaded into active session from %s\nGuidelines active.", args, skillMD),
		})
		_ = content
		return true, nil

	case "/learn":
		if args == "" {
			m.entries = append(m.entries, ChatEntry{
				Type:    MsgSystem,
				Sender:  "Learn",
				Content: "Usage: /learn <preference or convention>\nExample: /learn Always write table tests in Go",
			})
			return true, nil
		}
		memPath := MemoryFilePath()
		entry := fmt.Sprintf("- %s\n", args)
		f, err := os.OpenFile(memPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			m.entries = append(m.entries, ChatEntry{Type: MsgError, Sender: "Learn", Content: err.Error()})
			return true, nil
		}
		defer f.Close()
		_, _ = f.WriteString(entry)

		m.entries = append(m.entries, ChatEntry{
			Type:    MsgSystem,
			Sender:  "Memory",
			Content: fmt.Sprintf("[OK] Learned and saved to persistent memory:\n  \"%s\"\n(Stored in %s)", args, memPath),
		})
		return true, nil

	case "/remember":
		memPath := MemoryFilePath()
		b, err := os.ReadFile(memPath)
		memContent := string(b)
		if err != nil || strings.TrimSpace(memContent) == "" {
			memContent = "(no memories saved yet. Use /learn <rule> to save preferences)"
		}

		if args != "" {
			var matches []string
			for _, line := range strings.Split(memContent, "\n") {
				if strings.Contains(strings.ToLower(line), strings.ToLower(args)) {
					matches = append(matches, line)
				}
			}
			if len(matches) > 0 {
				memContent = strings.Join(matches, "\n")
			} else {
				memContent = fmt.Sprintf("No memories matching '%s'", args)
			}
		}

		m.entries = append(m.entries, ChatEntry{
			Type:    MsgSystem,
			Sender:  "Remember",
			Content: fmt.Sprintf("Known Preferences & Memories:\n%s", memContent),
		})
		return true, nil

	case "/rewind", "/undo":
		if len(m.entries) >= 2 {
			m.entries = m.entries[:len(m.entries)-2]
			m.agent.ResetConversation()
			m.entries = append(m.entries, ChatEntry{
				Type:    MsgSystem,
				Sender:  "Rewind",
				Content: "[OK] Rewound conversation by 1 turn. Previous prompt & response removed.",
			})
		} else {
			m.entries = append(m.entries, ChatEntry{
				Type:    MsgSystem,
				Sender:  "Rewind",
				Content: "Nothing to rewind.",
			})
		}
		return true, nil

	case "/compact":
		if len(m.entries) > 4 {
			kept := m.entries[len(m.entries)-3:]
			m.agent.ResetConversation()
			m.entries = append([]ChatEntry{
				{
					Type:    MsgSystem,
					Sender:  "Compact",
					Content: "[OK] Compacted chat history. Older turns cleared to free context tokens.",
				},
			}, kept...)
		} else {
			m.entries = append(m.entries, ChatEntry{
				Type:    MsgSystem,
				Sender:  "Compact",
				Content: "Chat history already minimal.",
			})
		}
		return true, nil

	case "/doctor":
		var sb strings.Builder
		sb.WriteString("Ecosystem Diagnostic Check:\n")
		checkTool := func(name, desc string) {
			if _, err := exec.LookPath(name); err == nil {
				sb.WriteString(fmt.Sprintf("  ✓ %s: installed (%s)\n", name, desc))
			} else {
				sb.WriteString(fmt.Sprintf("  - %s: not installed (%s)\n", name, desc))
			}
		}
		checkTool("go", "Compiler")
		checkTool("rtk", "Rust Token Killer proxy")
		checkTool("ov", "OpenViking context database")
		checkTool("ccc", "CocoIndex semantic code search")
		checkTool("graphify", "AST knowledge graph")
		checkTool("agy", "Antigravity OAuth engine")

		m.entries = append(m.entries, ChatEntry{
			Type:    MsgSystem,
			Sender:  "Doctor",
			Content: sb.String(),
		})
		return true, nil

	case "/diff":
		out, err := exec.Command("git", "diff").CombinedOutput()
		diffStr := strings.TrimSpace(string(out))
		if err != nil || diffStr == "" {
			diffStr = "(no unstaged git changes)"
		}
		m.entries = append(m.entries, ChatEntry{
			Type:    MsgDiff,
			Sender:  "git diff",
			Content: diffStr,
		})
		return true, nil

	case "/status":
		out, err := exec.Command("git", "status", "--short").CombinedOutput()
		statusStr := strings.TrimSpace(string(out))
		if err != nil || statusStr == "" {
			statusStr = "Working tree clean."
		}
		m.entries = append(m.entries, ChatEntry{
			Type:    MsgSystem,
			Sender:  "git status",
			Content: statusStr,
		})
		return true, nil

	case "/tools":
		m.entries = append(m.entries, ChatEntry{
			Type:    MsgSystem,
			Sender:  "Tools",
			Content: "Available Tools:\n  - run_bash    : Execute terminal shell commands (RTK-optimized)\n  - edit_file   : Exact block replacement/patching\n  - read_file   : Read file content from disk\n  - write_file  : Create/overwrite files\n  - search_code : Fast ripgrep/grep search across code\n  - ccc_search  : CocoIndex semantic codebase search\n  - ov_find     : OpenViking knowledge base search\n  - graphify    : Graphify AST architectural queries",
		})
		return true, nil

	case "/clear":
		m.entries = nil
		return true, nil

	case "/help":
		helpMsg := `Bae Commands (0 tokens used):
  /skill <name>   - List or activate skills (e.g. /skill tdd)
  /learn <rule>   - Save developer preferences to persistent memory
  /remember [q]   - Search or review persistent memory
  /rewind         - Undo / rewind the last conversation turn
  /compact        - Free context tokens by compacting older history
  /diff           - View unstaged git code changes
  /status         - View git status summary
  /tools          - List available agent tools
  /doctor         - Run ecosystem tool diagnosis
  /clear          - Clear screen history
  /exit           - Exit session (or Ctrl+C)

Shortcuts:
  Tab             - Autocomplete slash command
  Up / Down       - Cycle previous prompt history
  Esc             - Cancel running prompt (or clear draft)`
		m.entries = append(m.entries, ChatEntry{
			Type:    MsgSystem,
			Sender:  "Help",
			Content: helpMsg,
		})
		return true, nil

	case "/exit", "/quit":
		return false, tea.Quit

	default:
		m.entries = append(m.entries, ChatEntry{
			Type:    MsgSystem,
			Sender:  "System",
			Content: fmt.Sprintf("Unknown command '%s'. Type /help for available commands. (No tokens consumed)", input),
		})
		return true, nil
	}
}

func RenderCommandSuggestions(input string, width int) string {
	if !strings.HasPrefix(input, "/") {
		return ""
	}

	var matches []string
	lowerInput := strings.ToLower(input)
	for _, cmd := range SlashCommands {
		if strings.HasPrefix(cmd, lowerInput) {
			matches = append(matches, cmd)
		}
	}

	if len(matches) == 0 {
		return ""
	}

	palette := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#89DCEB")).
		Bold(true).
		Render("  [Tab to complete]: " + strings.Join(matches, "  "))

	return palette
}
