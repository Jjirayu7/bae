package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"bae/internal/tools"

	"google.golang.org/genai"
)

type StreamEvent struct {
	Type   string // "tool_start", "tool_done", "response", "error"
	Tool   string
	Args   string
	Output string
	Text   string
}

type StreamHandler func(event StreamEvent)

type Provider interface {
	Send(userInput string) (string, error)
	SendStream(ctx context.Context, userInput string, onEvent StreamHandler) (string, error)
	Name() string
}

type Agent struct {
	provider Provider
	context  string
}

func NewAgent(ctx context.Context, preferredProvider string, modelName string) (*Agent, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")

	if preferredProvider == "gemini" || (preferredProvider == "" && apiKey != "") {
		if apiKey == "" {
			return nil, fmt.Errorf("GEMINI_API_KEY is required for the 'gemini' provider. Export it or switch to 'agy'")
		}
		p, err := NewGeminiProvider(ctx, modelName, apiKey)
		if err != nil {
			return nil, err
		}
		return &Agent{provider: p}, nil
	}

	agyPath, err := exec.LookPath("agy")
	if err == nil && agyPath != "" {
		p := NewAgyProvider(agyPath, modelName)
		return &Agent{provider: p}, nil
	}

	fallbackPaths := []string{
		os.ExpandEnv("$HOME/.local/bin/agy"),
		os.ExpandEnv("$HOME/.gemini/bin/agy"),
		"/usr/local/bin/agy",
	}
	for _, path := range fallbackPaths {
		if _, err := os.Stat(path); err == nil {
			p := NewAgyProvider(path, modelName)
			return &Agent{provider: p}, nil
		}
	}

	return nil, fmt.Errorf("no authentication provider found.\nEither:\n  1. Install / login to Antigravity CLI ('agy')\n  2. Or export GEMINI_API_KEY='your-key'")
}

func (a *Agent) Send(userInput string) (string, error) {
	return a.SendStream(context.Background(), userInput, nil)
}

func (a *Agent) SendStream(ctx context.Context, userInput string, onEvent StreamHandler) (string, error) {
	if a.context != "" {
		userInput = a.context + "\n\nUser Request:\n" + userInput
	}
	return a.provider.SendStream(ctx, userInput, onEvent)
}

// AddContext adds durable instructions to the current agent session.
func (a *Agent) AddContext(content string) {
	if strings.TrimSpace(content) == "" {
		return
	}
	if a.context == "" {
		a.context = "Active session context:\n" + content
		return
	}
	a.context += "\n\n" + content
}

// ResetConversation drops provider-side conversation state and injected context.
func (a *Agent) ResetConversation() {
	a.context = ""
	if p, ok := a.provider.(*AgyProvider); ok {
		p.conversationID = ""
	}
}

func (a *Agent) ProviderName() string {
	return a.provider.Name()
}

const SystemPrompt = `You are Bae Agent, an ultra-efficient autonomous developer agent.

# Caveman Mode Rules:
- Drop fluff: Eliminate pleasantries, filler words, and conversational preambles.
- Style: Use terse, direct fragments and short synonyms. Pattern: [thing] [action] [reason]. [next step].
- Language preservation: Preserve user's language (Thai/English).
- Technical precision: Keep code blocks, file paths, symbol names, line numbers, terminal commands 100% exact.

# Ponytail Principles:
- Zero-bloat, strict minimalism. Reuse existing tokens and components. No redundant code.

# Tool Usage Strategy:
- RTK: All shell commands are automatically token-optimized with rtk.
- CocoIndex (ccc_search): First pass for broad codebase exploration and semantic code lookup.
- Graphify (graphify_explain, graphify_path): Architecture dependency graph & AST symbol queries.
- OpenViking (ov_find, ov_search, ov_read): Persistent context database & cross-session memory retrieval.
- Edit File (edit_file): Make exact targeted block replacements without rewriting whole files.`

// -------------------------------------------------------------
// Provider 1: Antigravity CLI Provider with Live NDJSON Streaming
// -------------------------------------------------------------
type AgyProvider struct {
	binaryPath     string
	modelName      string
	conversationID string
}

func NewAgyProvider(binaryPath, modelName string) *AgyProvider {
	return &AgyProvider{
		binaryPath: binaryPath,
		modelName:  modelName,
	}
}

func (p *AgyProvider) Name() string {
	return "Antigravity Provider (OAuth - No API Key required)"
}

func (p *AgyProvider) Send(userInput string) (string, error) {
	return p.SendStream(context.Background(), userInput, nil)
}

