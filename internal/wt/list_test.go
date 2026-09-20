package wt

import (
	"testing"
)

func TestParseListJSONSchema2(t *testing.T) {
	raw := []byte(`{
  "schema": 2,
  "items": [
    {
      "branch": "main",
      "worktree": {
        "path": "/repo",
        "main": true,
        "current": true,
        "changes": {"staged": false, "modified": true, "untracked": false, "conflicted": false}
      }
    },
    {"branch": "feature"}
  ]
}`)
	list, err := parseListJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 2 {
		t.Fatalf("items: got %d", len(list.Items))
	}
	if list.Items[0].Worktree == nil || list.Items[0].Worktree.Path != "/repo" {
		t.Fatalf("worktree path: %+v", list.Items[0].Worktree)
	}
	if !list.Items[0].Worktree.Changes.Dirty() {
		t.Fatal("expected dirty changes")
	}
	if list.Items[1].Worktree != nil {
		t.Fatalf("branch-only row should have nil worktree")
	}
}

func TestParseListJSONSchema1(t *testing.T) {
	raw := []byte(`[
  {
    "branch": "main",
    "path": "/repo",
    "kind": "worktree",
    "is_current": true,
    "is_main": true,
    "working_tree": {"staged": false, "modified": false, "untracked": true, "conflicted": false}
  },
  {
    "branch": "orphan",
    "kind": "branch",
    "is_current": false,
    "is_main": false
  }
]`)
	list, err := parseListJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 2 {
		t.Fatalf("items: got %d", len(list.Items))
	}
	wt0 := list.Items[0].Worktree
	if wt0 == nil || wt0.Path != "/repo" || !wt0.Main || !wt0.Current {
		t.Fatalf("normalized worktree: %+v", wt0)
	}
	if !wt0.Changes.Dirty() {
		t.Fatal("expected dirty from untracked")
	}
	if list.Items[1].Worktree != nil {
		t.Fatalf("branch-only: %+v", list.Items[1].Worktree)
	}
}
