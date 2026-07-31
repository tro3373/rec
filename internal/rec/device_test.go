package rec

import "testing"

func TestParseDevices(t *testing.T) {
	const info = `{"default_sink_name":"spk","default_source_name":"mic"}`
	const sinks = `[{"name":"other","monitor_source":"other.monitor"},
	                {"name":"spk","monitor_source":"spk.monitor"}]`

	tests := []struct {
		name    string
		info    string
		sinks   string
		want    devices
		wantErr bool
	}{
		{
			name:  "既定シンクとソースが揃う場合_自分はマイク相手はモニターになること",
			info:  info,
			sinks: sinks,
			want:  devices{Self: "mic", Other: "spk.monitor"},
		},
		{
			name:    "既定シンクが無い場合_エラーになること",
			info:    `{"default_source_name":"mic"}`,
			sinks:   sinks,
			wantErr: true,
		},
		{
			name:    "既定ソースが無い場合_エラーになること",
			info:    `{"default_sink_name":"spk"}`,
			sinks:   sinks,
			wantErr: true,
		},
		{
			name:    "既定シンクがsink一覧に無い場合_エラーになること",
			info:    info,
			sinks:   `[{"name":"other","monitor_source":"other.monitor"}]`,
			wantErr: true,
		},
		{
			name:    "既定シンクにモニターが無い場合_エラーになること",
			info:    info,
			sinks:   `[{"name":"spk","monitor_source":""}]`,
			wantErr: true,
		},
		{
			name:    "JSONとして壊れている場合_エラーになること",
			info:    `{"default_sink_name":`,
			sinks:   sinks,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDevices([]byte(tt.info), []byte(tt.sinks))
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseDevices() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Errorf("parseDevices() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
