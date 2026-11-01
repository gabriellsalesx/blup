package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Tags are the tags used in the notes' properties (tags: [a, b] or
// tags: a), the most used first, for the tags field's suggestions.
func (s *Studio) Tags() ([]string, error) {
	site, err := s.current()
	if err != nil {
		return nil, err
	}
	count := map[string]int{}
	root := site.ContentDir()
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != root && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(p), ".md") {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		fm, _ := splitFM(string(b))
		if fm == "" {
			return nil
		}
		var props struct {
			Tags any `yaml:"tags"`
		}
		if yaml.Unmarshal([]byte(fm), &props) != nil {
			return nil
		}
		for _, t := range tagList(props.Tags) {
			count[t]++
		}
		return nil
	})
	out := make([]string, 0, len(count))
	for t := range count {
		out = append(out, t)
	}
	slices.SortFunc(out, func(a, b string) int {
		if count[a] != count[b] {
			return count[b] - count[a]
		}
		return strings.Compare(strings.ToLower(a), strings.ToLower(b))
	})
	return out, err
}

// tagList reads a tags property: a list, or one tag (or several separated
// by commas); a leading # is dropped.
func tagList(v any) []string {
	var raw []string
	switch x := v.(type) {
	case string:
		raw = strings.Split(x, ",")
	case []any:
		for _, i := range x {
			if s, ok := i.(string); ok {
				raw = append(raw, s)
			}
		}
	}
	var out []string
	for _, t := range raw {
		if t = strings.TrimPrefix(strings.TrimSpace(t), "#"); t != "" {
			out = append(out, t)
		}
	}
	return out
}
