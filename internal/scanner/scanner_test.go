package scanner

import (
	"regexp"

	"config-guard/internal/config"
	"config-guard/internal/resolver"
	"os"
	"path/filepath"
	"testing"
)

func TestRunCorrectness_BasicRules(t *testing.T) {
	basePath := t.TempDir()
	appDir := filepath.Join(basePath, "testapp", "uat-aps1")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := `
image:
  repository: registry.example.com/testapp
  tag: v1.0.0
region: aps1
service:
  type: ClusterIP
`
	if err := os.WriteFile(filepath.Join(appDir, "values.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	rules := &config.RuleFile{
		Rules: []*config.Rule{
			{ID: "image-required", Type: config.TypeRequired, Key: "image.repository"},
			{ID: "region-enum", Type: config.TypeEnum, Key: "region", Values: []string{"aps1", "use1"}},
			{ID: "service-type", Type: config.TypeEnum, Key: "service.type", Values: []string{"LoadBalancer"}},
		},
	}

	result, err := RunCorrectness(resolver.NewDirResolver(basePath), "testapp", []string{"uat-aps1"}, rules)
	if err != nil {
		t.Fatal(err)
	}

	if result.Summary.Total != 1 {
		t.Errorf("expected 1 issue, got %d", result.Summary.Total)
	}

	if len(result.Issues) > 0 && result.Issues[0].RuleID != "service-type" {
		t.Errorf("expected service-type issue, got %s", result.Issues[0].RuleID)
	}
}

func TestRunCorrectness_RegexRule(t *testing.T) {
	basePath := t.TempDir()
	appDir := filepath.Join(basePath, "testapp", "uat-aps1")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := `
image:
  repository: registry.example.com/testapp
  tag: v1.0.0
`
	if err := os.WriteFile(filepath.Join(appDir, "values.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	rules := &config.RuleFile{
		Rules: []*config.Rule{
			{ID: "image-repo", Type: config.TypeRegex, Key: "image.repository", Pattern: "^registry\\.example\\.com/[a-z]+$", CompiledPattern: regexp.MustCompile("^registry\\.example\\.com/[a-z]+$")},
			{ID: "image-tag", Type: config.TypeRegex, Key: "image.tag", Pattern: "^v[0-9]+\\.[0-9]+\\.[0-9]+$", CompiledPattern: regexp.MustCompile("^v[0-9]+\\.[0-9]+\\.[0-9]+$")},
		},
	}

	result, err := RunCorrectness(resolver.NewDirResolver(basePath), "testapp", []string{"uat-aps1"}, rules)
	if err != nil {
		t.Fatal(err)
	}

	if result.Summary.Total != 0 {
		t.Errorf("expected 0 issues, got %d", result.Summary.Total)
	}
}

func TestRunCorrectness_CompareWithLabel(t *testing.T) {
	basePath := t.TempDir()
	appDir := filepath.Join(basePath, "testapp", "uat-aps1")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := `
region: aps1
spring:
  application:
    name: testapp
`
	if err := os.WriteFile(filepath.Join(appDir, "values.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	rules := &config.RuleFile{
		Rules: []*config.Rule{
			{ID: "region-match", Type: config.TypeCompare, Key: "region", Op: "==", TargetLabel: "region"},
			{ID: "app-name-match", Type: config.TypeCompare, Key: "spring.application.name", Op: "==", TargetLabel: "app"},
		},
	}

	result, err := RunCorrectness(resolver.NewDirResolver(basePath), "testapp", []string{"uat-aps1"}, rules)
	if err != nil {
		t.Fatal(err)
	}

	if result.Summary.Total != 0 {
		t.Errorf("expected 0 issues, got %d", result.Summary.Total)
	}
}

func TestRunCorrectness_DeriveRule(t *testing.T) {
	basePath := t.TempDir()
	appDir := filepath.Join(basePath, "testapp", "uat-aps1")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := `
istio:
  gateway:
    port: 11443
`
	if err := os.WriteFile(filepath.Join(appDir, "values.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	rules := &config.RuleFile{
		Rules: []*config.Rule{
			{
				ID:   "gw-port",
				Type: config.TypeDerive,
				Key:  "istio.gateway.port",
				From: "instance.labels.region",
				Map: map[string]interface{}{
					"aps1": 11443,
					"use1": 12443,
				},
			},
		},
	}

	result, err := RunCorrectness(resolver.NewDirResolver(basePath), "testapp", []string{"uat-aps1"}, rules)
	if err != nil {
		t.Fatal(err)
	}

	if result.Summary.Total != 0 {
		t.Errorf("expected 0 issues, got %d", result.Summary.Total)
	}
}

func TestRunDiff_BasicDiff(t *testing.T) {
	basePath := t.TempDir()

	uatDir := filepath.Join(basePath, "testapp", "uat-aps1")
	stagingDir := filepath.Join(basePath, "testapp", "staging-aps1")
	if err := os.MkdirAll(uatDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(stagingDir, 0755); err != nil {
		t.Fatal(err)
	}

	uatContent := `
replicas: 2
logging:
  level: DEBUG
`
	stagingContent := `
replicas: 3
logging:
  level: INFO
`
	if err := os.WriteFile(filepath.Join(uatDir, "values.yaml"), []byte(uatContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stagingDir, "values.yaml"), []byte(stagingContent), 0644); err != nil {
		t.Fatal(err)
	}

	rules := &config.RuleFile{
		Whitelist: config.Whitelist{
			Diff: []config.DiffWhitelist{
				{Key: "logging", Mode: config.ModeAllowDiff, Reason: "test"},
			},
		},
	}

	result, err := RunDiff(resolver.NewDirResolver(basePath), "testapp", "uat-aps1", "staging-aps1", rules)
	if err != nil {
		t.Fatal(err)
	}

	if result.Summary.Total != 1 {
		t.Errorf("expected 1 issue, got %d", result.Summary.Total)
	}

	if len(result.Issues) > 0 && result.Issues[0].Key != "replicas" {
		t.Errorf("expected replicas diff, got %s", result.Issues[0].Key)
	}
}

func TestRunDiff_MissingKey(t *testing.T) {
	basePath := t.TempDir()

	uatDir := filepath.Join(basePath, "testapp", "uat-aps1")
	stagingDir := filepath.Join(basePath, "testapp", "staging-aps1")
	if err := os.MkdirAll(uatDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(stagingDir, 0755); err != nil {
		t.Fatal(err)
	}

	uatContent := `
feature:
  enabled: true
  timeout: 3000
`
	stagingContent := `
feature:
  enabled: false
`
	if err := os.WriteFile(filepath.Join(uatDir, "values.yaml"), []byte(uatContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stagingDir, "values.yaml"), []byte(stagingContent), 0644); err != nil {
		t.Fatal(err)
	}

	rules := &config.RuleFile{}

	result, err := RunDiff(resolver.NewDirResolver(basePath), "testapp", "uat-aps1", "staging-aps1", rules)
	if err != nil {
		t.Fatal(err)
	}

	if result.Summary.MissingRight != 1 {
		t.Errorf("expected 1 missing_right, got %d", result.Summary.MissingRight)
	}
}
