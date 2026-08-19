package rec

import (
	"slices"
	"testing"
)

func TestTracks(t *testing.T) {
	dev := devices{Self: "mic", Other: "spk.monitor"}

	tests := []struct {
		name string
		dir  string
		want []track
	}{
		{
			name: "2トラック分が自分_相手の順で組み立てられること",
			dir:  "/out/20260731",
			want: []track{
				{Speaker: speakerSelf, Source: "mic", Path: "/out/20260731/self.wav"},
				{Speaker: speakerOther, Source: "spk.monitor", Path: "/out/20260731/other.wav"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tracks(dev, tt.dir)
			if !slices.Equal(got, tt.want) {
				t.Errorf("tracks() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestFfmpegCmd(t *testing.T) {
	tests := []struct {
		name  string
		track track
		want  []string
	}{
		{
			name:  "pulseのsourceを16kHzモノラルwavへ録る引数になること",
			track: track{Speaker: speakerSelf, Source: "mic", Path: "/out/self.wav"},
			want: []string{
				"ffmpeg",
				"-hide_banner", "-loglevel", "error", "-nostdin",
				"-f", "pulse", "-i", "mic",
				"-ac", "1", "-ar", "16000",
				"-y", "/out/self.wav",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ffmpegCmd(ffmpegArgs(tt.track)...).Args
			if !slices.Equal(got, tt.want) {
				t.Errorf("ffmpegCmd(ffmpegArgs()) = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMixArgs(t *testing.T) {
	tests := []struct {
		name   string
		tracks []track
		path   string
		want   []string
	}{
		{
			name: "2トラックを原音量のままリミッタ付きで1本のwavへ合成する引数になること",
			tracks: []track{
				{Speaker: speakerSelf, Path: "/out/self.wav"},
				{Speaker: speakerOther, Path: "/out/other.wav"},
			},
			path: "/out/mixed.wav",
			want: []string{
				"ffmpeg",
				"-hide_banner", "-loglevel", "error", "-nostdin",
				"-i", "/out/self.wav",
				"-i", "/out/other.wav",
				"-filter_complex", "amix=inputs=2:duration=longest:normalize=0,alimiter=limit=0.95",
				"-y", "/out/mixed.wav",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ffmpegCmd(mixArgs(tt.tracks, tt.path)...).Args
			if !slices.Equal(got, tt.want) {
				t.Errorf("ffmpegCmd(mixArgs()) = %v, want %v", got, tt.want)
			}
		})
	}
}
