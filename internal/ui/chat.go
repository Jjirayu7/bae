package ui

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"bae/internal/agent"
)

type AgentSender interface {
	Send(userInput string) (string, error)
	SendStream(ctx context.Context, userInput string, onEvent agent.StreamHandler) (string, error)
	ProviderName() string
}

type MessageType int

const (
	MsgUser MessageType = iota
	MsgAgent
	MsgTool
	MsgDiff
	MsgSystem
	MsgError
)

type ChatEntry struct {
	Type    MessageType
	Sender  string
	Content string
}

type ChatModel struct {
	agent        AgentSender
	viewport     viewport.Model
	input        textinput.Model
	spinner      spinner.Model
	entries      []ChatEntry
	history      []string
	historyIdx   int
	statusInfo   StatusInfo
	isThinking   bool
	cancelPrompt context.CancelFunc
	ready        bool
	width        int
	height       int
	currDir      string
	eventChan    chan agent.StreamEvent
}

type streamEventMsg agent.StreamEvent

type responseMsg struct {
	text string
	err  error
}

func listenForEvents(sub chan agent.StreamEvent) tea.Cmd {
	return func() tea.Msg {
		if sub == nil {
			return nil
		}
		event, ok := <-sub
		if !ok {
			return nil
		}
		return streamEventMsg(event)
	}
}

func NewChatModel(agent AgentSender) ChatModel {
	ti := textinput.New()
	ti.Placeholder = "Ask Bae anything or type / (Tab to complete)..."
	ti.Focus()
	ti.Prompt = "> "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00D1FF")).Bold(true)
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#CDD6F4"))
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#585B70"))
	ti.CompletionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#585B70")).Italic(true)
	ti.CharLimit = 4000
	ti.Cursor.SetMode(cursor.CursorStatic)
	ti.ShowSuggestions = true
	ti.SetSuggestions(SlashCommands)

	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#00D1FF")).Bold(true)

	wd, _ := os.Getwd()
	displayWd := wd
	if len(displayWd) > 30 {
		displayWd = "..." + displayWd[len(displayWd)-27:]
	}

	initialEntries := []ChatEntry{
		{
			Type:    MsgSystem,
			Sender:  "System",
			Content: fmt.Sprintf("Bae Developer Agent online. [Engine: %s]\nTip: Mouse drag-selection (copy) is enabled. Use Up/Down arrows for history.", agent.ProviderName()),
		},
		{
			Type:    MsgAgent,
			Sender:  "Bae",
			Content: "Ready. Code changes, tool execution, and diffs will appear here.",
		},
	}

	return ChatModel{
		agent:      agent,
		input:      ti,
		spinner:    sp,
		entries:    initialEntries,
		history:    make([]string, 0),
		historyIdx: 0,
		currDir:    displayWd,
		statusInfo: StatusInfo{
			Model:    "Gemini",
			Provider: agent.ProviderName(),
			State:    "idle",
			Cwd:      wd,
		},
	}
}

func (m ChatModel) Init() tea.Cmd {
	// Do NOT start continuous spinner ticks when idle to prevent redrawing and dropping text selection
	return nil
}

