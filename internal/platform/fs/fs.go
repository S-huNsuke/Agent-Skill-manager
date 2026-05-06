package fs

import (
	"fmt"
	"os"
	"path/filepath"
)

// ResolveRepoRoot finds the nearest ancestor containing the Wails config file.
func ResolveRepoRoot(start string) (string, error) {
	absStart, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve start path: %w", err)
	}

	current := absStart
	for {
		if _, err := os.Stat(filepath.Join(current, "wails.json")); err == nil {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("wails.json not found from %s", absStart)
		}
		current = parent
	}
}
