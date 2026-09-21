package pin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fattman2008/lead/internal/gitutil"
)

func TestFile(t *testing.T) {
	got := File("/tmp/home")
	want := filepath.Join("/tmp/home", ".config", "lead", "pins.json")
	if got != want {
		t.Fatalf("File = %q, want %q", got, want)
	}
}

func TestLoadMissingIsEmpty(t *testing.T) {
	file := filepath.Join(t.TempDir(), "missing.json")
	pins, err := Load(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(pins) != 0 {
		t.Fatalf("got %#v", pins)
	}
}

func TestSetGetClear(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "pins.json")
	main := filepath.Join(dir, "repo")
	wt := filepath.Join(dir, "worktree")
	if err := os.MkdirAll(main, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(wt, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := Set(file, main, wt); err != nil {
		t.Fatal(err)
	}
	got, ok, err := Get(file, main)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || got != gitutil.CanonPath(wt) {
		t.Fatalf("Get = %q ok=%v, want %q", got, ok, gitutil.CanonPath(wt))
	}

	if err := Clear(file, main); err != nil {
		t.Fatal(err)
	}
	got, ok, err = Get(file, main)
	if err != nil {
		t.Fatal(err)
	}
	if ok || got != "" {
		t.Fatalf("after clear Get = %q ok=%v", got, ok)
	}
}

func TestSetOverwrites(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "pins.json")
	main := filepath.Join(dir, "repo")
	first := filepath.Join(dir, "a")
	second := filepath.Join(dir, "b")
	for _, p := range []string{main, first, second} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := Set(file, main, first); err != nil {
		t.Fatal(err)
	}
	if err := Set(file, main, second); err != nil {
		t.Fatal(err)
	}
	got, ok, err := Get(file, main)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || got != gitutil.CanonPath(second) {
		t.Fatalf("Get = %q ok=%v", got, ok)
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	file := filepath.Join(t.TempDir(), "pins.json")
	if err := os.WriteFile(file, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(file); err == nil {
		t.Fatal("expected parse error")
	}
}
