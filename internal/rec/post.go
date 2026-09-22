package rec

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// runPostCmd runs the configured post command with the output directory as its
// last argument. The command line is split on spaces only, like -minutes-cmd.
func runPostCmd(cmdline, dir string) error {
	argv := append(strings.Fields(cmdline), dir)
	// #nosec G204 -- the command line is the user's own configuration.
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("post command %s failed: %w", argv[0], err)
	}
	return nil
}
