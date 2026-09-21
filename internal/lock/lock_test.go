package lock

import "testing"

func TestDirty(t *testing.T) {
	d := Dirty{}
	if locked, _ := d.Locked("feat", "/tmp/feat", false); locked {
		t.Fatal("clean should not lock")
	}
	locked, reason := d.Locked("feat", "/tmp/feat", true)
	if !locked {
		t.Fatal("dirty should lock")
	}
	if reason == "" {
		t.Fatal("expected reason")
	}
}

func TestAny(t *testing.T) {
	always := CheckerFunc(func(branch, path string, dirty bool) (bool, string) {
		return true, "always"
	})
	never := CheckerFunc(func(branch, path string, dirty bool) (bool, string) {
		return false, ""
	})
	a := Any{never, always}
	locked, reason := a.Locked("x", "/p", false)
	if !locked || reason != "always" {
		t.Fatalf("got locked=%v reason=%q", locked, reason)
	}
	if locked, _ := (Any{never}).Locked("x", "/p", false); locked {
		t.Fatal("all unlocked should not lock")
	}
}

// CheckerFunc adapts a function to Checker for tests.
type CheckerFunc func(branch, path string, dirty bool) (bool, string)

func (f CheckerFunc) Locked(branch, path string, dirty bool) (bool, string) {
	return f(branch, path, dirty)
}
