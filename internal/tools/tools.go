package tools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"google.golang.org/genai"
)

func GetToolDeclarations() []*genai.FunctionDeclaration {
	return []*genai.FunctionDeclaration{
		{
			Name:        "run_bash",
			Description: "Execute a shell command in the terminal. Automatically optimized with RTK proxy for token savings.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"command": {
						Type:        genai.TypeString,
						Description: "The bash command line to run.",
					},
				},
				Required: []string{"command"},
			},
		},
		{
			Name:        "read_file",
			Description: "Read the entire content of a file from disk.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"path": {
						Type:        genai.TypeString,
						Description: "The relative or absolute file path to read.",
					},
				},
				Required: []string{"path"},
			},
		},
		{
			Name:        "edit_file",
			Description: "Make targeted edits to an existing file by replacing an exact block of code (target) with a new block (replacement).",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"path": {
						Type:        genai.TypeString,
						Description: "The path of the file to edit.",
					},
					"target": {
						Type:        genai.TypeString,
						Description: "The exact character sequence to replace, including exact whitespace and indentation.",
					},
					"replacement": {
						Type:        genai.TypeString,
						Description: "The content to replace the target with.",
					},
				},
				Required: []string{"path", "target", "replacement"},
			},
		},
		{
			Name:        "write_file",
			Description: "Write content to a file. Overwrites or creates the file if it does not exist.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"path": {
						Type:        genai.TypeString,
						Description: "The target file path.",
					},
					"content": {
						Type:        genai.TypeString,
						Description: "The content to write into the file.",
					},
				},
				Required: []string{"path", "content"},
			},
		},
		{
			Name:        "search_code",
			Description: "Fast regex/string search across codebase files using ripgrep/grep.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"query": {
						Type:        genai.TypeString,
						Description: "Search query or regex pattern.",
					},
					"path": {
						Type:        genai.TypeString,
						Description: "Directory or file path to search in (defaults to '.').",
					},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "list_dir",
			Description: "List files and directories in a given path.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"path": {
						Type:        genai.TypeString,
						Description: "The directory path to list (defaults to current dir '.').",
					},
				},
			},
		},
		{
			Name:        "ov_find",
			Description: "OpenViking semantic search in persistent knowledge base and memories.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"query": {
						Type:        genai.TypeString,
						Description: "Search query for knowledge base.",
					},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "ov_search",
			Description: "OpenViking context-aware retrieval across cross-session memory.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"query": {
						Type:        genai.TypeString,
						Description: "Context query to search for past decisions and preferences.",
					},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "ov_read",
			Description: "OpenViking read file or memory content from a viking:// URI.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"uri": {
						Type:        genai.TypeString,
						Description: "The URI to read (e.g. 'viking://user/default/memories/preferences.md').",
					},
				},
				Required: []string{"uri"},
			},
		},
		{
			Name:        "ccc_search",
			Description: "CocoIndex Code fast semantic codebase search across large repositories.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"query": {
						Type:        genai.TypeString,
						Description: "Natural language or semantic code search query.",
					},
					"path": {
						Type:        genai.TypeString,
						Description: "Optional specific directory to search.",
					},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "graphify_explain",
			Description: "Graphify explain a symbol/node and its connections in the architecture dependency graph.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"symbol": {
						Type:        genai.TypeString,
						Description: "The symbol, function, or class name to explain.",
					},
				},
				Required: []string{"symbol"},
			},
		},
		{
			Name:        "graphify_path",
			Description: "Graphify find the shortest dependency path between two symbols/nodes.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"from": {
						Type:        genai.TypeString,
						Description: "Starting node / symbol name.",
					},
					"to": {
						Type:        genai.TypeString,
						Description: "Target node / symbol name.",
					},
				},
				Required: []string{"from", "to"},
			},
		},
	}
}

