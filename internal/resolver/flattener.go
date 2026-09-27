package resolver

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type FlatResult struct {
	Values    map[string]interface{}
	KeySource map[string]string
}

func FlattenDir(dir string) (*FlatResult, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}

	result := &FlatResult{
		Values:    make(map[string]interface{}),
		KeySource: make(map[string]string),
	}

	files := make([]string, 0)
	for _, e := range entries {
		if !e.IsDir() && (strings.HasSuffix(e.Name(), ".yaml") || strings.HasSuffix(e.Name(), ".yml")) {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, name := range files {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", filepath.Join(dir, name), err)
		}

		// Decode with yaml.Decoder so we can detect empty files and
		// reject files that contain multiple YAML documents.
		dec := yaml.NewDecoder(bytes.NewReader(data))
		var doc interface{}
		if err := dec.Decode(&doc); err != nil {
			if errors.Is(err, io.EOF) {
				continue // empty or comment-only file, contributes no keys
			}
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
		if doc == nil {
			continue // null document, contributes no keys
		}
		if err := dec.Decode(new(interface{})); err == nil {
			return nil, fmt.Errorf("parse %s: multiple YAML documents are not supported, use one document per file", name)
		}

		flat := flattenValue(doc, "")
		for k, v := range flat {
			if _, exists := result.Values[k]; exists {
				return nil, fmt.Errorf("duplicate key %q found in %s and %s", k, result.KeySource[k], name)
			}
			result.Values[k] = v
			result.KeySource[k] = name
		}
	}
	return result, nil
}

func flattenValue(val interface{}, prefix string) map[string]interface{} {
	result := make(map[string]interface{})

	switch v := val.(type) {
	case map[string]interface{}:
		for k, child := range v {
			newKey := joinKey(prefix, k)
			for fk, fv := range flattenValue(child, newKey) {
				result[fk] = fv
			}
		}
	case []interface{}:
		for i, child := range v {
			newKey := fmt.Sprintf("%s[%d]", prefix, i)
			for fk, fv := range flattenValue(child, newKey) {
				result[fk] = fv
			}
		}
	default:
		result[prefix] = v
	}
	return result
}

func joinKey(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}
