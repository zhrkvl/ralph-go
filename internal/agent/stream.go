package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

// parseStreamJSON converts a single line of Claude's stream-json output
// into zero or more human-readable display lines for the TUI.
//
// Stream-json format (one JSON object per line):
//   {"type":"system","subtype":"init",...}
//   {"type":"assistant","message":{"content":[{"type":"text","text":"..."},{"type":"tool_use",...}]}}
//   {"type":"result","subtype":"success",...}
func parseStreamJSON(line string) []string {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	var event map[string]any
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		// Not JSON — return as-is (might be plain text from stderr)
		return []string{line}
	}

	typ, _ := event["type"].(string)

	switch typ {
	case "system":
		return parseSystemEvent(event)
	case "assistant":
		return parseAssistantEvent(event)
	case "result":
		return parseResultEvent(event)
	default:
		return nil
	}
}

func parseSystemEvent(event map[string]any) []string {
	subtype, _ := event["subtype"].(string)
	switch subtype {
	case "init":
		model, _ := event["model"].(string)
		if model != "" {
			return []string{fmt.Sprintf("[init] model=%s", model)}
		}
		return []string{"[init]"}
	case "hook_started":
		name, _ := event["hook_name"].(string)
		if name != "" {
			return []string{fmt.Sprintf("[hook] %s", name)}
		}
	case "hook_response":
		// Skip verbose hook output
	}
	return nil
}

func parseAssistantEvent(event map[string]any) []string {
	msg, ok := event["message"].(map[string]any)
	if !ok {
		return nil
	}

	content, ok := msg["content"].([]any)
	if !ok {
		return nil
	}

	var lines []string
	for _, item := range content {
		block, ok := item.(map[string]any)
		if !ok {
			continue
		}

		blockType, _ := block["type"].(string)
		switch blockType {
		case "text":
			text, _ := block["text"].(string)
			if text != "" {
				// Split multi-line text into separate lines
				for _, l := range strings.Split(text, "\n") {
					lines = append(lines, l)
				}
			}

		case "tool_use":
			name, _ := block["name"].(string)
			input, _ := block["input"].(map[string]any)
			lines = append(lines, formatToolUse(name, input))

		case "tool_result":
			// Show truncated tool result
			content := extractToolResultContent(block)
			if content != "" {
				// Just show first line, truncated
				first := strings.SplitN(content, "\n", 2)[0]
				if len(first) > 120 {
					first = first[:117] + "..."
				}
				lines = append(lines, fmt.Sprintf("  → %s", first))
			}

		case "server_tool_use":
			name, _ := block["name"].(string)
			lines = append(lines, fmt.Sprintf("[%s]", name))
		}
	}

	return lines
}

func parseResultEvent(event map[string]any) []string {
	subtype, _ := event["subtype"].(string)
	durationMs, _ := event["duration_ms"].(float64)
	numTurns, _ := event["num_turns"].(float64)
	cost, _ := event["total_cost_usd"].(float64)

	status := "done"
	if subtype != "" {
		status = subtype
	}

	line := fmt.Sprintf("[result] %s", status)
	if numTurns > 0 {
		line += fmt.Sprintf(" | %d turns", int(numTurns))
	}
	if durationMs > 0 {
		secs := durationMs / 1000
		if secs >= 60 {
			line += fmt.Sprintf(" | %.0fm%.0fs", secs/60, float64(int(secs)%60))
		} else {
			line += fmt.Sprintf(" | %.1fs", secs)
		}
	}
	if cost > 0 {
		line += fmt.Sprintf(" | $%.4f", cost)
	}

	return []string{line}
}

func formatToolUse(name string, input map[string]any) string {
	switch name {
	case "Read":
		path, _ := input["file_path"].(string)
		return fmt.Sprintf("[Read] %s", path)
	case "Write":
		path, _ := input["file_path"].(string)
		return fmt.Sprintf("[Write] %s", path)
	case "Edit":
		path, _ := input["file_path"].(string)
		return fmt.Sprintf("[Edit] %s", path)
	case "Bash":
		cmd, _ := input["command"].(string)
		desc, _ := input["description"].(string)
		if desc != "" {
			return fmt.Sprintf("[Bash] %s $ %s", desc, truncate(cmd, 80))
		}
		return fmt.Sprintf("[Bash] $ %s", truncate(cmd, 100))
	case "Glob":
		pattern, _ := input["pattern"].(string)
		return fmt.Sprintf("[Glob] %s", pattern)
	case "Grep":
		pattern, _ := input["pattern"].(string)
		path, _ := input["path"].(string)
		if path != "" {
			return fmt.Sprintf("[Grep] %s in %s", pattern, path)
		}
		return fmt.Sprintf("[Grep] %s", pattern)
	case "Task":
		desc, _ := input["description"].(string)
		return fmt.Sprintf("[Task] %s", desc)
	case "TodoWrite":
		return "[TodoWrite]"
	default:
		return fmt.Sprintf("[%s]", name)
	}
}

func extractToolResultContent(block map[string]any) string {
	// Tool results can have various content formats
	if content, ok := block["content"].(string); ok {
		return content
	}
	if content, ok := block["output"].(string); ok {
		return content
	}
	if content, ok := block["text"].(string); ok {
		return content
	}
	return ""
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}
