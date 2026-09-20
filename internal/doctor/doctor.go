package doctor

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/fattman2008/lead/internal/gt"
	"github.com/fattman2008/lead/internal/wt"
)

// Run checks that peer dependencies are available.
func Run(out io.Writer) error {
	ok := true
	if path, err := gt.LookPath(); err != nil {
		fmt.Fprintln(out, "gt: missing (install Graphite CLI: https://graphite.dev)")
		ok = false
	} else {
		fmt.Fprintf(out, "gt: ok (%s)\n", path)
	}
	if path, err := wt.LookPath(); err != nil {
		fmt.Fprintln(out, "wt: missing (install Worktrunk: https://worktrunk.dev)")
		ok = false
	} else {
		fmt.Fprintf(out, "wt: ok (%s)\n", path)
	}
	if _, err := exec.LookPath("git"); err != nil {
		fmt.Fprintln(out, "git: missing")
		ok = false
	} else {
		fmt.Fprintln(out, "git: ok")
	}
	if !ok {
		return fmt.Errorf("one or more dependencies missing")
	}
	return nil
}