func (p *AgyProvider) SendStream(ctx context.Context, userInput string, onEvent StreamHandler) (string, error) {
	args := []string{"--output-format", "stream-json", "--dangerously-skip-permissions"}

	prompt := userInput
	if p.conversationID == "" {
		prompt = fmt.Sprintf("%s\n\nUser Request:\n%s", SystemPrompt, userInput)
	}

	if p.conversationID != "" {
		args = append(args, "--conversation", p.conversationID)
	}
	if p.modelName != "" {
		args = append(args, "--model", p.modelName)
	}
	args = append(args, "-p", prompt)

	cmd := exec.CommandContext(ctx, p.binaryPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to open stdout pipe: %w", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start agy: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	var finalResponse string

	for scanner.Scan() {
		line := scanner.Bytes()
		var generic map[string]any
		if err := json.Unmarshal(line, &generic); err != nil {
			continue
		}

		event, _ := generic["event"].(string)

		if convID, ok := generic["conversation_id"].(string); ok && convID != "" {
			p.conversationID = convID
		}

		switch event {
		case "step_update":
			step, ok := generic["step_update"].(map[string]any)
			if !ok {
				continue
			}
			if convID, ok := step["conversation_id"].(string); ok && convID != "" {
				p.conversationID = convID
			}

			stepType, _ := step["step_type"].(string)
			state, _ := step["state"].(string)
			toolName, _ := step["tool_name"].(string)

			if stepType == "tool" {
				toolInfo, _ := step["tool_info"].(map[string]any)
				if state == "ACTIVE" {
					var argsStr string
					if params, ok := toolInfo["parameters"].(map[string]any); ok {
						var parts []string
						for k, v := range params {
							parts = append(parts, fmt.Sprintf("%s=%v", k, v))
						}
						argsStr = strings.Join(parts, " ")
					}
					if onEvent != nil {
						onEvent(StreamEvent{
							Type: "tool_start",
							Tool: toolName,
							Args: argsStr,
						})
					}
				} else if state == "DONE" || state == "ERROR" {
					outStr := ""
					if o, ok := toolInfo["output"].(string); ok {
						outStr = o
					} else if errMap, ok := toolInfo["error"].(map[string]any); ok {
						outStr = fmt.Sprintf("Error: %v", errMap["message"])
					}
					if onEvent != nil {
						onEvent(StreamEvent{
							Type:   "tool_done",
							Tool:   toolName,
							Output: outStr,
						})
					}
				}
			}

		case "result":
			resObj, ok := generic["result"].(map[string]any)
			if ok {
				if convID, ok := resObj["conversation_id"].(string); ok && convID != "" {
					p.conversationID = convID
				}
				if r, ok := resObj["response"].(string); ok {
					finalResponse = r
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		_ = cmd.Wait()
		return "", fmt.Errorf("failed reading agy output: %w", err)
	}

	if err := cmd.Wait(); err != nil && strings.TrimSpace(finalResponse) == "" {
		return "", fmt.Errorf("agy exited with error: %w\n%s", err, stderr.String())
	}

	if strings.TrimSpace(finalResponse) == "" && stderr.Len() > 0 {
		return "", fmt.Errorf("agy provider error: %s", stderr.String())
	}

	return strings.TrimRight(finalResponse, "\n"), nil
}

// -------------------------------------------------------------
// Provider 2: Direct Gemini API Provider (via google.golang.org/genai)
// -------------------------------------------------------------
type GeminiProvider struct {
	client *genai.Client
	chat   *genai.Chat
	ctx    context.Context
}

func NewGeminiProvider(ctx context.Context, modelName, apiKey string) (*GeminiProvider, error) {
	if modelName == "" {
		modelName = "gemini-2.5-flash"
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	config := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{
				genai.NewPartFromText(SystemPrompt),
			},
		},
		Tools: []*genai.Tool{
			{
				FunctionDeclarations: tools.GetToolDeclarations(),
			},
		},
	}

	chat, err := client.Chats.Create(ctx, modelName, config, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat session: %w", err)
	}

	return &GeminiProvider{
		client: client,
		chat:   chat,
		ctx:    ctx,
	}, nil
}

func (p *GeminiProvider) Name() string {
	return "Gemini API Provider (GEMINI_API_KEY)"
}

func (p *GeminiProvider) Send(userInput string) (string, error) {
	return p.SendStream(context.Background(), userInput, nil)
}

func (p *GeminiProvider) SendStream(ctx context.Context, userInput string, onEvent StreamHandler) (string, error) {
	resp, err := p.chat.SendMessage(ctx, *genai.NewPartFromText(userInput))
	if err != nil {
		return "", err
	}

	maxTurns := 15
	turn := 0

	for turn < maxTurns {
		turn++
		fnCalls := resp.FunctionCalls()
		if len(fnCalls) == 0 {
			return resp.Text(), nil
		}

		var toolResponses []genai.Part
		for _, call := range fnCalls {
			var argStrings []string
			for k, v := range call.Args {
				argStrings = append(argStrings, fmt.Sprintf("%s=%v", k, v))
			}
			argsJoined := strings.Join(argStrings, " ")

			if onEvent != nil {
				onEvent(StreamEvent{Type: "tool_start", Tool: call.Name, Args: argsJoined})
			}

			out, toolErr := tools.ExecuteTool(ctx, call.Name, call.Args)

			if onEvent != nil {
				outDisplay := out
				if toolErr != nil {
					outDisplay = toolErr.Error()
				}
				onEvent(StreamEvent{Type: "tool_done", Tool: call.Name, Output: outDisplay})
			}

			respMap := map[string]any{"output": out}
			if toolErr != nil {
				respMap["error"] = toolErr.Error()
			}

			part := genai.NewPartFromFunctionResponse(call.Name, respMap)
			toolResponses = append(toolResponses, *part)
		}

		resp, err = p.chat.SendMessage(ctx, toolResponses...)
		if err != nil {
			return "", fmt.Errorf("error sending tool response to Gemini: %w", err)
		}
	}

	return resp.Text(), nil
}