func (m ChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		spCmd tea.Cmd
	)

	// Only update spinner when actually thinking
	if m.isThinking {
		m.spinner, spCmd = m.spinner.Update(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerHeight := 3
		footerHeight := 5
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width-2, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.input.Width = msg.Width - 6
			m.viewport.SetContent(m.renderConversation())
			m.ready = true
		} else {
			m.viewport.Width = msg.Width - 2
			m.viewport.Height = msg.Height - verticalMarginHeight
			m.input.Width = msg.Width - 6
			m.viewport.SetContent(m.renderConversation())
		}

	case streamEventMsg:
		switch msg.Type {
		case "tool_start":
			m.statusInfo.State = "working"
			m.statusInfo.ToolName = msg.Tool

			entryType := MsgTool
			if strings.Contains(msg.Tool, "edit_file") || strings.Contains(msg.Tool, "replace") {
				entryType = MsgDiff
			}

			m.entries = append(m.entries, ChatEntry{
				Type:    entryType,
				Sender:  msg.Tool,
				Content: msg.Args,
			})
			m.viewport.SetContent(m.renderConversation())
			m.viewport.GotoBottom()
			return m, listenForEvents(m.eventChan)

		case "tool_done":
			m.statusInfo.State = "thinking"
			m.statusInfo.ToolName = ""
			preview := strings.TrimSpace(msg.Output)

			if strings.Contains(preview, "--- a/") || strings.Contains(preview, "diff --git") {
				m.entries = append(m.entries, ChatEntry{
					Type:    MsgDiff,
					Sender:  msg.Tool + " [Diff]",
					Content: preview,
				})
			} else {
				lines := strings.Split(preview, "\n")
				if len(lines) > 5 {
					preview = strings.Join(lines[:5], "\n") + fmt.Sprintf("\n... (+%d more lines)", len(lines)-5)
				}
				if preview == "" {
					preview = "(done with no output)"
				}
				m.entries = append(m.entries, ChatEntry{
					Type:    MsgSystem,
					Sender:  msg.Tool,
					Content: fmt.Sprintf("[OK] Result:\n%s", preview),
				})
			}

			m.viewport.SetContent(m.renderConversation())
			m.viewport.GotoBottom()
			return m, listenForEvents(m.eventChan)
		}
		return m, listenForEvents(m.eventChan)

	case responseMsg:
		if !m.isThinking {
			return m, nil
		}
		m.isThinking = false
		m.cancelPrompt = nil
		m.statusInfo.State = "idle"
		m.statusInfo.ToolName = ""
		m.statusInfo.TurnCount++
		if msg.err != nil {
			if !strings.Contains(msg.err.Error(), "context canceled") && !strings.Contains(msg.err.Error(), "signal: killed") {
				m.entries = append(m.entries, ChatEntry{
					Type:    MsgError,
					Sender:  "Error",
					Content: msg.err.Error(),
				})
			}
		} else {
			m.entries = append(m.entries, ChatEntry{
				Type:    MsgAgent,
				Sender:  "Bae",
				Content: msg.text,
			})
		}
		m.viewport.SetContent(m.renderConversation())
		m.viewport.GotoBottom()
		m.input.Focus()

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit

		case tea.KeyEsc:
			if m.isThinking {
				if m.cancelPrompt != nil {
					m.cancelPrompt()
					m.cancelPrompt = nil
				}
				m.isThinking = false
				m.statusInfo.State = "idle"
				m.statusInfo.ToolName = ""
				m.entries = append(m.entries, ChatEntry{
					Type:    MsgSystem,
					Sender:  "System",
					Content: "[Prompt cancelled by user]",
				})
				m.viewport.SetContent(m.renderConversation())
				m.viewport.GotoBottom()
				m.input.Focus()
				return m, nil
			}

			// If not thinking, clear input line without quitting the session
			if m.input.Value() != "" {
				m.input.Reset()
			}
			return m, nil

		// Prompt History: Arrow Up
		case tea.KeyUp:
			if len(m.history) > 0 && m.historyIdx > 0 {
				m.historyIdx--
				m.input.SetValue(m.history[m.historyIdx])
				m.input.CursorEnd()
				return m, nil
			}

		// Prompt History: Arrow Down
		case tea.KeyDown:
			if len(m.history) > 0 && m.historyIdx < len(m.history)-1 {
				m.historyIdx++
				m.input.SetValue(m.history[m.historyIdx])
				m.input.CursorEnd()
				return m, nil
			} else if m.historyIdx == len(m.history)-1 {
				m.historyIdx = len(m.history)
				m.input.SetValue("")
				return m, nil
			}

		case tea.KeyEnter:
			if m.isThinking {
				return m, nil
			}
			inputVal := strings.TrimSpace(m.input.Value())
			if inputVal == "" {
				return m, nil
			}

			// Save to command history
			if len(m.history) == 0 || m.history[len(m.history)-1] != inputVal {
				m.history = append(m.history, inputVal)
			}
			m.historyIdx = len(m.history)

			// Intercept ALL slash commands locally - ZERO tokens used!
			if strings.HasPrefix(inputVal, "/") {
				m.input.Reset()
				stay, cmd := HandleCommand(inputVal, &m)
				m.viewport.SetContent(m.renderConversation())
				m.viewport.GotoBottom()
				if !stay {
					return m, tea.Quit
				}
				return m, cmd
			}

			// Add user message to conversation
			m.entries = append(m.entries, ChatEntry{
				Type:    MsgUser,
				Sender:  "You",
				Content: inputVal,
			})
			m.viewport.SetContent(m.renderConversation())
			m.viewport.GotoBottom()
			m.input.Reset()
			m.isThinking = true
			m.statusInfo.State = "thinking"

			// Create cancellable context
			ctx, cancel := context.WithCancel(context.Background())
			m.cancelPrompt = cancel

			// Create live stream channel
			eventChan := make(chan agent.StreamEvent, 50)
			m.eventChan = eventChan

			return m, tea.Batch(
				m.spinner.Tick,
				listenForEvents(eventChan),
				func() tea.Msg {
					reply, err := m.agent.SendStream(ctx, inputVal, func(ev agent.StreamEvent) {
						eventChan <- ev
					})
					close(eventChan)
					return responseMsg{text: reply, err: err}
				},
			)
		}
	}

	m.input, tiCmd = m.input.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	return m, tea.Batch(tiCmd, vpCmd, spCmd)
}

