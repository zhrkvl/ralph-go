package agent

import (
	"regexp"
	"testing"
)

var stampPrefix = regexp.MustCompile(`^\d{2}:\d{2}:\d{2} `)

// A flushed partial line used to be stamped twice — once by flush and once by
// parseLine — which defeats any exact match on the line's content.
func TestFlushedPartialTextIsStampedOnce(t *testing.T) {
	const text = "<promise>COMPLETE</promise>"

	sp := newStreamParser()
	delta := `{"type":"stream_event","event":{"type":"content_block_delta",` +
		`"delta":{"type":"text_delta","text":"` + text + `"}}}`
	if got := sp.parseLine(delta); len(got) != 0 {
		t.Fatalf("parseLine emitted %v before a newline or flush", got)
	}

	lines := sp.parseLine(`{"type":"result","subtype":"success","num_turns":81}`)
	if len(lines) != 2 {
		t.Fatalf("parseLine(result) = %v, want the flushed text plus the result line", lines)
	}
	for _, l := range lines {
		if !stampPrefix.MatchString(l) {
			t.Errorf("line %q has no timestamp", l)
		}
		if stampPrefix.MatchString(stampPrefix.ReplaceAllString(l, "")) {
			t.Errorf("line %q carries two timestamps", l)
		}
	}
	if got := stampPrefix.ReplaceAllString(lines[0], ""); got != text {
		t.Errorf("flushed line = %q, want %q", got, text)
	}
}

func TestFlushAtEndOfStreamIsStamped(t *testing.T) {
	sp := newStreamParser()
	sp.parseLine(`{"type":"stream_event","event":{"type":"content_block_delta",` +
		`"delta":{"type":"text_delta","text":"trailing"}}}`)

	lines := stamp(sp.flush())
	if len(lines) != 1 {
		t.Fatalf("flush() = %v, want one line", lines)
	}
	if got := stampPrefix.ReplaceAllString(lines[0], ""); got != "trailing" {
		t.Errorf("flushed line = %q, want timestamp + %q", lines[0], "trailing")
	}
}
