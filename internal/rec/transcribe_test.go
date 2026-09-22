package rec

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestParseEngine(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    engine
		wantErr bool
	}{
		{name: "未指定の場合_既定のwhisperになること", in: "", want: engineWhisper},
		{name: "whisper指定の場合_whisperになること", in: "whisper", want: engineWhisper},
		{name: "gemini指定の場合_geminiになること", in: "gemini", want: engineGemini},
		{name: "未知の値の場合_エラーになること", in: "openai", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseEngine(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseEngine() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Errorf("parseEngine() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseWhisperJSON(t *testing.T) {
	const ok = `{
      "transcription": [
        {"offsets": {"from": 0, "to": 2000}, "text": " こんにちは"},
        {"offsets": {"from": 2000, "to": 4000}, "text": " [BLANK_AUDIO]"},
        {"offsets": {"from": 4000, "to": 6000}, "text": "  "},
        {"offsets": {"from": 6000, "to": 8000}, "text": " よろしく "}
      ]
    }`

	tests := []struct {
		name    string
		in      string
		want    []segment
		wantErr bool
	}{
		{
			name: "断片の開始ミリ秒と前後空白を除いた本文が取れること",
			in:   ok,
			want: []segment{{StartMs: 0, Text: "こんにちは"}, {StartMs: 6000, Text: "よろしく"}},
		},
		{
			name: "無音や空文字の断片が除外されること",
			in:   `{"transcription":[{"offsets":{"from":0},"text":" [BLANK_AUDIO]"}]}`,
			want: []segment{},
		},
		{
			name: "断片が無い場合_空になること",
			in:   `{"transcription":[]}`,
			want: []segment{},
		},
		{
			name:    "JSONとして壊れている場合_エラーになること",
			in:      `{"transcription":`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseWhisperJSON([]byte(tt.in))
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseWhisperJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("parseWhisperJSON() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseGeminiJSON(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    []segment
		wantErr bool
	}{
		{
			name: "開始時刻文字列がミリ秒に変換されること",
			in:   `[{"start":"00:00:00","text":"こんにちは"},{"start":"01:01:01","text":"よろしく"}]`,
			want: []segment{{StartMs: 0, Text: "こんにちは"}, {StartMs: 3661000, Text: "よろしく"}},
		},
		{
			name: "本文が空の断片が除外されること",
			in:   `[{"start":"00:00:01","text":"  "}]`,
			want: []segment{},
		},
		{
			name:    "開始時刻が不正な場合_エラーになること",
			in:      `[{"start":"あとで","text":"やあ"}]`,
			wantErr: true,
		},
		{
			name:    "JSONとして壊れている場合_エラーになること",
			in:      `[{"start":`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGeminiJSON([]byte(tt.in))
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseGeminiJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("parseGeminiJSON() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    int
		wantErr bool
	}{
		{name: "HHMMSS形式が変換できること", in: "01:02:03", want: 3723000},
		{name: "MMSS形式が変換できること", in: "02:03", want: 123000},
		{name: "区切りが多すぎる場合_エラーになること", in: "1:2:3:4", wantErr: true},
		{name: "数値でない場合_エラーになること", in: "aa:bb", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTimestamp(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseTimestamp() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Errorf("parseTimestamp() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDropRepeats(t *testing.T) {
	seg := func(ms int, text string) segment { return segment{StartMs: ms, Text: text} }

	tests := []struct {
		name string
		segs []segment
		want []segment
	}{
		{
			name: "3連続以上の同一テキストが2件に切り詰められること",
			segs: []segment{
				seg(0, "本題"),
				seg(1000, "ご視聴ありがとうございました"),
				seg(2000, "ご視聴ありがとうございました"),
				seg(3000, "ご視聴ありがとうございました"),
				seg(4000, "ご視聴ありがとうございました"),
				seg(5000, "続き"),
			},
			want: []segment{
				seg(0, "本題"),
				seg(1000, "ご視聴ありがとうございました"),
				seg(2000, "ご視聴ありがとうございました"),
				seg(5000, "続き"),
			},
		},
		{
			name: "2連続までの相槌は残ること",
			segs: []segment{seg(0, "はい。"), seg(1000, "はい。"), seg(2000, "本題")},
			want: []segment{seg(0, "はい。"), seg(1000, "はい。"), seg(2000, "本題")},
		},
		{
			name: "連続していない同一テキストは畳まれないこと",
			segs: []segment{seg(0, "はい。"), seg(1000, "本題"), seg(2000, "はい。")},
			want: []segment{seg(0, "はい。"), seg(1000, "本題"), seg(2000, "はい。")},
		},
		{
			name: "空入力で空が返ること",
			segs: nil,
			want: []segment{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dropRepeats(tt.segs)
			if !slices.Equal(got, tt.want) {
				t.Errorf("dropRepeats() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestPreflight(t *testing.T) {
	opts := Options{GeminiAPIKey: "key"}
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{name: "議事録コマンドがある場合_通ること", path: "bin", wantErr: false},
		{name: "議事録コマンドが無い場合_エラーになること", path: "empty", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			bin := filepath.Join(dir, "bin")
			if err := os.MkdirAll(filepath.Join(dir, "empty"), 0o750); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(bin, 0o750); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(bin, "claude"), []byte("#!/bin/sh\n"), 0o700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", filepath.Join(dir, tt.path))
			if err := preflight(engineGemini, opts); (err != nil) != tt.wantErr {
				t.Errorf("preflight() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