func (m ChatModel) renderConversation() string {
	var b strings.Builder
	contentWidth := m.width - 6
	if contentWidth < 40 {
		contentWidth = 70
	}

	userBadge := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#CDD6F4")).
		Background(lipgloss.Color("#45475A")).
		Padding(0, 1).
		Render("[YOU]")

	baeBadge := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#11111B")).
		Background(lipgloss.Color("#89DCEB")).
		Padding(0, 1).
		Render("[BAE]")

	diffBadge := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#11111B")).
		Background(lipgloss.Color("#74C7EC")).
		Padding(0, 1)

	toolBadge := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#11111B")).
		Background(lipgloss.Color("#F38BA8")).
		Padding(0, 1)

	systemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A6ADC8"))

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F38BA8")).
		Bold(true)

	for _, entry := range m.entries {
		switch entry.Type {
		case MsgUser:
			b.WriteString(userBadge + "\n" + entry.Content + "\n\n")

		case MsgAgent:
			b.WriteString(baeBadge + "\n" + entry.Content + "\n\n")

		case MsgDiff:
			b.WriteString(diffBadge.Render("[DIFF: "+entry.Sender+"]") + "\n" + FormatDiffText(entry.Content) + "\n\n")

		case MsgTool:
			b.WriteString(toolBadge.Render("[TOOL: "+entry.Sender+"]") + " " + entry.Content + "\n\n")

		case MsgSystem:
			b.WriteString(systemStyle.Render("[SYS] "+entry.Content) + "\n\n")

		case MsgError:
			b.WriteString(errorStyle.Render("[ERR] "+entry.Content) + "\n\n")
		}
	}

	return b.String()
}

func (m ChatModel) View() string {
	if !m.ready {
		return "\n  Initializing Bae..."
	}

	// 1. Header Bar
	headerLeft := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00D1FF")).
		Render("[BAE AGENT]")

	headerMid := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A6ADC8")).
		Render(fmt.Sprintf(" [%s]", m.agent.ProviderName()))

	headerRight := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6C7086")).
		Render(fmt.Sprintf(" [DIR: %s]", m.currDir))

	header := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(lipgloss.Color("#313244")).
		PaddingBottom(0).
		Width(m.width).
		Render(fmt.Sprintf("%s%s  %s", headerLeft, headerMid, headerRight))

	// 2. Chatbox Input (Bordered ONLY for the input box)
	inputBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#45475A")).
		Width(m.width - 2).
		Padding(0, 1).
		Render(m.input.View())

	// 3. Command Auto-Suggestions Palette or Thinking Spinner
	var suggestionPalette string
	if !m.isThinking && strings.HasPrefix(m.input.Value(), "/") {
		suggestionPalette = RenderCommandSuggestions(m.input.Value(), m.width)
	}

	var hint string
	if m.isThinking {
		actionText := "Thinking..."
		if m.statusInfo.ToolName != "" {
			actionText = fmt.Sprintf("Running tool: %s", m.statusInfo.ToolName)
		}
		hint = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#89B4FA")).
			Render(fmt.Sprintf(" %s %s", m.spinner.View(), actionText))
	}

	// 4. Statusline Bar
	statusline := RenderStatusline(m.statusInfo, m.width)

	var midAlert string
	if suggestionPalette != "" {
		midAlert = suggestionPalette + "\n"
	} else if hint != "" {
		midAlert = hint + "\n"
	}

	return fmt.Sprintf("%s\n%s\n%s%s\n%s",
		header,
		m.viewport.View(),
		midAlert,
		inputBox,
		statusline,
	)
}

// RunChatUI starts the fullscreen Bubble Tea Chat UI
func RunChatUI(agent AgentSender) error {
	p := tea.NewProgram(NewChatModel(agent), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
