package flow

import (
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
			got := willStripHere(list, tc.strip, tc.here)
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
