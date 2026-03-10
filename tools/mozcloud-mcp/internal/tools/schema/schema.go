// Package schema implements JSON schema validation tools:
//   - schema_validate_values
package schema

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mozilla/mozcloud/tools/mozcloud-mcp/internal/mcperr"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

// --- schema_validate_values ---

type validationError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type validateValuesResult struct {
	Valid  bool              `json:"valid"`
	Errors []validationError `json:"errors"`
}

func SchemaValidateValues(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	valuesYAML := req.GetString("values_yaml", "")
	valuesFile := req.GetString("values_file", "")
	schemaJSON := req.GetString("schema_json", "")
	schemaFile := req.GetString("schema_file", "")

	// Resolve values
	if valuesYAML == "" && valuesFile == "" {
		return mcp.NewToolResultText(mcperr.New(
			"invalid_input",
			"at least one of values_yaml or values_file is required",
			"Provide values_yaml (inline YAML string) or values_file (path to a YAML file)",
		).JSON()), nil
	}
	if schemaJSON == "" && schemaFile == "" {
		return mcp.NewToolResultText(mcperr.New(
			"invalid_input",
			"at least one of schema_json or schema_file is required",
			"Provide schema_json (inline JSON schema string) or schema_file (path to a JSON schema file)",
		).JSON()), nil
	}

	if valuesYAML == "" {
		data, err := os.ReadFile(valuesFile)
		if err != nil {
			return mcp.NewToolResultText(mcperr.New(
				"file_read_error",
				"failed to read values_file: "+err.Error(),
				"Ensure the file exists at: "+valuesFile,
			).JSON()), nil
		}
		valuesYAML = string(data)
	}

	if schemaJSON == "" {
		data, err := os.ReadFile(schemaFile)
		if err != nil {
			return mcp.NewToolResultText(mcperr.New(
				"file_read_error",
				"failed to read schema_file: "+err.Error(),
				"Ensure the file exists at: "+schemaFile,
			).JSON()), nil
		}
		schemaJSON = string(data)
	}

	// Convert YAML values to a Go map, then to JSON for gojsonschema
	var valuesMap any
	if err := yaml.Unmarshal([]byte(valuesYAML), &valuesMap); err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"yaml_parse_error",
			"failed to parse values YAML: "+err.Error(),
			"Ensure the values YAML is valid. Validate with: helm lint --values <file>",
		).JSON()), nil
	}

	// yaml.Unmarshal may produce map[string]any with any keys;
	// convert to JSON-compatible types via a JSON round-trip.
	valuesJSONStr, err := jsonRoundTrip(valuesMap)
	if err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"values_conversion_error",
			"failed to convert values to JSON: "+err.Error(),
			"Ensure the values YAML contains only JSON-compatible types",
		).JSON()), nil
	}

	schemaLoader := gojsonschema.NewStringLoader(schemaJSON)
	documentLoader := gojsonschema.NewStringLoader(valuesJSONStr)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"validation_error",
			"schema validation failed to run: "+err.Error(),
			"Ensure schema_json is a valid JSON Schema document",
		).JSON()), nil
	}

	var valErrs []validationError
	for _, e := range result.Errors() {
		valErrs = append(valErrs, validationError{
			Path:    e.Field(),
			Message: e.Description(),
		})
	}
	if valErrs == nil {
		valErrs = []validationError{}
	}

	res := validateValuesResult{
		Valid:  result.Valid(),
		Errors: valErrs,
	}
	b, _ := json.Marshal(res)
	return mcp.NewToolResultText(string(b)), nil
}

// jsonRoundTrip converts a value through JSON marshal/unmarshal to ensure
// all map keys are strings (gojsonschema requirement).
func jsonRoundTrip(v any) (string, error) {
	b, err := json.Marshal(convertYAMLMap(v))
	if err != nil {
		return "", fmt.Errorf("json marshal: %w", err)
	}
	return string(b), nil
}

// convertYAMLMap recursively converts map[any]any (produced by
// yaml.v3) to map[string]any so it can be JSON-marshalled.
func convertYAMLMap(v any) any {
	switch val := v.(type) {
	case map[any]any:
		m := make(map[string]any, len(val))
		for k, v2 := range val {
			m[fmt.Sprintf("%v", k)] = convertYAMLMap(v2)
		}
		return m
	case map[string]any:
		m := make(map[string]any, len(val))
		for k, v2 := range val {
			m[k] = convertYAMLMap(v2)
		}
		return m
	case []any:
		s := make([]any, len(val))
		for i, v2 := range val {
			s[i] = convertYAMLMap(v2)
		}
		return s
	default:
		return v
	}
}
