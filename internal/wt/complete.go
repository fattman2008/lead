package wt

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Completion is one clap-style completion entry.
type Completion struct {
	Value string
	Desc  string
}

// ClapComplete asks wt for dynamic completions via COMPLETE=zsh (clap).
//
// words should start with "wt" (e.g. []string{"wt", "list", "--form"}).
// index is the 0-based index of the word being completed (_CLAP_COMPLETE_INDEX).
func ClapComplete(words []string, index int) ([]Completion, error) {
	if len(words) == 0 {
		words = []string{Bin}
	}
	if index < 0 {
		index = 0
	}
	if index >= len(words) {
		index = len(words) - 1
	}

	args := append([]string{"--"}, words...)
	cmd := exec.Command(Bin, args...)
	cmd.Env = append(os.Environ(),
		"COMPLETE=zsh",
		"_CLAP_COMPLETE_INDEX="+strconv.Itoa(index),
		"_CLAP_IFS=\n",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stdout.Len() == 0 {
			msg := strings.TrimSpace(stderr.String())
			if msg == "" {
				msg = err.Error()
			}
			return nil, fmt.Errorf("wt completions: %s", msg)
		}
	}
	return parseClapLines(stdout.String()), nil
}

func parseClapLines(s string) []Completion {
	var out []Completion
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		val, desc, _ := strings.Cut(line, ":")
		if val == "" {
			continue
		}
		out = append(out, Completion{Value: val, Desc: desc})
	}
	return out
}
