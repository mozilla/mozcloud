// Package scaffold generates annotated YAML values files from JSON schemas.
package scaffold

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Schema is a minimal JSON Schema representation covering the fields we need
// to generate useful YAML output.
type Schema struct {
	Type                 string             `json:"type"`
	Description          string             `json:"description"`
	Properties           map[string]*Schema `json:"properties"`
	AdditionalProperties *AdditionalProps   `json:"additionalProperties"`
	Items                *Schema            `json:"items"`
	Required             []string           `json:"required"`
	Default              interface{}        `json:"default"`
	AnyOf                []*Schema          `json:"anyOf"`
	OneOf                []*Schema          `json:"oneOf"`
	Ref                  string             `json:"$ref"`
	Defs                 map[string]*Schema `json:"$defs"`
	Enum                 []interface{}      `json:"enum"`
}

// AdditionalProperties can be a bool or a schema object in JSON Schema.
// We unmarshal it manually to handle both.
type AdditionalProps struct {
	Schema *Schema
}

func (a *AdditionalProps) UnmarshalJSON(data []byte) error {
	// Try schema first
	var s Schema
	if err := json.Unmarshal(data, &s); err == nil && s.Type != "" {
		a.Schema = &s
		return nil
	}
	// Ignore bool (true/false)
	return nil
}

// GenerateYAML produces an annotated YAML string from the schema, wrapped
// under a top-level key (e.g. "mozcloud").
func GenerateYAML(root *Schema, topKey string) string {
	g := &generator{root: root}
	var sb strings.Builder
	if topKey != "" {
		sb.WriteString(topKey + ":\n")
		g.writeObject(&sb, root, 1, root.Required)
	} else {
		g.writeObject(&sb, root, 0, root.Required)
	}
	return sb.String()
}

type generator struct {
	root *Schema
}

func (g *generator) resolve(s *Schema) *Schema {
	if s.Ref == "" {
		return s
	}
	// Only handle local $defs refs: "#/$defs/Foo"
	ref := strings.TrimPrefix(s.Ref, "#/$defs/")
	if g.root.Defs != nil {
		if def, ok := g.root.Defs[ref]; ok {
			return def
		}
	}
	return s
}

// effectiveType returns the resolved type, unwrapping anyOf/oneOf nullables.
// e.g. anyOf: [{type: string}, {type: null}] → "string"
func (g *generator) effectiveType(s *Schema) string {
	if s.Type != "" {
		return s.Type
	}
	candidates := append(s.AnyOf, s.OneOf...)
	for _, c := range candidates {
		c = g.resolve(c)
		if c.Type != "" && c.Type != "null" {
			return c.Type
		}
	}
	return ""
}

// effectiveSchema returns the most informative schema from anyOf/oneOf nullable unions.
func (g *generator) effectiveSchema(s *Schema) *Schema {
	if s.Type != "" || len(s.Properties) > 0 {
		return s
	}
	candidates := append(s.AnyOf, s.OneOf...)
	for _, c := range candidates {
		c = g.resolve(c)
		if c.Type != "" && c.Type != "null" {
			return c
		}
	}
	return s
}

func (g *generator) writeObject(sb *strings.Builder, s *Schema, depth int, required []string) {
	if len(s.Properties) == 0 {
		return
	}

	requiredSet := make(map[string]bool, len(required))
	for _, r := range required {
		requiredSet[r] = true
	}

	// Sort: required fields first, then alphabetical within each group.
	keys := make([]string, 0, len(s.Properties))
	for k := range s.Properties {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		ri, rj := requiredSet[keys[i]], requiredSet[keys[j]]
		if ri != rj {
			return ri
		}
		return keys[i] < keys[j]
	})

	indent := strings.Repeat("  ", depth)
	for _, key := range keys {
		prop := g.resolve(s.Properties[key])
		prop = g.effectiveSchema(prop)
		typ := g.effectiveType(prop)

		if prop.Description != "" {
			for _, line := range strings.Split(prop.Description, "\n") {
				sb.WriteString(indent + "# " + strings.TrimSpace(line) + "\n")
			}
		}

		switch typ {
		case "object":
			if len(prop.Properties) > 0 {
				sb.WriteString(indent + key + ":\n")
				g.writeObject(sb, prop, depth+1, prop.Required)
			} else if prop.AdditionalProperties != nil && prop.AdditionalProperties.Schema != nil {
				// Named map pattern (e.g. workloads, configMaps) — emit a commented example
				g.writeAdditionalProps(sb, key, prop.AdditionalProperties.Schema, depth)
			} else {
				sb.WriteString(indent + key + ": {}\n")
			}
		case "array":
			sb.WriteString(indent + key + ": []\n")
		case "boolean":
			val := g.defaultBool(prop)
			sb.WriteString(fmt.Sprintf("%s%s: %v\n", indent, key, val))
		case "integer", "number":
			val := g.defaultNumber(prop)
			sb.WriteString(fmt.Sprintf("%s%s: %v\n", indent, key, val))
		default: // string, or unknown
			val := g.defaultString(prop)
			sb.WriteString(fmt.Sprintf("%s%s: %q\n", indent, key, val))
		}

		// Blank line between top-level keys for readability
		if depth == 1 {
			sb.WriteString("\n")
		}
	}
}

// writeAdditionalProps emits a commented example entry for map-of-objects fields.
func (g *generator) writeAdditionalProps(sb *strings.Builder, key string, itemSchema *Schema, depth int) {
	indent := strings.Repeat("  ", depth)
	itemSchema = g.resolve(itemSchema)
	itemSchema = g.effectiveSchema(itemSchema)

	sb.WriteString(indent + key + ":\n")
	// Write a commented skeleton of one entry
	inner := &strings.Builder{}
	innerIndent := strings.Repeat("  ", depth+2)
	innerG := &generator{root: g.root}
	innerG.writeObject(inner, itemSchema, depth+2, itemSchema.Required)

	commentIndent := indent + "  "
	sb.WriteString(commentIndent + "# <name>:\n")
	for _, line := range strings.Split(strings.TrimRight(inner.String(), "\n"), "\n") {
		sb.WriteString(commentIndent + "# " + strings.TrimPrefix(line, innerIndent) + "\n")
	}
	sb.WriteString("\n")
}

func (g *generator) defaultString(s *Schema) string {
	if s.Default != nil {
		return fmt.Sprintf("%v", s.Default)
	}
	if len(s.Enum) > 0 {
		return fmt.Sprintf("%v", s.Enum[0])
	}
	return ""
}

func (g *generator) defaultBool(s *Schema) bool {
	if s.Default != nil {
		if b, ok := s.Default.(bool); ok {
			return b
		}
	}
	return false
}

func (g *generator) defaultNumber(s *Schema) interface{} {
	if s.Default != nil {
		return s.Default
	}
	return 0
}
