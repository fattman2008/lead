package wt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrefixFromTemplate(t *testing.T) {
	home := "/Users/demo"
	cases := []struct {
		tmpl string
		want string
	}{
		{"", ""},
		{"~/worktrees/{{ repo }}/{{ branch | sanitize }}", filepath.Join(home, "worktrees")},
		{`~/worktrees/{{ repo }}/{{ branch | sanitize }}`, filepath.Join(home, "worktrees")},
		{"/abs/wt/{{ branch }}", "/abs/wt"},
		{"{{ branch }}", ""},
		{".worktrees/{{ branch }}", ""},
	}
	for _, tc := range cases {
		got := PrefixFromTemplate(tc.tmpl, home)
		if got != tc.want {
			t.Fatalf("PrefixFromTemplate(%q) = %q, want %q", tc.tmpl, got, tc.want)
		}
	}
}

func TestManagedPrefixFromHome(t *testing.T) {
	home := t.TempDir()

	t.Run("missing config", func(t *testing.T) {
		if got := ManagedPrefixFromHome(home); got != "" {
			t.Fatalf("got %q, want empty", got)
		}
	})

	t.Run("lead default", func(t *testing.T) {
		cfg := ConfigFile(home)
		if err := os.MkdirAll(filepath.Dir(cfg), 0o755); err != nil {
			t.Fatal(err)
		}
		content := "# Managed in part by lead (pt) setup\n\nworktree-path = \"~/worktrees/{{ repo }}/{{ branch | sanitize }}\"\n"
		if err := os.WriteFile(cfg, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		want := filepath.Join(home, "worktrees")
		if got := ManagedPrefixFromHome(home); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("commented and other keys ignored", func(t *testing.T) {
		cfg := ConfigFile(home)
		content := "# worktree-path = \"~/other/{{ branch }}\"\nother = 1\nworktree-path = '/custom/{{ repo }}/{{ branch }}' # trail\n"
		if err := os.WriteFile(cfg, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := ManagedPrefixFromHome(home); got != "/custom" {
			t.Fatalf("got %q, want /custom", got)
		}
	})
}

func TestIsManaged(t *testing.T) {
	prefix := "/Users/demo/worktrees"
	if !IsManaged("/Users/demo/worktrees/lead/feat", "") {
		t.Fatal("empty prefix manages all")
	}
	if !IsManaged("/Users/demo/worktrees/lead/feat", prefix) {
		t.Fatal("expected managed")
	}
	if IsManaged("/Users/demo/Projects/lead-hotfix", prefix) {
		t.Fatal("sibling project path is unmanaged")
	}
	if IsManaged("/Users/demo/worktrees-extra/feat", prefix) {
		t.Fatal("prefix must not match worktrees-extra")
	}
	if IsManaged("", prefix) {
		t.Fatal("empty path is not managed")
	}
}
