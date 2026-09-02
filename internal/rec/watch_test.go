package rec

import (
	"testing"
	"time"
)

func TestParseCallStreams(t *testing.T) {
	// Shapes taken from pactl on a real Slack huddle, Zoom call and Google Meet.
	const slackPaused = `[{"corked":true,"properties":{"application.process.binary":"slack"}}]`
	const slackLive = `[{"corked":false,"properties":{"application.process.binary":"slack"}}]`
	const meetTwoStreams = `[{"corked":false,"properties":{"application.process.binary":"chrome"}},
	                         {"corked":true,"properties":{"application.process.binary":"chrome"}}]`
	const ourOwnRecording = `[{"corked":false,"properties":{"application.process.binary":"ffmpeg"}}]`

	tests := []struct {
		name    string
		data    string
		want    callStreams
		wantErr bool
	}{
		{
			name: "誰もマイクを掴んでいない場合_0本になること",
			data: `[]`,
			want: callStreams{},
		},
		{
			name: "対象アプリが再生中の場合_稼働1本と数えること",
			data: slackLive,
			want: callStreams{Total: 1, Active: 1},
		},
		{
			name: "対象アプリが一時停止中の場合_存在のみ数えること",
			data: slackPaused,
			want: callStreams{Total: 1, Active: 0},
		},
		{
			name: "同時に2本立つ場合_両方数えること",
			data: meetTwoStreams,
			want: callStreams{Total: 2, Active: 1},
		},
		{
			name: "自分の録音のffmpegしかいない場合_数えないこと",
			data: ourOwnRecording,
			want: callStreams{},
		},
		{
			name:    "JSONとして壊れている場合_エラーになること",
			data:    `[{"corked":`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCallStreams([]byte(tt.data), watchedApps)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseCallStreams() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Errorf("parseCallStreams() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// observation is one poll of the stream counts at a point in time.
type observation struct {
	streams callStreams
	after   time.Duration // Elapsed since the start of the case.
	want    callEvent
}

func TestCallWatcherStep(t *testing.T) {
	const quietFor = 10 * time.Second
	live := callStreams{Total: 1, Active: 1}
	paused := callStreams{Total: 1, Active: 0}
	gone := callStreams{}

	tests := []struct {
		name string
		obs  []observation
	}{
		{
			name: "稼働ストリームが現れた場合_開始になること",
			obs: []observation{
				{streams: gone, want: callNone},
				{streams: live, want: callStarted},
			},
		},
		{
			name: "一時停止で立ち上がった場合_稼働するまで開始しないこと",
			obs: []observation{
				{streams: paused, want: callNone},
				{streams: live, want: callStarted},
			},
		},
		{
			name: "通話中にストリームを張り直した場合_終了しないこと",
			obs: []observation{
				{streams: live, want: callStarted},
				{streams: gone, after: time.Second, want: callNone},
				{streams: live, after: 2 * time.Second, want: callNone},
				{streams: gone, after: 3 * time.Second, want: callNone},
				{streams: live, after: 4 * time.Second, want: callNone},
			},
		},
		{
			name: "通話中にミュートした場合_終了しないこと",
			obs: []observation{
				{streams: live, want: callStarted},
				{streams: paused, after: time.Minute, want: callNone},
				{streams: paused, after: 2 * time.Minute, want: callNone},
			},
		},
		{
			name: "ストリームが消えて静穏時間が過ぎた場合_終了になること",
			obs: []observation{
				{streams: live, want: callStarted},
				{streams: gone, after: time.Second, want: callNone},
				{streams: gone, after: time.Second + quietFor - time.Millisecond, want: callNone},
				{streams: gone, after: time.Second + quietFor, want: callEnded},
			},
		},
		{
			name: "静穏の途中で復活した場合_計測をやり直すこと",
			obs: []observation{
				{streams: live, want: callStarted},
				{streams: gone, after: time.Second, want: callNone},
				{streams: live, after: 5 * time.Second, want: callNone},
				{streams: gone, after: 6 * time.Second, want: callNone},
				{streams: gone, after: 6*time.Second + quietFor - time.Millisecond, want: callNone},
				{streams: gone, after: 6*time.Second + quietFor, want: callEnded},
			},
		},
		{
			name: "終了した後にまた稼働した場合_もう一度開始になること",
			obs: []observation{
				{streams: live, want: callStarted},
				{streams: gone, after: time.Second, want: callNone},
				{streams: gone, after: time.Second + quietFor, want: callEnded},
				{streams: live, after: time.Minute, want: callStarted},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &callWatcher{quietFor: quietFor}
			base := time.Now()
			for i, o := range tt.obs {
				if got := w.step(o.streams, base.Add(o.after)); got != o.want {
					t.Errorf("step(%d) = %v, want %v", i, got, o.want)
				}
			}
		})
	}
}
