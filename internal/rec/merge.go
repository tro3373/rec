package rec

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

// Speaker labels appear in the transcript and the minutes, so they stay Japanese.
const (
	speakerSelf  = "自分"
	speakerOther = "相手"
)

// segment is one transcribed chunk of a single track.
type segment struct {
	StartMs int
	Text    string
}

// line is one utterance after merging both tracks.
type line struct {
	StartMs int
	Speaker string
	Text    string
}

// mergeSegments orders both tracks by start time and attaches speaker labels.
// Ties put the own voice first.
func mergeSegments(self, other []segment) []line {
	lines := make([]line, 0, len(self)+len(other))
	for _, s := range self {
		lines = append(lines, line{StartMs: s.StartMs, Speaker: speakerSelf, Text: s.Text})
	}
	for _, s := range other {
		lines = append(lines, line{StartMs: s.StartMs, Speaker: speakerOther, Text: s.Text})
	}
	slices.SortStableFunc(lines, func(a, b line) int { return cmp.Compare(a.StartMs, b.StartMs) })
	return lines
}

// renderTranscript formats the merged result as a markdown bullet list.
func renderTranscript(lines []line) string {
	var sb strings.Builder
	for _, l := range lines {
		fmt.Fprintf(&sb, "- [%s] %s: %s\n", formatTimestamp(l.StartMs), l.Speaker, l.Text)
	}
	return sb.String()
}

// formatTimestamp renders milliseconds as HH:MM:SS.
func formatTimestamp(ms int) string {
	sec := ms / 1000
	return fmt.Sprintf("%02d:%02d:%02d", sec/3600, sec/60%60, sec%60)
}
