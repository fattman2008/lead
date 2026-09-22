package flow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fattman2008/lead/internal/wt"
)

func TestMainOccupancy(t *testing.T) {
	list := &wt.List{Items: []wt.Item{
		{Branch: "feat", Worktree: &wt.Worktree{Path: "/tmp/feat"}},
		{Branch: "main", Worktree: &wt.Worktree{Path: "/tmp/main", Main: true, Changes: wt.Changes{Modified: true}}},
	}}
	occ, err := mainOccupancy(list, "main")
	if err != nil {
		t.Fatal(err)
	}
	if occ.Path != "/tmp/main" || occ.Branch != "main" || occ.Trunk != "main" {
		t.Fatalf("got %+v", occ)
	}
	if !occ.Dirty {
		t.Fatal("expected dirty")
	}
	if occ.Borrowed() {
		t.Fatal("trunk on main is not borrowed")
	}

	list.Items[1].Branch = "feat"
	occ, err = mainOccupancy(list, "main")
	if err != nil {
		t.Fatal(err)
	}
	if !occ.Borrowed() || occ.Branch != "feat" {
		t.Fatalf("expected borrowed feat, got %+v", occ)
	}

	if _, err := mainOccupancy(&wt.List{}, "main"); err == nil {
		t.Fatal("expected no main worktree")
	}
}

func TestClassifyNav(t *testing.T) {
	idle := occupancy{Path: "/tmp/main", Branch: "main", Trunk: "main"}
	borrowed := occupancy{Path: "/tmp/main", Branch: "feat", Trunk: "main"}
	cases := []struct {
		name   string
		occ    occupancy
		target string
		want   navAction
	}{
		{"idle to feat", idle, "feat", navSwitch},
		{"idle to trunk", idle, "main", navSwitch},
		{"exit to worktree", borrowed, "feat", navExitWorktree},
		{"exit to trunk", borrowed, "main", navExitTrunk},
		{"jail other", borrowed, "other", navJail},
		{"detached not borrowed", occupancy{Path: "/tmp/main", Trunk: "main"}, "feat", navSwitch},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyNav(tc.occ, tc.target)
			if got != tc.want {
				t.Fatalf("classifyNav(%q) = %v, want %v", tc.target, got, tc.want)
			}
		})
	}
}

func TestCanRestoreMain(t *testing.T) {
	list := &wt.List{Items: []wt.Item{
		{Branch: "feat", Worktree: &wt.Worktree{Path: "/tmp/main", Main: true}},
	}}
	occ := occupancy{Path: "/tmp/main", Branch: "feat", Trunk: "main"}
	if err := canRestoreMain(list, occ); err != nil {
		t.Fatal(err)
	}

	occ.Dirty = true
	if err := canRestoreMain(list, occ); err == nil || !strings.Contains(err.Error(), "uncommitted") {
		t.Fatalf("got %v", err)
	}

	occ.Dirty = false
	if err := canRestoreMain(list, occupancy{Path: "/tmp/main", Branch: "main", Trunk: "main"}); err != nil {
		t.Fatal(err)
	}

	list.Items = append(list.Items, wt.Item{
		Branch:   "main",
		Worktree: &wt.Worktree{Path: "/tmp/trunk-wt"},
	})
	if err := canRestoreMain(list, occ); err == nil || !strings.Contains(err.Error(), "linked worktree") {
		t.Fatalf("got %v", err)
	}
}

func TestLinkedWorktree(t *testing.T) {
	list := &wt.List{Items: []wt.Item{
		{Branch: "feat", Worktree: &wt.Worktree{Path: "/tmp/main", Main: true}},
		{Branch: "feat", Worktree: &wt.Worktree{Path: "/tmp/feat"}},
		{Branch: "other", Worktree: nil},
	}}
	if w := linkedWorktree(list, "feat"); w == nil || w.Path != "/tmp/feat" {
		t.Fatalf("got %#v", w)
	}
	if w := linkedWorktree(list, "missing"); w != nil {
		t.Fatalf("got %#v", w)
	}
	list = &wt.List{Items: []wt.Item{
		{Branch: "feat", Worktree: &wt.Worktree{Path: "/tmp/main", Main: true}},
	}}
	if w := linkedWorktree(list, "feat"); w != nil {
		t.Fatalf("main is not a linked worktree: %#v", w)
	}
}

func TestIsWorktreeShortcut(t *testing.T) {
	if !isWorktreeShortcut("@") || !isWorktreeShortcut("-") || !isWorktreeShortcut("^") {
		t.Fatal("expected shortcuts")
	}
	if isWorktreeShortcut("feat") || isWorktreeShortcut("") {
		t.Fatal("did not expect shortcut")
	}
}

func TestMainJailError(t *testing.T) {
	err := mainJailError(occupancy{Branch: "feat", Trunk: "main"}, "other")
	msg := err.Error()
	for _, want := range []string{"feat", "--main other", "checkout feat", "checkout -t"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing %q in %q", want, msg)
		}
	}
}

func TestShouldCreateStay(t *testing.T) {
	main := t.TempDir()
	child := filepath.Join(main, "sub")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	other := t.TempDir()
	occ := occupancy{Path: main, Branch: "feat", Trunk: "main"}
	if !shouldCreateStay(occ, main) || !shouldCreateStay(occ, child) {
		t.Fatal("expected stay inside main worktree")
	}
	if shouldCreateStay(occ, other) {
		t.Fatal("did not expect stay in other worktree")
	}
	idle := occupancy{Path: main, Branch: "main", Trunk: "main"}
	if shouldCreateStay(idle, main) {
		t.Fatal("did not expect stay when not borrowed")
	}
}

func TestShouldReleaseMain(t *testing.T) {
	occ := occupancy{Path: "/tmp/main", Branch: "feat", Trunk: "main"}
	if !shouldReleaseMain(occ, []string{"feat"}) {
		t.Fatal("expected release")
	}
	if !shouldReleaseMain(occ, []string{"base", "feat"}) {
		t.Fatal("expected release in strip set")
	}
	if shouldReleaseMain(occ, []string{"other"}) {
		t.Fatal("did not expect release")
	}
	idle := occupancy{Path: "/tmp/main", Branch: "main", Trunk: "main"}
	if shouldReleaseMain(idle, []string{"main"}) {
		t.Fatal("idle main is not borrowed")
	}
}

func TestRejectMainRewrite(t *testing.T) {
	main := t.TempDir()
	other := t.TempDir()
	occ := occupancy{Path: main, Branch: "feat", Trunk: "main"}
	want := map[string]bool{"feat": true, "kid": true}

	if err := rejectMainRewrite(occ, main, want); err != nil {
		t.Fatalf("from main: %v", err)
	}
	if err := rejectMainRewrite(occ, other, want); err == nil || !strings.Contains(err.Error(), "main worktree") {
		t.Fatalf("from other: %v", err)
	}
	if err := rejectMainRewrite(occ, other, map[string]bool{"kid": true}); err != nil {
		t.Fatalf("branch not in want: %v", err)
	}
	idle := occupancy{Path: main, Branch: "main", Trunk: "main"}
	if err := rejectMainRewrite(idle, other, want); err != nil {
		t.Fatalf("not borrowed: %v", err)
	}
}

func TestContainsBranch(t *testing.T) {
	if !containsBranch([]string{"a", "b"}, "b") {
		t.Fatal("expected b")
	}
	if containsBranch([]string{"a"}, "b") {
		t.Fatal("did not expect b")
	}
	if containsBranch(nil, "a") {
		t.Fatal("empty")
	}
}
