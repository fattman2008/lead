package flow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fattman2008/lead/internal/gitutil"
	"github.com/fattman2008/lead/internal/wt"
)

func TestResolve(t *testing.T) {
	main := t.TempDir()
	feat := t.TempDir()
	other := t.TempDir()
	list := &wt.List{Items: []wt.Item{
		{Branch: "main", Worktree: &wt.Worktree{Path: main, Main: true}},
		{Branch: "feat", Worktree: &wt.Worktree{Path: feat}},
		{Branch: "orphan", Worktree: nil},
	}}

	t.Run("cwd worktree wins over pin", func(t *testing.T) {
		cwd := filepath.Join(feat, "internal")
		if err := os.MkdirAll(cwd, 0o755); err != nil {
			t.Fatal(err)
		}
		path, warn, err := Resolve(list, cwd, main)
		if err != nil {
			t.Fatal(err)
		}
		if warn != "" {
			t.Fatalf("warn = %q", warn)
		}
		if gitutil.CanonPath(path) != gitutil.CanonPath(feat) {
			t.Fatalf("path = %q, want %q", path, feat)
		}
	})

	t.Run("pin when cwd is outside", func(t *testing.T) {
		path, warn, err := Resolve(list, other, feat)
		if err != nil {
			t.Fatal(err)
		}
		if warn != "" {
			t.Fatalf("warn = %q", warn)
		}
		if gitutil.CanonPath(path) != gitutil.CanonPath(feat) {
			t.Fatalf("path = %q, want %q", path, feat)
		}
	})

	t.Run("stale pin falls back to main", func(t *testing.T) {
		gone := filepath.Join(t.TempDir(), "gone")
		path, warn, err := Resolve(list, other, gone)
		if err != nil {
			t.Fatal(err)
		}
		if warn == "" || !strings.Contains(warn, "not a worktree") {
			t.Fatalf("warn = %q", warn)
		}
		if gitutil.CanonPath(path) != gitutil.CanonPath(main) {
			t.Fatalf("path = %q, want main %q", path, main)
		}
	})

	t.Run("no pin uses main", func(t *testing.T) {
		path, warn, err := Resolve(list, other, "")
		if err != nil {
			t.Fatal(err)
		}
		if warn != "" {
			t.Fatalf("warn = %q", warn)
		}
		if gitutil.CanonPath(path) != gitutil.CanonPath(main) {
			t.Fatalf("path = %q, want main %q", path, main)
		}
	})

	t.Run("cwd in main uses main", func(t *testing.T) {
		cwd := filepath.Join(main, "pkg")
		if err := os.MkdirAll(cwd, 0o755); err != nil {
			t.Fatal(err)
		}
		path, warn, err := Resolve(list, cwd, feat)
		if err != nil {
			t.Fatal(err)
		}
		if warn != "" {
			t.Fatalf("warn = %q", warn)
		}
		if gitutil.CanonPath(path) != gitutil.CanonPath(main) {
			t.Fatalf("path = %q, want main %q", path, main)
		}
	})
}

func TestResolveNoMain(t *testing.T) {
	list := &wt.List{Items: []wt.Item{
		{Branch: "feat", Worktree: &wt.Worktree{Path: t.TempDir()}},
	}}
	_, _, err := Resolve(list, t.TempDir(), "")
	if err == nil || !strings.Contains(err.Error(), "no main worktree") {
		t.Fatalf("err = %v", err)
	}
}

func TestPinTarget(t *testing.T) {
	main := t.TempDir()
	feat := t.TempDir()
	list := &wt.List{Items: []wt.Item{
		{Branch: "main", Worktree: &wt.Worktree{Path: main, Main: true}},
		{Branch: "feat", Worktree: &wt.Worktree{Path: feat}},
		{Branch: "ghost"},
	}}

	t.Run("current worktree", func(t *testing.T) {
		cwd := filepath.Join(feat, "src")
		if err := os.MkdirAll(cwd, 0o755); err != nil {
			t.Fatal(err)
		}
		got, err := pinTarget(list, cwd, "")
		if err != nil {
			t.Fatal(err)
		}
		if gitutil.CanonPath(got) != gitutil.CanonPath(feat) {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("at is main", func(t *testing.T) {
		got, err := pinTarget(list, feat, "@")
		if err != nil {
			t.Fatal(err)
		}
		if gitutil.CanonPath(got) != gitutil.CanonPath(main) {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("named branch", func(t *testing.T) {
		got, err := pinTarget(list, main, "feat")
		if err != nil {
			t.Fatal(err)
		}
		if gitutil.CanonPath(got) != gitutil.CanonPath(feat) {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("missing worktree", func(t *testing.T) {
		_, err := pinTarget(list, main, "ghost")
		if err == nil || !strings.Contains(err.Error(), "no worktree") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("unknown branch", func(t *testing.T) {
		_, err := pinTarget(list, main, "nope")
		if err == nil || !strings.Contains(err.Error(), "no worktree") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("cwd outside", func(t *testing.T) {
		_, err := pinTarget(list, t.TempDir(), "")
		if err == nil || !strings.Contains(err.Error(), "not inside") {
			t.Fatalf("err = %v", err)
		}
	})
}
