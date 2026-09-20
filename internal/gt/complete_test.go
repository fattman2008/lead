package gt

import (
	"reflect"
	"testing"
)

func TestParseYargsLines(t *testing.T) {
	in := "submit:Push the stack\n--draft:Create as draft\nmain\n\n:bad\n"
	got := parseYargsLines(in)
	want := []Completion{
		{Value: "submit", Desc: "Push the stack"},
		{Value: "--draft", Desc: "Create as draft"},
		{Value: "main"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}
