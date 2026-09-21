package flow

import (
	"strings"
	"testing"

	"github.com/fattman2008/lead/internal/lock"
	"github.com/fattman2008/lead/internal/wt"
)

func TestDetachCandidates(t *testing.T) {
	list := &wt.List{Items: []wt.Item{
		{Branch: "main", Worktree: &wt.Worktree{Path: "/tmp/main", Main: true}},
		{Branch: "base", Worktree: &wt.Worktree{Path: "/tmp/base"}},
		{Branch: "feat", Worktree: &wt.Worktree{Path: "/tmp/feat", Changes: wt.Changes{Modified: true}}},
		{Branch: "clean", Worktree: &wt.Worktree{Path: "/tmp/clean"}},
		{Branch: "orphan", Worktree: nil},
	}}

	t.Run("skips main and current path", func(t *testing.T) {
		want := map[string]bool{"main": true, "base": true, "clean": true}
		got, err := detachCandidates(list, "/tmp/base", want, lock.Dirty{}, false)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].Branch != "clean" {
			t.Fatalf("got %#v", got)
		}
	})

	t.Run("dirty blocks without force", func(t *testing.T) {
		want := map[string]bool{"feat": true, "clean": true}
		_, err := detachCandidates(list, "/tmp/base", want, lock.Dirty{}, false)
		if err == nil || !strings.Contains(err.Error(), "uncommitted") {
			t.Fatalf("got err %v", err)
		}
	})

	t.Run("dirty allowed with force", func(t *testing.T) {
		want := map[string]bool{"feat": true}
		got, err := detachCandidates(list, "/tmp/base", want, lock.Dirty{}, true)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].Branch != "feat" {
			t.Fatalf("got %#v", got)
		}
	})

	t.Run("allOther skips missing worktrees", func(t *testing.T) {
		want := map[string]bool{}
		for _, item := range list.Items {
			if item.Branch != "" {
				want[item.Branch] = true
			}
		}
		got, err := detachCandidates(list, "/tmp/base", want, lock.Dirty{}, true)
		if err != nil {
			t.Fatal(err)
		}
		names := map[string]bool{}
		for _, d := range got {
			names[d.Branch] = true
		}
		if names["main"] || names["base"] || names["orphan"] {
			t.Fatalf("unexpected candidates %#v", got)
		}
		if !names["feat"] || !names["clean"] {
			t.Fatalf("missing candidates %#v", got)
		}
	})
}
