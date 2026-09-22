package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// envFile is the settings file shared by a manual run and the service.
func envFile() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot resolve the config directory: %w", err)
	}
	return filepath.Join(dir, "rec", "env"), nil
}

// parseEnv reads KEY=VALUE lines. Blank lines and # comments are skipped, and
// one pair of matching quotes around the value is removed.
func parseEnv(r io.Reader) (map[string]string, error) {
	vars := map[string]string{}
	sc := bufio.NewScanner(r)
	for n := 1; sc.Scan(); n++ {
		l := strings.TrimSpace(sc.Text())
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		key, value, ok := strings.Cut(l, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return nil, fmt.Errorf("line %d: expected KEY=VALUE: %q", n, l)
		}
		vars[key] = unquote(strings.TrimSpace(value))
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return vars, nil
}

// unquote removes one pair of matching single or double quotes.
func unquote(v string) string {
	if len(v) < 2 || (v[0] != '"' && v[0] != '\'') || v[len(v)-1] != v[0] {
		return v
	}
	return v[1 : len(v)-1]
}

// loadEnvFile exports the file's variables that the environment does not set
// already, so the shell wins over the file and a flag wins over both.
// A missing file is not an error.
func loadEnvFile(path string) error {
	f, err := os.Open(path) // #nosec G304 -- the user's own config file.
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("cannot open %s: %w", path, err)
	}
	defer f.Close()
	vars, err := parseEnv(f)
	if err != nil {
		return fmt.Errorf("cannot parse %s: %w", path, err)
	}
	for k, v := range vars {
		if _, set := os.LookupEnv(k); set {
			continue
		}
		if err := os.Setenv(k, v); err != nil {
			return fmt.Errorf("cannot set %s: %w", k, err)
		}
	}
	return nil
}
