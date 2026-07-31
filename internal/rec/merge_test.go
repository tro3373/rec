package rec

import (
	"slices"
	"testing"
)

func TestMergeSegments(t *testing.T) {
	tests := []struct {
		name  string
		self  []segment
		other []segment
		want  []line
	}{
		{
			name:  "両トラックに発言がある場合_開始時刻順に話者ラベル付きで並ぶこと",
			self:  []segment{{StartMs: 0, Text: "こんにちは"}, {StartMs: 5000, Text: "はい"}},
			other: []segment{{StartMs: 2000, Text: "よろしく"}},
			want: []line{
				{StartMs: 0, Speaker: speakerSelf, Text: "こんにちは"},
				{StartMs: 2000, Speaker: speakerOther, Text: "よろしく"},
				{StartMs: 5000, Speaker: speakerSelf, Text: "はい"},
			},
		},
		{
			name:  "開始時刻が同じ場合_自分を先に並べること",
			self:  []segment{{StartMs: 1000, Text: "A"}},
			other: []segment{{StartMs: 1000, Text: "B"}},
			want: []line{
				{StartMs: 1000, Speaker: speakerSelf, Text: "A"},
				{StartMs: 1000, Speaker: speakerOther, Text: "B"},
			},
		},
		{
			name:  "片方が空の場合_もう片方だけが並ぶこと",
			self:  nil,
			other: []segment{{StartMs: 0, Text: "だけ"}},
			want:  []line{{StartMs: 0, Speaker: speakerOther, Text: "だけ"}},
		},
		{
			name:  "両方空の場合_空になること",
			self:  nil,
			other: nil,
			want:  []line{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeSegments(tt.self, tt.other)
			if !slices.Equal(got, tt.want) {
				t.Errorf("mergeSegments() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestRenderTranscript(t *testing.T) {
	tests := []struct {
		name  string
		lines []line
		want  string
	}{
		{
			name: "時刻と話者を先頭に付けた箇条書きになること",
			lines: []line{
				{StartMs: 0, Speaker: speakerSelf, Text: "こんにちは"},
				{StartMs: 3661000, Speaker: speakerOther, Text: "よろしく"},
			},
			want: "- [00:00:00] 自分: こんにちは\n- [01:01:01] 相手: よろしく\n",
		},
		{
			name:  "発言が無い場合_空文字になること",
			lines: []line{},
			want:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderTranscript(tt.lines)
			if got != tt.want {
				t.Errorf("renderTranscript() = %q, want %q", got, tt.want)
			}
		})
	}
}
