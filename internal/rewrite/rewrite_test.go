package rewrite

import "testing"

func TestBrand(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"run gt submit", "run pt submit"},
		{"see wt list", "see pt list"},
		{"graphite is fine", "graphite is fine"},
		{"worktrunk stays", "worktrunk stays"},
		{"Change gt behavior", "Change pt behavior"}, // known deficiency
		{"internal/gt/gt.go", "internal/gt/gt.go"},
		{"create mode 100644 internal/gt/gt.go", "create mode 100644 internal/gt/gt.go"},
		{"use `gt submit`", "use `pt submit`"},
	}
	for _, tc := range cases {
		if got := Brand(tc.in); got != tc.want {
			t.Errorf("Brand(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
