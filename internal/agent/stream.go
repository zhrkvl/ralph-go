package agent

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// streamParser accumulates text deltas into lines and emits complete lines.
type streamParser struct {
	textBuf strings.Builder // accumulates text deltas until newline
}

func newStreamParser() *streamParser {
	return &streamParser{}
}

// stamp prepends a local timestamp to each non-empty line.
func stamp(lines []string) []string {
	if len(lines) == 0 {
		return nil
	}
	ts := time.Now().Format("15:04:05")
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		if l == "" {
			out = append(out, "")
		} else {
			out = append(out, ts+" "+l)
		}
	}
	return out
}

// parseLine converts a single line of Claude's stream-json output
// into zero or more human-readable display lines for the TUI.
//
// Stream-json with --include-partial-messages produces these event types:
//   {"type":"system","subtype":"init",...}
//   {"type":"assistant","message":{"content":[{"type":"text","text":"..."},{"type":"tool_use",...}]}}
//   {"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"..."}}}
//   {"type":"result",...}
func (sp *streamParser) parseLine(line string) []string {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	var event map[string]any
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		// Not JSON — return as-is (might be plain text from stderr)
		return stamp([]string{line})
	}

	typ, _ := event["type"].(string)

	switch typ {
	case "stream_event":
		return stamp(sp.parseStreamEvent(event))
	case "system":
		return stamp(sp.flushAndParse(parseSystemEvent(event)))
	case "assistant":
		return stamp(sp.flushAndParse(parseAssistantEvent(event)))
	case "result":
		return stamp(sp.flushAndParse(parseResultEvent(event)))
	default:
		return nil
	}
}

// flush emits any accumulated text as a stamped line (even if no trailing newline).
func (sp *streamParser) flush() []string {
	if sp.textBuf.Len() == 0 {
		return nil
	}
	line := sp.textBuf.String()
	sp.textBuf.Reset()
	return stamp([]string{line})
}

// flushAndParse flushes the text buffer before returning other parsed lines.
func (sp *streamParser) flushAndParse(lines []string) []string {
	flushed := sp.flush()
	return append(flushed, lines...)
}

// parseStreamEvent handles token-by-token text_delta events.
func (sp *streamParser) parseStreamEvent(event map[string]any) []string {
	ev, ok := event["event"].(map[string]any)
	if !ok {
		return nil
	}

	// Check for content_block_delta with text_delta
	delta, ok := ev["delta"].(map[string]any)
	if !ok {
		return nil
	}

	deltaType, _ := delta["type"].(string)
	if deltaType != "text_delta" {
		return nil
	}

	text, _ := delta["text"].(string)
	if text == "" {
		return nil
	}

	// Accumulate text. Emit complete lines (split on newline).
	var lines []string
	for _, ch := range text {
		if ch == '\n' {
			lines = append(lines, sp.textBuf.String())
			sp.textBuf.Reset()
		} else {
			sp.textBuf.WriteRune(ch)
		}
	}
	return lines
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
				for _, l := range strings.Split(text, "\n") {
					lines = append(lines, l)
				}
			}

		case "tool_use":
			name, _ := block["name"].(string)
			input, _ := block["input"].(map[string]any)
			lines = append(lines, formatToolUse(name, input))

		case "tool_result":
			content := extractToolResultContent(block)
			if content != "" {
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