func ExecuteTool(ctx context.Context, name string, args map[string]any) (string, error) {
	switch name {
	case "run_bash":
		cmdStr, ok := args["command"].(string)
		if !ok || cmdStr == "" {
			return "", fmt.Errorf("missing or invalid 'command' parameter")
		}
		return RunBash(ctx, cmdStr)

	case "read_file":
		path, ok := args["path"].(string)
		if !ok || path == "" {
			return "", fmt.Errorf("missing or invalid 'path' parameter")
		}
		return ReadFile(path)

	case "edit_file":
		path, ok := args["path"].(string)
		target, okTarget := args["target"].(string)
		replacement, okRepl := args["replacement"].(string)
		if !ok || !okTarget || !okRepl {
			return "", fmt.Errorf("missing 'path', 'target', or 'replacement' parameter")
		}
		return EditFile(path, target, replacement)

	case "write_file":
		path, ok := args["path"].(string)
		content, okContent := args["content"].(string)
		if !ok || !okContent {
			return "", fmt.Errorf("missing 'path' or 'content' parameter")
		}
		return WriteFile(path, content)

	case "search_code":
		query, ok := args["query"].(string)
		if !ok || query == "" {
			return "", fmt.Errorf("missing or invalid 'query' parameter")
		}
		path := "."
		if p, ok := args["path"].(string); ok && p != "" {
			path = p
		}
		return SearchCode(ctx, query, path)

	case "list_dir":
		path := "."
		if p, ok := args["path"].(string); ok && p != "" {
			path = p
		}
		return ListDir(path)

	case "ov_find":
		query, _ := args["query"].(string)
		if _, err := exec.LookPath("ov"); err != nil {
			return fmt.Sprintf("[Notice]: OpenViking (ov) CLI is not installed on this system. You can install it via 'ov' or check viking:// docs."), nil
		}
		return RunBash(ctx, fmt.Sprintf("ov find %q", query))

	case "ov_search":
		query, _ := args["query"].(string)
		if _, err := exec.LookPath("ov"); err != nil {
			return fmt.Sprintf("[Notice]: OpenViking (ov) CLI is not installed on this system. You can install it via 'ov'."), nil
		}
		return RunBash(ctx, fmt.Sprintf("ov search %q", query))

	case "ov_read":
		uri, _ := args["uri"].(string)
		if _, err := exec.LookPath("ov"); err != nil {
			return fmt.Sprintf("[Notice]: OpenViking (ov) CLI is not installed on this system."), nil
		}
		return RunBash(ctx, fmt.Sprintf("ov read %q", uri))

	case "ccc_search":
		query, _ := args["query"].(string)
		path, _ := args["path"].(string)
		if _, err := exec.LookPath("ccc"); err != nil {
			searchDir := "."
			if path != "" {
				searchDir = path
			}
			return SearchCode(ctx, query, searchDir)
		}
		cmd := fmt.Sprintf("ccc search %q", query)
		if path != "" {
			cmd += fmt.Sprintf(" --path %q", path)
		}
		return RunBash(ctx, cmd)

	case "graphify_explain":
		symbol, _ := args["symbol"].(string)
		if _, err := exec.LookPath("graphify"); err != nil {
			return fmt.Sprintf("[Notice]: Graphify CLI is not installed on this system. Fallback to code search for symbol '%s'", symbol), nil
		}
		return RunBash(ctx, fmt.Sprintf("graphify explain %q", symbol))

	case "graphify_path":
		from, _ := args["from"].(string)
		to, _ := args["to"].(string)
		if _, err := exec.LookPath("graphify"); err != nil {
			return "[Notice]: Graphify CLI is not installed on this system.", nil
		}
		return RunBash(ctx, fmt.Sprintf("graphify path %q %q", from, to))

	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

func RunBash(ctx context.Context, command string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	cmdToRun := strings.TrimSpace(command)
	if _, err := exec.LookPath("rtk"); err == nil {
		if !strings.HasPrefix(cmdToRun, "rtk ") && !strings.HasPrefix(cmdToRun, "rtk\t") {
			cmdToRun = "rtk proxy " + cmdToRun
		}
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", cmdToRun)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	out := stdout.String()
	if stderr.Len() > 0 {
		if out != "" {
			out += "\n"
		}
		out += "[stderr]: " + stderr.String()
	}
	if err != nil {
		return out, fmt.Errorf("command exit error: %w\n%s", err, out)
	}
	if strings.TrimSpace(out) == "" {
		out = "(command executed with no output)"
	}
	return out, nil
}

func ReadFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func EditFile(path, target, replacement string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file '%s': %w", path, err)
	}
	content := string(b)

	count := strings.Count(content, target)
	if count == 0 {
		return "", fmt.Errorf("target string not found in '%s'. Make sure whitespace/indentation matches exactly", path)
	}
	if count > 1 {
		return "", fmt.Errorf("target string occurs %d times in '%s'; refusing ambiguous edit", count, path)
	}

	newContent := strings.Replace(content, target, replacement, 1)
	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("failed to save edits to '%s': %w", path, err)
	}

	return fmt.Sprintf("Successfully edited '%s' (replaced 1 occurrence, delta: %+d bytes)", path, len(replacement)-len(target)), nil
}

func WriteFile(path string, content string) (string, error) {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory: %w", err)
		}
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}
	return fmt.Sprintf("Successfully written %d bytes to %s", len(content), path), nil
}

func SearchCode(ctx context.Context, query, path string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if _, err := exec.LookPath("rg"); err == nil {
		cmd = exec.CommandContext(ctx, "rg", "-n", "--max-columns", "250", "-i", query, path)
	} else {
		cmd = exec.CommandContext(ctx, "grep", "-rnI", "-E", "-i", query, path)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	_ = cmd.Run()
	out := stdout.String()
	if strings.TrimSpace(out) == "" {
		return fmt.Sprintf("No matches found for '%s' in '%s'", query, path), nil
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) > 50 {
		preview := strings.Join(lines[:50], "\n")
		return fmt.Sprintf("%s\n... (truncated %d additional matches)", preview, len(lines)-50), nil
	}

	return out, nil
}

func ListDir(path string) (string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	var sb strings.Builder
	for _, entry := range entries {
		typeStr := "FILE"
		if entry.IsDir() {
			typeStr = "DIR "
		}
		sb.WriteString(fmt.Sprintf("%s  %s\n", typeStr, entry.Name()))
	}
	res := sb.String()
	if res == "" {
		res = "(empty directory)"
	}
	return res, nil
}
