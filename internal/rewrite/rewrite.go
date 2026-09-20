// Package rewrite brands subprocess CLI output as pt.
//
// TODO(post-v1): even with path-safe boundaries, standalone "gt"/"wt" in user
// content still rewrites (e.g. commit subjects like "Change gt behavior").
// Prefer context-aware rewriting (help/hints only, or command-invocation
// patterns) so arbitrary user text is left alone.
package rewrite

import (
	"io"
	"regexp"
	"sync"
)

// Match gt/wt as command-like tokens, but not path segments (foo/gt/bar, gt.go).
var (
	gtToken = regexp.MustCompile(`(^|[^/.\w])gt\b`)
	wtToken = regexp.MustCompile(`(^|[^/.\w])wt\b`)
)

// Brand rewrites standalone gt/wt tokens to pt.
func Brand(s string) string {
	s = gtToken.ReplaceAllString(s, "${1}pt")
	s = wtToken.ReplaceAllString(s, "${1}pt")
	return s
}

// Writer wraps an io.Writer and brands text as it is written.
// Incomplete tokens across Write calls are not buffered; branding is
// best-effort per Write chunk (fine for line-oriented CLI output).
type Writer struct {
	dst io.Writer
	mu  sync.Mutex
}

func NewWriter(dst io.Writer) *Writer {
	return &Writer{dst: dst}
}

func (w *Writer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	branded := Brand(string(p))
	n, err := io.WriteString(w.dst, branded)
	if err != nil {
		return 0, err
	}
	// Report original length so exec.Cmd does not treat length changes as failure.
	// Token lengths match (gt/wt → pt), so n should equal len(p) when successful.
	if n == len(branded) {
		return len(p), nil
	}
	return n, err
}
