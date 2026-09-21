package flow

import (
	"strings"
	"testing"

	"github.com/fattman2008/lead/internal/wt"
)

func TestWillStripHere(t *testing.T) {
	list := &wt.List{Items: []wt.Item{
		{Branch: "feat", Worktree: &wt.Worktree{Path: "/tmp/feat"}},
		{Branch: "main", Worktree: &wt.Worktree{Path: "/tmp/main", Main: true}},
		{Branch: "orphan", Worktree: nil},
	}}
	cases := []struct {
		name  string
		strip []string
		here  string
		want  bool
	}{
		{"current in strip", []string{"feat"}, "feat", true},
		{"upstack includes here", []string{"base", "feat"}, "feat", true},
		{"here not in strip", []string{"other"}, "feat", false},
		{"main skipped", []string{"main"}, "main", false},
		{"no worktree", []string{"orphan"}, "orphan", false},
		{"empty here", []string{"feat"}, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := willStripHere(list, tc.strip, tc.here, "")
			if got != tc.want {
				t.Fatalf("willStripHere(%v, %q) = %v, want %v", tc.strip, tc.here, got, tc.want)
			}
		})
	}
}

func TestHasForceFlag(t *testing.T) {
	if !hasForceFlag([]string{"-f"}) {
		t.Fatal("expected -f")
	}
	if !hasForceFlag([]string{"--force", "branch"}) {
		t.Fatal("expected --force")
	}
	if hasForceFlag([]string{"--upstack", "branch"}) {
		t.Fatal("did not expect force")
	}
}

func TestShouldNavigateAfterDelete(t *testing.T) {
	cases := []struct {
		here     string
		hereGone bool
		want     bool
	}{
		{"feature", true, true},
		{"feature", false, false},
		{"", true, false},
		{"", false, false},
	}
	for _, tc := range cases {
		got := shouldNavigateAfterDelete(tc.here, tc.hereGone)
		if got != tc.want {
			t.Fatalf("shouldNavigateAfterDelete(%q, %v) = %v, want %v", tc.here, tc.hereGone, got, tc.want)
		}
	}
}

func TestPositionalBranch(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{nil, ""},
		{[]string{}, ""},
		{[]string{"-f"}, ""},
		{[]string{"--force", "--upstack"}, ""},
		{[]string{"feature"}, "feature"},
		{[]string{"-f", "feature"}, "feature"},
		{[]string{"--close", "feature", "--upstack"}, "feature"},
		{[]string{"feature", "-f"}, "feature"},
	}
	for _, tc := range cases {
		got := positionalBranch(tc.args)
		if got != tc.want {
			t.Fatalf("positionalBranch(%v) = %q, want %q", tc.args, got, tc.want)
		}
	}
}

func TestEnsureBranchArg(t *testing.T) {
	cases := []struct {
		args   []string
		branch string
		want   []string
	}{
		{nil, "feature", []string{"feature"}},
		{[]string{}, "feature", []string{"feature"}},
		{[]string{"-f"}, "feature", []string{"-f", "feature"}},
		{[]string{"--upstack"}, "feature", []string{"--upstack", "feature"}},
		{[]string{"feature"}, "feature", []string{"feature"}},
		{[]string{"-f", "other"}, "feature", []string{"-f", "other"}},
		{[]string{"--force"}, "", []string{"--force"}},
	}
	for _, tc := range cases {
		got := ensureBranchArg(tc.args, tc.branch)
		if len(got) != len(tc.want) {
			t.Fatalf("ensureBranchArg(%v, %q) = %v, want %v", tc.args, tc.branch, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("ensureBranchArg(%v, %q) = %v, want %v", tc.args, tc.branch, got, tc.want)
			}
		}
	}
}

func TestHasFlag(t *testing.T) {
	if !hasFlag([]string{"-f", "--upstack"}, "--upstack") {
		t.Fatal("expected --upstack")
	}
	if hasFlag([]string{"-f", "--force"}, "--upstack") {
		t.Fatal("did not expect --upstack")
	}
}

func TestRefuseUnmanagedDelete(t *testing.T) {
	prefix := "/Users/demo/worktrees"
	list := &wt.List{Items: []wt.Item{
		{Branch: "main", Worktree: &wt.Worktree{Path: "/Users/demo/Projects/lead", Main: true}},
		{Branch: "parked", Worktree: &wt.Worktree{Path: "/Users/demo/worktrees/lead/parked"}},
		{Branch: "hotfix", Worktree: &wt.Worktree{Path: "/Users/demo/Projects/lead-hotfix"}},
	}}

	t.Run("empty prefix allows all", func(t *testing.T) {
		if err := refuseUnmanagedDelete(list, []string{"hotfix", "parked"}, ""); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("managed only ok", func(t *testing.T) {
		if err := refuseUnmanagedDelete(list, []string{"parked"}, prefix); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("unmanaged errors before strip", func(t *testing.T) {
		err := refuseUnmanagedDelete(list, []string{"hotfix"}, prefix)
		if err == nil || !strings.Contains(err.Error(), "unmanaged worktree") || !strings.Contains(err.Error(), "hotfix") {
			t.Fatalf("got err %v", err)
		}
		if !strings.Contains(err.Error(), "/Users/demo/Projects/lead-hotfix") {
			t.Fatalf("error should name path: %v", err)
		}
	})

	t.Run("upstack mixed lists unmanaged", func(t *testing.T) {
		err := refuseUnmanagedDelete(list, []string{"parked", "hotfix"}, prefix)
		if err == nil || !strings.Contains(err.Error(), "hotfix") {
			t.Fatalf("got err %v", err)
		}
		if strings.Contains(err.Error(), "parked") {
			t.Fatalf("managed branch should not be in error: %v", err)
		}
	})

	t.Run("main is not unmanaged", func(t *testing.T) {
		if err := refuseUnmanagedDelete(list, []string{"main"}, prefix); err != nil {
			t.Fatal(err)
		}
	})
}

func TestWillStripHereUnmanaged(t *testing.T) {
	prefix := "/Users/demo/worktrees"
	list := &wt.List{Items: []wt.Item{
		{Branch: "hotfix", Worktree: &wt.Worktree{Path: "/Users/demo/Projects/lead-hotfix"}},
		{Branch: "parked", Worktree: &wt.Worktree{Path: "/Users/demo/worktrees/lead/parked"}},
	}}
	if willStripHere(list, []string{"hotfix"}, "hotfix", prefix) {
		t.Fatal("unmanaged current should not relocate-for-strip")
	}
	if !willStripHere(list, []string{"parked"}, "parked", prefix) {
		t.Fatal("managed current should strip")
	}
}
