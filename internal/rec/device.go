package rec

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
)

// devices holds the two sources to record.
type devices struct {
	Self  string // Own voice. The default microphone input.
	Other string // Remote voice. The monitor of the default speaker output.
}

// parseDevices resolves the recording sources from pactl JSON output.
// Human readable output is localized, so -f json is mandatory.
func parseDevices(infoJSON, sinksJSON []byte) (devices, error) {
	var info struct {
		DefaultSink   string `json:"default_sink_name"`
		DefaultSource string `json:"default_source_name"`
	}
	if err := json.Unmarshal(infoJSON, &info); err != nil {
		return devices{}, fmt.Errorf("cannot parse pactl info JSON: %w", err)
	}
	if info.DefaultSink == "" {
		return devices{}, errors.New("no default sink in pactl info")
	}
	if info.DefaultSource == "" {
		return devices{}, errors.New("no default source in pactl info")
	}

	var sinks []struct {
		Name    string `json:"name"`
		Monitor string `json:"monitor_source"`
	}
	if err := json.Unmarshal(sinksJSON, &sinks); err != nil {
		return devices{}, fmt.Errorf("cannot parse pactl list sinks JSON: %w", err)
	}
	for _, s := range sinks {
		if s.Name != info.DefaultSink {
			continue
		}
		if s.Monitor == "" {
			return devices{}, fmt.Errorf("sink %s has no monitor source", info.DefaultSink)
		}
		return devices{Self: info.DefaultSource, Other: s.Monitor}, nil
	}
	return devices{}, fmt.Errorf("default sink %s is not in the sink list", info.DefaultSink)
}

// detectDevices asks pactl for the recording sources.
func detectDevices() (devices, error) {
	info, err := exec.Command("pactl", "-f", "json", "info").Output()
	if err != nil {
		return devices{}, fmt.Errorf("cannot run pactl info: %w", err)
	}
	sinks, err := exec.Command("pactl", "-f", "json", "list", "sinks").Output()
	if err != nil {
		return devices{}, fmt.Errorf("cannot run pactl list sinks: %w", err)
	}
	return parseDevices(info, sinks)
}
