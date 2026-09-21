package complete

import (
	"testing"

	"github.com/fattman2008/lead/internal/gt"
	"github.com/spf13/cobra"
)

func TestGtPassthroughSkipCoversOverrides(t *testing.T) {
	for _, name := range []string{"create", "checkout", "up", "down", "sync", "delete", "list", "remove", "root", "pin", "completion"} {
		if !gtPassthroughSkip[name] {
			t.Fatalf("expected %q in gtPassthroughSkip", name)
		}
	}
}

func TestRegisterGtPassthroughsNoOpsWhenGtMissing(t *testing.T) {
	t.Setenv("PATH", "/nonexistent")
	root := &cobra.Command{Use: "pt"}
	root.AddCommand(&cobra.Command{Use: "create"})
	RegisterGtPassthroughs(root, func(name, alias, short string) *cobra.Command {
		t.Fatal("should not create passthroughs when gt is missing")
		return nil
	})
}

func TestFilterPrefix(t *testing.T) {
	comps := filterPrefix([]gt.Completion{
		{Value: "--draft", Desc: "draft"},
		{Value: "--stack", Desc: "stack"},
		{Value: "main"},
	}, "--d")
	if len(comps) != 1 || comps[0] != cobra.CompletionWithDesc("--draft", "draft") {
		t.Fatalf("got %#v", comps)
	}
}

func TestFromGtDirectiveOnFailure(t *testing.T) {
	t.Setenv("PATH", "/nonexistent")
	_, dir := FromGt("submit")(nil, nil, "--x")
	if dir != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("directive = %v", dir)
	}
}

func TestBranchArgStopsAfterFirst(t *testing.T) {
	comps, dir := BranchArg(nil, []string{"already"}, "x")
	if len(comps) != 0 {
		t.Fatalf("expected no comps, got %v", comps)
	}
	if dir != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("directive = %v", dir)
	}
}
