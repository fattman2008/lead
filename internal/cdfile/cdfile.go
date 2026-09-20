package cdfile

import (
	"fmt"
	"os"
	"strings"
)

const EnvVar = "LEAD_CD_FILE"

// Emit writes an absolute path for the shell wrapper to cd into.
// No-op if LEAD_CD_FILE is unset (binary invoked without shell integration).
func Emit(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("empty cd path")
	}
	file := os.Getenv(EnvVar)
	if file == "" {
		fmt.Fprintf(os.Stderr, "note: shell integration inactive; cd manually to %s\n", path)
		fmt.Fprintf(os.Stderr, "      run: pt setup\n")
		return nil
	}
	return os.WriteFile(file, []byte(path+"\n"), 0o600)
}
