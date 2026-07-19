package discord

import (
	"strings"
	"testing"
)

func TestSplitMessage_ShortContentUnchanged(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"empty", ""},
		{"single word", "hello"},
		{"multiline", "line one\nline two\nline three"},
		{"exactly at limit", strings.Repeat("a", maxMessageLength)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := splitMessage(tt.content, maxMessageLength)
			if len(chunks) != 1 {
				t.Fatalf("len(chunks) = %d, want 1", len(chunks))
			}
			if chunks[0] != tt.content {
				t.Errorf("chunks[0] = %q, want %q", chunks[0], tt.content)
			}
		})
	}
}

func TestSplitMessage_RespectsLimit(t *testing.T) {
	// A long run of words, well over the limit.
	content := strings.TrimSpace(strings.Repeat("word ", 2000))

	chunks := splitMessage(content, maxMessageLength)
	if len(chunks) < 2 {
		t.Fatalf("len(chunks) = %d, want at least 2", len(chunks))
	}

	for i, chunk := range chunks {
		if n := len([]rune(chunk)); n > maxMessageLength {
			t.Errorf("chunk %d has %d chars, want <= %d", i, n, maxMessageLength)
		}
	}
}

// Every character must survive the split; only the whitespace we break on is
// consumed. Reassembling with a single space must reproduce the input.
func TestSplitMessage_PreservesContent(t *testing.T) {
	content := strings.TrimSpace(strings.Repeat("alpha beta gamma ", 400))

	chunks := splitMessage(content, maxMessageLength)
	if got := strings.Join(chunks, " "); got != content {
		t.Errorf("rejoined content does not match input\ngot len %d, want len %d", len(got), len(content))
	}
}

func TestSplitMessage_BreaksOnNewline(t *testing.T) {
	// Two paragraphs that together exceed the limit, each individually under it.
	para := strings.Repeat("x", 1500)
	content := para + "\n" + para

	chunks := splitMessage(content, maxMessageLength)
	if len(chunks) != 2 {
		t.Fatalf("len(chunks) = %d, want 2", len(chunks))
	}
	if chunks[0] != para || chunks[1] != para {
		t.Error("expected the split to land on the newline boundary")
	}
}

// With both blank-line and single-newline separators available, the split must
// land on the blank line. Breaking on any newline strands part of a list or
// separates a heading from the text below it.
func TestSplitMessage_PrefersParagraphBreak(t *testing.T) {
	// Two blocks whose lines are single-newline separated, separated from each
	// other by a blank line. Each block is sized so the blank line sits well
	// past the midpoint of the window, where the progress guard accepts it.
	block := "intro line\n" + strings.Repeat("- a list item with trailing text\n", 42)
	content := block + "\n" + block

	chunks := splitMessage(content, maxMessageLength)
	if len(chunks) != 2 {
		t.Fatalf("len(chunks) = %d, want 2", len(chunks))
	}

	// The first chunk must be the whole first block, not a partial list.
	want := strings.TrimRight(block, "\n")
	if chunks[0] != want {
		t.Errorf("first chunk did not stop at the paragraph break\ngot  (%d chars) %q\nwant (%d chars) %q",
			len([]rune(chunks[0])), tail(chunks[0], 40), len([]rune(want)), tail(want, 40))
	}
}

func TestLastIndexParagraphBreak(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"no break", "a\nb\nc", -1},
		{"single break", "ab\n\ncd", 2},
		{"last of several", "a\n\nb\n\nc", 4},
		{"empty", "", -1},
		{"trailing pair", "ab\n\n", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lastIndexParagraphBreak([]rune(tt.input)); got != tt.want {
				t.Errorf("lastIndexParagraphBreak(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func tail(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return "..." + string(r[len(r)-n:])
}

// A single unbroken run has no natural break point; it must still be split
// rather than looping forever or emitting an oversized chunk.
func TestSplitMessage_HardSplitsUnbrokenRun(t *testing.T) {
	content := strings.Repeat("a", maxMessageLength*2+50)

	chunks := splitMessage(content, maxMessageLength)
	if len(chunks) != 3 {
		t.Fatalf("len(chunks) = %d, want 3", len(chunks))
	}
	for i, chunk := range chunks {
		if n := len([]rune(chunk)); n > maxMessageLength {
			t.Errorf("chunk %d has %d chars, want <= %d", i, n, maxMessageLength)
		}
	}
	if got := strings.Join(chunks, ""); got != content {
		t.Error("hard split lost content")
	}
}

// The limit is characters, not bytes: multi-byte runes must not be counted as
// several characters, nor split mid-rune.
func TestSplitMessage_CountsCharactersNotBytes(t *testing.T) {
	// Each emoji is 4 bytes but 1 character.
	content := strings.Repeat("🙂", maxMessageLength)

	chunks := splitMessage(content, maxMessageLength)
	if len(chunks) != 1 {
		t.Fatalf("len(chunks) = %d, want 1 (content is exactly at the character limit)", len(chunks))
	}
	if chunks[0] != content {
		t.Error("multi-byte content was altered")
	}
}

func TestSplitMessage_DoesNotSplitMidRune(t *testing.T) {
	content := strings.Repeat("🙂", maxMessageLength+100)

	chunks := splitMessage(content, maxMessageLength)
	for i, chunk := range chunks {
		if strings.ContainsRune(chunk, '�') {
			t.Errorf("chunk %d contains a replacement char, indicating a mid-rune split", i)
		}
	}
	if got := strings.Join(chunks, ""); got != content {
		t.Error("split lost or corrupted multi-byte content")
	}
}

// Whitespace-only content over the limit is pathological; it should be passed
// through so the API surfaces the error rather than silently sending nothing.
func TestSplitMessage_WhitespaceOnlyIsNotDropped(t *testing.T) {
	content := strings.Repeat(" ", maxMessageLength+10)

	chunks := splitMessage(content, maxMessageLength)
	if len(chunks) == 0 {
		t.Fatal("whitespace-only content was dropped entirely")
	}
}

func TestSplitMessage_NonPositiveLimit(t *testing.T) {
	content := "some content"

	chunks := splitMessage(content, 0)
	if len(chunks) != 1 || chunks[0] != content {
		t.Errorf("chunks = %v, want the content unchanged", chunks)
	}
}

func TestLastIndexRune(t *testing.T) {
	runes := []rune("a b\nc")

	if got := lastIndexRune(runes, '\n'); got != 3 {
		t.Errorf("lastIndexRune(newline) = %d, want 3", got)
	}
	if got := lastIndexRune(runes, ' '); got != 1 {
		t.Errorf("lastIndexRune(space) = %d, want 1", got)
	}
	if got := lastIndexRune(runes, 'z'); got != -1 {
		t.Errorf("lastIndexRune(missing) = %d, want -1", got)
	}
}
