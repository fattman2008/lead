package wt

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/fattman2008/lead/internal/gitutil"
)

// ConfigFile returns the Worktrunk config.toml path under home.
func ConfigFile(home string) string {
	return filepath.Join(home, ".config", "worktrunk", "config.toml")
}

// ManagedPrefixFromHome reads Worktrunk's worktree-path and returns the
// directory prefix before the first {{ template. Empty means "manage all"
// (missing config, missing key, or unresolvable relative path).
func ManagedPrefixFromHome(home string) string {
	if home == "" {
		return ""
	}
	data, err := os.ReadFile(ConfigFile(home))
	if err != nil {
		return ""
	}
	return PrefixFromTemplate(parseWorktreePathValue(string(data)), home)
}

// PrefixFromTemplate expands ~ and returns the absolute directory prefix of a
// Worktrunk worktree-path template (text before the first {{). Empty if the
// template is unset or does not resolve to an absolute path.
func PrefixFromTemplate(tmpl, home string) string {
	tmpl = strings.TrimSpace(tmpl)
	if tmpl == "" {
		return ""
	}
	prefix := tmpl
	if i := strings.Index(tmpl, "{{"); i >= 0 {
		prefix = tmpl[:i]
	}
	prefix = strings.TrimSpace(prefix)
	prefix = expandTilde(prefix, home)
	if prefix == "" || !filepath.IsAbs(prefix) {
		return ""
	}
	return filepath.Clean(prefix)
}

// IsManaged reports whether path is under prefix. An empty prefix means every
// path is managed (legacy fallback when worktree-path is unset).
func IsManaged(path, prefix string) bool {
	if prefix == "" {
		return true
	}
	if path == "" {
		return false
	}
	return gitutil.ContainsPath(prefix, path)
}

func parseWorktreePathValue(text string) string {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		key, val, ok := strings.Cut(trimmed, "=")
		if !ok || strings.TrimSpace(key) != "worktree-path" {
			continue
		}
		return unquoteTOML(strings.TrimSpace(val))
	}
	return ""
}

func unquoteTOML(s string) string {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') {
		q := s[0]
		for i := 1; i < len(s); i++ {
			if s[i] == q {
				return s[1:i]
			}
		}
	}
	if i := strings.Index(s, "#"); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	return s
}

func expandTilde(p, home string) string {
	if p == "~" {
		return home
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home, p[2:])
	}
	return p
}
