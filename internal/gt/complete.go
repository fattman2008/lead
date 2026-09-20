package gt

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Completion is one yargs completion entry (value plus optional description).
type Completion struct {
	Value string
	Desc  string
}

// CommandHelp is a gt subcommand discovered via yargs completions.
type CommandHelp struct {
	Name string
	Desc string
}

// YargsComplete asks gt for completions using the same protocol as
// `gt completion` shell scripts (`--get-yargs-completions` + COMP_*).
//
// words should start with "gt" (e.g. []string{"gt", "submit", "--dra"}).
// cword is the 0-based index of the word being completed.
func YargsComplete(words []string, cword int) ([]Completion, error) {
	if len(words) == 0 {
		words = []string{Bin}
	}
	if cword < 0 {
		cword = 0
	}
	if cword >= len(words) {
		cword = len(words) - 1
	}

	line := strings.Join(words, " ")
	args := append([]string{"--get-yargs-completions"}, words...)
	cmd := exec.Command(Bin, args...)
	cmd.Env = append(os.Environ(),
		"COMP_CWORD="+strconv.Itoa(cword),
		"COMP_LINE="+line,
		"COMP_POINT="+strconv.Itoa(len(line)),
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// yargs often exits non-zero when there are no matches; still use stdout.
		if stdout.Len() == 0 {
			msg := strings.TrimSpace(stderr.String())
			if msg == "" {
				msg = err.Error()
			}
			return nil, fmt.Errorf("gt completions: %s", msg)
		}
	}
	return parseYargsLines(stdout.String()), nil
}

// ListCommands returns gt subcommand names (and descriptions) for registration.
func ListCommands() ([]CommandHelp, error) {
	comps, err := YargsComplete([]string{Bin}, 0)
	if err != nil {
		return nil, err
	}
	out := make([]CommandHelp, 0, len(comps))
	for _, c := range comps {
		if c.Value == "" || strings.HasPrefix(c.Value, "-") {
			continue
		}
		out = append(out, CommandHelp{Name: c.Value, Desc: c.Desc})
	}
	return out, nil
}

func parseYargsLines(s string) []Completion {
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
