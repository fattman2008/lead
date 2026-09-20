package gt

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/fattman2008/lead/internal/gitutil"
)

func TestOrderFromTrunksAndStack(t *testing.T) {
	nodes := map[string]*BranchNode{
		"main": {
			Name: "main",
			Kids: []string{"feat-a", "feat-b"},
		},
		"feat-a": {
			Name:   "feat-a",
			Parent: "main",
			Kids:   []string{"feat-a2"},
		},
		"feat-a2": {
			Name:   "feat-a2",
			Parent: "feat-a",
		},
		"feat-b": {
			Name:   "feat-b",
			Parent: "main",
		},
	}

	ordered := orderFromTrunks(nodes, []string{"main"})
	got := names(ordered)
	want := []string{"main", "feat-a", "feat-a2", "feat-b"}
	if len(got) != len(want) {
		t.Fatalf("order=%v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order[%d]=%q want %q (full %v)", i, got[i], want[i], got)
		}
		if i > 0 && ordered[i].Depth < 1 && got[i] != "main" {
			// feat-a and feat-b depth 1; feat-a2 depth 2
		}
	}
	if ordered[0].Depth != 0 || ordered[1].Depth != 1 || ordered[2].Depth != 2 || ordered[3].Depth != 1 {
		t.Fatalf("depths=%v", depths(ordered))
	}

	allowed := stackSet(nodes, "feat-a")
	for _, name := range []string{"main", "feat-a", "feat-a2"} {
		if !allowed[name] {
			t.Fatalf("expected %s in stack", name)
		}
	}
	if allowed["feat-b"] {
		t.Fatalf("feat-b should not be in feat-a stack")
	}
}

func TestListCheckoutBranchesLive(t *testing.T) {
	cwd := repoRoot(t)
	here, err := gitutil.CurrentBranch(cwd)
	if err != nil || here == "" {
		t.Fatalf("current branch: %v", err)
	}
	cs, err := ListCheckoutBranches(cwd, ListOpts{Current: here})
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) == 0 {
		t.Fatal("expected branches")
	}
	found := false
	for _, c := range cs {
		if c.Name == here {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("current branch %q missing from %v", here, names(cs))
	}

	stacked, err := ListCheckoutBranches(cwd, ListOpts{Current: here, Stack: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stacked) == 0 {
		t.Fatal("expected stack branches")
	}
	if len(stacked) > len(cs) {
		t.Fatalf("stack filter grew list: stack=%d all=%d", len(stacked), len(cs))
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

func names(cs []BranchChoice) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.Name
	}
	return out
}

func depths(cs []BranchChoice) []int {
	out := make([]int, len(cs))
	for i, c := range cs {
		out[i] = c.Depth
	}
	return out
}
