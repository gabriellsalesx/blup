package obsidian

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Prop is a property to set on a note. A nil Value removes the property; a
// []string becomes a list, anything else a single value.
type Prop struct {
	Key   string
	Value any
}

// Note is a note split into its properties (front matter) and text.
type Note struct {
	Props map[string]any
	Body  string
}

// ReadNote reads a note's properties and text. A missing file is an empty
// note, not an error.
func ReadNote(path string) (Note, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Note{Props: map[string]any{}}, nil
	}
	if err != nil {
		return Note{}, err
	}
	fm, body := splitFrontMatter(string(b))
	n := Note{Props: map[string]any{}, Body: body}
	if fm != "" {
		if err := yaml.Unmarshal([]byte(fm), &n.Props); err != nil {
			return Note{}, err
		}
		if n.Props == nil {
			n.Props = map[string]any{}
		}
	}
	return n, nil
}

// String returns a property as text ("" when missing or not a value).
func (n Note) String(key string) string {
	switch v := n.Props[key].(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		b, _ := yaml.Marshal(v)
		return strings.TrimSpace(string(b))
	}
}

// List returns a property as a list of texts; a single value is a list of one.
func (n Note) List(key string) []string {
	switch v := n.Props[key].(type) {
	case []any:
		out := make([]string, 0, len(v))
		for _, x := range v {
			if s, ok := x.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		if strings.TrimSpace(v) != "" {
			return []string{v}
		}
	}
	return nil
}

// SetProperties changes properties of the note at path, keeping the others,
// their order and the text. The note is created when missing.
func SetProperties(path string, props ...Prop) error {
	b, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	fm, body := splitFrontMatter(string(b))
	var doc yaml.Node
	if strings.TrimSpace(fm) != "" {
		if err := yaml.Unmarshal([]byte(fm), &doc); err != nil {
			return err
		}
	}
	if doc.Kind == 0 {
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}}
	}
	m := doc.Content[0]
	if m.Kind != yaml.MappingNode {
		return errors.New(path + ": the properties are not a list of names and values")
	}
	for _, p := range props {
		i := -1
		for k := 0; k+1 < len(m.Content); k += 2 {
			if m.Content[k].Value == p.Key {
				i = k
				break
			}
		}
		if p.Value == nil {
			if i >= 0 {
				m.Content = append(m.Content[:i], m.Content[i+2:]...)
			}
			continue
		}
		var v yaml.Node
		if err := v.Encode(p.Value); err != nil {
			return err
		}
		if i >= 0 {
			m.Content[i+1] = &v
		} else {
			m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: p.Key}, &v)
		}
	}
	var out bytes.Buffer
	if len(m.Content) > 0 {
		enc := yaml.NewEncoder(&out)
		enc.SetIndent(2)
		if err := enc.Encode(&doc); err != nil {
			return err
		}
		enc.Close()
	}
	text := body
	if out.Len() > 0 {
		text = "---\n" + out.String() + "---\n" + body
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(text), 0o644)
}

// splitFrontMatter separates the YAML between the opening "---" lines from
// the rest of the note.
func splitFrontMatter(s string) (fm, body string) {
	s = strings.TrimPrefix(s, "\ufeff")
	norm := strings.ReplaceAll(s, "\r\n", "\n")
	if !strings.HasPrefix(norm, "---\n") {
		return "", s
	}
	rest := norm[4:]
	if strings.HasPrefix(rest, "---\n") || rest == "---" {
		return "", strings.TrimPrefix(strings.TrimPrefix(rest, "---"), "\n")
	}
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		if strings.HasSuffix(rest, "\n---") {
			return rest[:len(rest)-4], ""
		}
		return "", s
	}
	return rest[:end+1], rest[end+5:]
}

// SetBody replaces the text of the note at path, keeping its properties.
func SetBody(path, body string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	s := string(b)
	fm, _ := splitFrontMatter(s)
	if fm == "" && !strings.HasPrefix(strings.ReplaceAll(strings.TrimPrefix(s, "\ufeff"), "\r\n", "\n"), "---\n") {
		return os.WriteFile(path, []byte(body), 0o644)
	}
	return os.WriteFile(path, []byte("---\n"+fm+"---\n"+body), 0o644)
}
