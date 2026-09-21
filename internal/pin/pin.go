// Package pin persists a preferred worktree per repository for pt root.
package pin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fattman2008/lead/internal/gitutil"
)

const relativeFile = "lead/pins.json"

// File returns the default pins.json path under home.
func File(home string) string {
	return filepath.Join(home, ".config", relativeFile)
}

// DefaultFile returns the pins path under the current user's home.
func DefaultFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return File(home), nil
}

// Load reads main-worktree → pinned-worktree mappings. Missing file is empty.
func Load(file string) (map[string]string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	var pins map[string]string
	if err := json.Unmarshal(data, &pins); err != nil {
		return nil, fmt.Errorf("parse pins %s: %w", file, err)
	}
	if pins == nil {
		pins = map[string]string{}
	}
	out := make(map[string]string, len(pins))
	for k, v := range pins {
		out[gitutil.CanonPath(k)] = gitutil.CanonPath(v)
	}
	return out, nil
}

// Get returns the pinned worktree for a main worktree path.
func Get(file, mainPath string) (string, bool, error) {
	pins, err := Load(file)
	if err != nil {
		return "", false, err
	}
	p, ok := pins[gitutil.CanonPath(mainPath)]
	return p, ok && p != "", nil
}

// Set records a pin for mainPath.
func Set(file, mainPath, worktreePath string) error {
	pins, err := Load(file)
	if err != nil {
		return err
	}
	pins[gitutil.CanonPath(mainPath)] = gitutil.CanonPath(worktreePath)
	return save(file, pins)
}

// Clear removes the pin for mainPath.
func Clear(file, mainPath string) error {
	pins, err := Load(file)
	if err != nil {
		return err
	}
	delete(pins, gitutil.CanonPath(mainPath))
	return save(file, pins)
}

func save(file string, pins map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(pins, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(file, data, 0o644)
}
