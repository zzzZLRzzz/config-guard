package resolver

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirResolver_Resolve(t *testing.T) {
	basePath := t.TempDir()
	appDir := filepath.Join(basePath, "myapp", "uat-aps1")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := `
image:
  repository: registry.example.com/myapp
  tag: v1.0.0
replicas: 3
`
	if err := os.WriteFile(filepath.Join(appDir, "values.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	rsv := NewDirResolver(basePath)
	resolved, err := rsv.Resolve("uat-aps1", "myapp")
	if err != nil {
		t.Fatal(err)
	}

	if resolved.Labels["env"] != "uat" {
		t.Errorf("expected label env=uat, got %s", resolved.Labels["env"])
	}
	if resolved.Labels["region"] != "aps1" {
		t.Errorf("expected label region=aps1, got %s", resolved.Labels["region"])
	}
	if resolved.Labels["app"] != "myapp" {
		t.Errorf("expected label app=myapp, got %s", resolved.Labels["app"])
	}

	if v, ok := resolved.Values["image.repository"]; !ok || v != "registry.example.com/myapp" {
		t.Errorf("expected image.repository=registry.example.com/myapp, got %v", v)
	}
	if v, ok := resolved.Values["replicas"]; !ok || v != 3 {
		t.Errorf("expected replicas=3, got %v", v)
	}
	if src, ok := resolved.KeySource["image.repository"]; !ok || src != "values.yaml" {
		t.Errorf("expected key source values.yaml, got %s", src)
	}
}
