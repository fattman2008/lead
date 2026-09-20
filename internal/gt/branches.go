package gt

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fattman2008/lead/internal/gitutil"
	_ "modernc.org/sqlite"
)

// BranchNode is one Graphite-tracked branch and its parent link.
type BranchNode struct {
	Name   string
	Parent string
	Kids   []string
}

// ListOpts controls which branches appear in the checkout picker.
type ListOpts struct {
	Current       string // current branch (for --stack and cursor)
	Stack         bool   // only ancestors + descendants of Current
	All           bool   // include all configured trunks (vs primary trunk tree)
	ShowUntracked bool   // append local branches not tracked by Graphite
}

// BranchChoice is a picker row: branch name plus indent depth for display.
type BranchChoice struct {
	Name  string
	Depth int
}

// ListCheckoutBranches returns stack-ordered branches for interactive checkout.
func ListCheckoutBranches(cwd string, opts ListOpts) ([]BranchChoice, error) {
	cfg, err := readRepoConfig(cwd)
	if err != nil {
		return nil, err
	}
	nodes, err := loadBranchMetadata(cwd)
	if err != nil {
		return nil, err
	}

	trunks := cfg.trunkNames()
	if len(trunks) == 0 {
		return nil, fmt.Errorf("graphite repo config has no trunk")
	}
	primary := cfg.Trunk
	if primary == "" {
		primary = trunks[0]
	}

	roots := []string{primary}
	if opts.All {
		roots = trunks
	}

	ordered := orderFromTrunks(nodes, roots)
	if opts.Stack {
		if opts.Current == "" {
			return nil, fmt.Errorf("current branch required for --stack")
		}
		allowed := stackSet(nodes, opts.Current)
		var filtered []BranchChoice
		for _, c := range ordered {
			if allowed[c.Name] {
				filtered = append(filtered, c)
			}
		}
		ordered = filtered
	}

	if opts.ShowUntracked {
		local, err := gitutil.LocalBranches(cwd)
		if err != nil {
			return nil, err
		}
		tracked := make(map[string]bool, len(nodes))
		for name := range nodes {
			tracked[name] = true
		}
		seen := make(map[string]bool, len(ordered))
		for _, c := range ordered {
			seen[c.Name] = true
		}
		var extra []string
		for _, b := range local {
			if tracked[b] || seen[b] {
				continue
			}
			extra = append(extra, b)
		}
		sort.Strings(extra)
		for _, b := range extra {
			ordered = append(ordered, BranchChoice{Name: b, Depth: 0})
		}
	}

	if len(ordered) == 0 {
		return nil, fmt.Errorf("no branches to check out")
	}
	return ordered, nil
}

type repoConfig struct {
	Trunk  string `json:"trunk"`
	Trunks []struct {
		Name string `json:"name"`
	} `json:"trunks"`
}

func (c repoConfig) trunkNames() []string {
	seen := map[string]bool{}
	var out []string
	add := func(n string) {
		n = strings.TrimSpace(n)
		if n == "" || seen[n] {
			return
		}
		seen[n] = true
		out = append(out, n)
	}
	add(c.Trunk)
	for _, t := range c.Trunks {
		add(t.Name)
	}
	return out
}

func readRepoConfig(cwd string) (repoConfig, error) {
	common, err := gitutil.CommonDir(cwd)
	if err != nil {
		return repoConfig{}, err
	}
	data, err := os.ReadFile(filepath.Join(common, ".graphite_repo_config"))
	if err != nil {
		return repoConfig{}, fmt.Errorf("read graphite repo config: %w", err)
	}
	var cfg repoConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return repoConfig{}, fmt.Errorf("parse graphite repo config: %w", err)
	}
	return cfg, nil
}

func loadBranchMetadata(cwd string) (map[string]*BranchNode, error) {
	common, err := gitutil.CommonDir(cwd)
	if err != nil {
		return nil, err
	}
	dbPath := filepath.Join(common, ".graphite_metadata.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open graphite metadata: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(`SELECT branch_name, IFNULL(parent_branch_name, ''), IFNULL(children, '[]') FROM branch_metadata`)
	if err != nil {
		return nil, fmt.Errorf("query branch metadata: %w", err)
	}
	defer rows.Close()

	nodes := map[string]*BranchNode{}
	for rows.Next() {
		var name, parent, kidsJSON string
		if err := rows.Scan(&name, &parent, &kidsJSON); err != nil {
			return nil, err
		}
		var kids []string
		if err := json.Unmarshal([]byte(kidsJSON), &kids); err != nil {
			kids = nil
		}
		nodes[name] = &BranchNode{Name: name, Parent: parent, Kids: kids}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return nodes, nil
}

func orderFromTrunks(nodes map[string]*BranchNode, trunks []string) []BranchChoice {
	seen := map[string]bool{}
	var out []BranchChoice
	var walk func(name string, depth int)
	walk = func(name string, depth int) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		out = append(out, BranchChoice{Name: name, Depth: depth})
		n := nodes[name]
		if n == nil {
			return
		}
		kids := append([]string(nil), n.Kids...)
		sort.Strings(kids)
		for _, k := range kids {
			walk(k, depth+1)
		}
	}
	for _, t := range trunks {
		walk(t, 0)
	}
	// Orphans (tracked but not reachable from selected trunks), stable by name.
	var orphans []string
	for name := range nodes {
		if !seen[name] {
			orphans = append(orphans, name)
		}
	}
	sort.Strings(orphans)
	for _, name := range orphans {
		walk(name, 0)
	}
	return out
}

func stackSet(nodes map[string]*BranchNode, current string) map[string]bool {
	allowed := map[string]bool{current: true}
	// Ancestors
	for n := nodes[current]; n != nil && n.Parent != ""; n = nodes[n.Parent] {
		allowed[n.Parent] = true
	}
	// Descendants
	var queue []string
	if n := nodes[current]; n != nil {
		queue = append(queue, n.Kids...)
	}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if allowed[name] {
			continue
		}
		allowed[name] = true
		if n := nodes[name]; n != nil {
			queue = append(queue, n.Kids...)
		}
	}
	return allowed
}
