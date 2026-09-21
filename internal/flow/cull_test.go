package flow

import (
	"testing"

	"github.com/fattman2008/lead/internal/wt"
)

func TestCullable(t *testing.T) {
	prefix := "/Users/demo/worktrees"
	cases := []struct {
		name string
		item wt.Item
		want bool
	}{
		{
			name: "managed",
			item: wt.Item{Branch: "parked", Worktree: &wt.Worktree{Path: "/Users/demo/worktrees/lead/parked"}},
			want: true,
		},
		{
			name: "unmanaged",
			item: wt.Item{Branch: "hotfix", Worktree: &wt.Worktree{Path: "/Users/demo/Projects/lead-hotfix"}},
			want: false,
		},
		{
			name: "main",
			item: wt.Item{Branch: "main", Worktree: &wt.Worktree{Path: "/Users/demo/Projects/lead", Main: true}},
			want: false,
		},
		{
			name: "no worktree",
			item: wt.Item{Branch: "orphan"},
			want: false,
		},
		{
			name: "empty prefix manages non-main",
			item: wt.Item{Branch: "hotfix", Worktree: &wt.Worktree{Path: "/Users/demo/Projects/lead-hotfix"}},
			want: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := prefix
			if tc.name == "empty prefix manages non-main" {
				p = ""
			}
			got := cullable(tc.item, p)
			if got != tc.want {
				t.Fatalf("cullable = %v, want %v", got, tc.want)
			}
		})
	}
}
