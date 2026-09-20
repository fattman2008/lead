package wt

import (
	"reflect"
	"testing"
)

func TestParseClapLines(t *testing.T) {
	in := "remove:Remove worktree\n--force:Force removal\nfeature:+ 5m\n"
	got := parseClapLines(in)
	want := []Completion{
		{Value: "remove", Desc: "Remove worktree"},
		{Value: "--force", Desc: "Force removal"},
		{Value: "feature", Desc: "+ 5m"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}
