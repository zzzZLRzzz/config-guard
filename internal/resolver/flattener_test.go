package resolver

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFlattenDir_SimpleMap(t *testing.T) {
	dir := t.TempDir()
	content := `
image:
  repository: registry.example.com/app
  tag: v1.0.0
replicas: 3
`
	if err := os.WriteFile(filepath.Join(dir, "values.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	flat, err := FlattenDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		key   string
		value interface{}
	}{
		{"image.repository", "registry.example.com/app"},
		{"image.tag", "v1.0.0"},
		{"replicas", 3},
	}

	for _, tt := range tests {
		if v, ok := flat.Values[tt.key]; !ok {
			t.Errorf("missing key %s", tt.key)
		} else if v != tt.value {
			t.Errorf("key %s: got %v, want %v", tt.key, v, tt.value)
		}
	}
}

func TestFlattenDir_NestedMap(t *testing.T) {
	dir := t.TempDir()
	content := `
spring:
  datasource:
    url: jdbc:mysql://localhost:3306/db
    username: root
  redis:
    host: localhost
`
	if err := os.WriteFile(filepath.Join(dir, "application.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	flat, err := FlattenDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{
		"spring.datasource.url",
		"spring.datasource.username",
		"spring.redis.host",
	}

	for _, key := range expected {
		if _, ok := flat.Values[key]; !ok {
			t.Errorf("missing key %s", key)
		}
	}
}

func TestFlattenDir_MultipleFiles(t *testing.T) {
	dir := t.TempDir()

	values := `
image:
  repository: registry.example.com/app
  tag: v1.0.0
`
	application := `
spring:
  application:
    name: myapp
`
	if err := os.WriteFile(filepath.Join(dir, "values.yaml"), []byte(values), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "application.yaml"), []byte(application), 0644); err != nil {
		t.Fatal(err)
	}

	flat, err := FlattenDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(flat.Values) != 3 {
		t.Errorf("expected 3 keys, got %d", len(flat.Values))
	}

	if src, ok := flat.KeySource["image.repository"]; !ok || src != "values.yaml" {
		t.Errorf("image.repository should come from values.yaml")
	}
	if src, ok := flat.KeySource["spring.application.name"]; !ok || src != "application.yaml" {
		t.Errorf("spring.application.name should come from application.yaml")
	}
}

func TestFlattenDir_ArrayFlattening(t *testing.T) {
	dir := t.TempDir()
	content := `
env:
  - name: ENV_VAR
    value: test
  - name: ANOTHER
    value: val
`
	if err := os.WriteFile(filepath.Join(dir, "values.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	flat, err := FlattenDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{
		"env[0].name",
		"env[0].value",
		"env[1].name",
		"env[1].value",
	}

	for _, key := range expected {
		if _, ok := flat.Values[key]; !ok {
			t.Errorf("missing key %s", key)
		}
	}
}

func TestFlattenDir_EmptyAndCommentOnlyFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "empty.yaml"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "comments.yaml"), []byte("# just a comment\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "values.yaml"), []byte("a: 1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	flat, err := FlattenDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(flat.Values) != 1 {
		t.Errorf("expected only key a, got %d keys: %v", len(flat.Values), flat.Values)
	}
	if _, has := flat.Values[""]; has {
		t.Errorf("empty/comment-only file injected phantom key %q", "")
	}
}

func TestFlattenDir_MultipleDocumentsRejected(t *testing.T) {
	dir := t.TempDir()
	content := "a: 1\n---\nb: 2\n"
	if err := os.WriteFile(filepath.Join(dir, "multi.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := FlattenDir(dir)
	if err == nil {
		t.Fatal("expected error for multiple YAML documents")
	}
}
