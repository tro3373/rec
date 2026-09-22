package rec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunPostCmd(t *testing.T) {
	tests := []struct {
		name    string
		args    string // Appended to the script path. Empty runs false instead.
		want    string // What the script records as its arguments.
		wantErr bool
	}{
		{name: "成功する場合_引数の後に出力先が渡されること", args: " arg", want: "arg out/1\n"},
		{name: "失敗する場合_エラーになること", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			marker := filepath.Join(dir, "ran")
			script := filepath.Join(dir, "post.sh")
			if err := os.WriteFile(script, []byte("#!/bin/sh\necho \"$1 $2\" > \""+marker+"\"\n"), 0o700); err != nil {
				t.Fatal(err)
			}
			cmdline := "false"
			if !tt.wantErr {
				cmdline = script + tt.args
			}
			err := runPostCmd(cmdline, "out/1")
			if (err != nil) != tt.wantErr {
				t.Fatalf("runPostCmd() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			got, err := os.ReadFile(marker)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("post command got %q, want %q", got, tt.want)
			}
		})
	}
}
