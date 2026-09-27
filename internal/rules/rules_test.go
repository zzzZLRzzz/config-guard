package rules

import (
	"regexp"

	"config-guard/internal/config"
	"testing"
)

func TestEvaluate_Required(t *testing.T) {
	values := map[string]interface{}{
		"key1": "value1",
	}
	keySource := map[string]string{"key1": "test.yaml"}
	labels := map[string]string{}

	rule := &config.Rule{ID: "test", Type: config.TypeRequired, Key: "key1"}
	if issue := Evaluate(rule, values, keySource, labels, "test-instance"); issue != nil {
		t.Errorf("required rule should pass for existing key")
	}

	rule = &config.Rule{ID: "test", Type: config.TypeRequired, Key: "nonexistent"}
	if issue := Evaluate(rule, values, keySource, labels, "test-instance"); issue == nil {
		t.Errorf("required rule should fail for missing key")
	}
}

func TestEvaluate_Enum(t *testing.T) {
	values := map[string]interface{}{
		"env": "uat",
	}
	keySource := map[string]string{"env": "test.yaml"}
	labels := map[string]string{}

	rule := &config.Rule{ID: "test", Type: config.TypeEnum, Key: "env", Values: []string{"uat", "staging", "prod"}}
	if issue := Evaluate(rule, values, keySource, labels, "test-instance"); issue != nil {
		t.Errorf("enum rule should pass for valid value")
	}

	rule = &config.Rule{ID: "test", Type: config.TypeEnum, Key: "env", Values: []string{"staging", "prod"}}
	if issue := Evaluate(rule, values, keySource, labels, "test-instance"); issue == nil {
		t.Errorf("enum rule should fail for invalid value")
	}
}

func TestEvaluate_Regex(t *testing.T) {
	values := map[string]interface{}{
		"image": "registry.example.com/app",
	}
	keySource := map[string]string{"image": "test.yaml"}
	labels := map[string]string{}

	rule := &config.Rule{ID: "test", Type: config.TypeRegex, Key: "image", Pattern: "^registry\\.example\\.com/[a-z]+$", CompiledPattern: regexp.MustCompile("^registry\\.example\\.com/[a-z]+$")}
	if issue := Evaluate(rule, values, keySource, labels, "test-instance"); issue != nil {
		t.Errorf("regex rule should pass for matching value")
	}

	rule = &config.Rule{ID: "test", Type: config.TypeRegex, Key: "image", Pattern: "^other\\.com/", CompiledPattern: regexp.MustCompile("^other\\.com/")}
	if issue := Evaluate(rule, values, keySource, labels, "test-instance"); issue == nil {
		t.Errorf("regex rule should fail for non-matching value")
	}
}

func TestEvaluate_Equals(t *testing.T) {
	values := map[string]interface{}{
		"region": "aps1",
	}
	keySource := map[string]string{"region": "test.yaml"}
	labels := map[string]string{}

	rule := &config.Rule{ID: "test", Type: config.TypeEquals, Key: "region", Value: "aps1"}
	if issue := Evaluate(rule, values, keySource, labels, "test-instance"); issue != nil {
		t.Errorf("equals rule should pass for matching value")
	}

	rule = &config.Rule{ID: "test", Type: config.TypeEquals, Key: "region", Value: "use1"}
	if issue := Evaluate(rule, values, keySource, labels, "test-instance"); issue == nil {
		t.Errorf("equals rule should fail for non-matching value")
	}
}

func TestEvaluate_Compare_WithValue(t *testing.T) {
	values := map[string]interface{}{
		"autoscaling.maxReplicas": 10,
		"autoscaling.minReplicas": 3,
	}
	keySource := map[string]string{
		"autoscaling.maxReplicas": "test.yaml",
		"autoscaling.minReplicas": "test.yaml",
	}
	labels := map[string]string{}

	rule := &config.Rule{ID: "test", Type: config.TypeCompare, Key: "autoscaling.maxReplicas", Op: ">", Target: "autoscaling.minReplicas"}
	if issue := Evaluate(rule, values, keySource, labels, "test-instance"); issue != nil {
		t.Errorf("compare rule should pass: 10 > 3")
	}

	rule = &config.Rule{ID: "test", Type: config.TypeCompare, Key: "autoscaling.minReplicas", Op: ">", Target: "autoscaling.maxReplicas"}
	if issue := Evaluate(rule, values, keySource, labels, "test-instance"); issue == nil {
		t.Errorf("compare rule should fail: 3 > 10 is false")
	}
}

func TestEvaluate_Compare_WithLabel(t *testing.T) {
	values := map[string]interface{}{
		"region": "aps1",
	}
	keySource := map[string]string{"region": "test.yaml"}
	labels := map[string]string{"region": "aps1"}

	rule := &config.Rule{ID: "test", Type: config.TypeCompare, Key: "region", Op: "==", TargetLabel: "region"}
	if issue := Evaluate(rule, values, keySource, labels, "test-instance"); issue != nil {
		t.Errorf("compare rule should pass: region == aps1")
	}

	labels = map[string]string{"region": "use1"}
	if issue := Evaluate(rule, values, keySource, labels, "test-instance"); issue == nil {
		t.Errorf("compare rule should fail: aps1 != use1")
	}
}

func TestEvaluate_Compare_WithTargetValue(t *testing.T) {
	values := map[string]interface{}{
		"replicas": 5,
	}
	keySource := map[string]string{"replicas": "test.yaml"}
	labels := map[string]string{}

	rule := &config.Rule{ID: "test", Type: config.TypeCompare, Key: "replicas", Op: "==", TargetValue: "5"}
	if issue := Evaluate(rule, values, keySource, labels, "test-instance"); issue != nil {
		t.Errorf("compare rule should pass: replicas == 5")
	}

	rule = &config.Rule{ID: "test", Type: config.TypeCompare, Key: "replicas", Op: ">", TargetValue: "10"}
	if issue := Evaluate(rule, values, keySource, labels, "test-instance"); issue == nil {
		t.Errorf("compare rule should fail: 5 > 10 is false")
	}
}

func TestEvaluate_Derive(t *testing.T) {
	values := map[string]interface{}{
		"port": 8080,
	}
	keySource := map[string]string{"port": "test.yaml"}
	labels := map[string]string{"region": "aps1"}

	rule := &config.Rule{
		ID:   "test",
		Type: config.TypeDerive,
		Key:  "port",
		From: "instance.labels.region",
		Map: map[string]interface{}{
			"aps1": 8080,
			"use1": 9090,
		},
	}

	if issue := Evaluate(rule, values, keySource, labels, "test-instance"); issue != nil {
		t.Errorf("derive rule should pass: port matches derived value for aps1")
	}

	labels = map[string]string{"region": "use1"}
	if issue := Evaluate(rule, values, keySource, labels, "test-instance"); issue == nil {
		t.Errorf("derive rule should fail: port 8080 != derived value 9090 for use1")
	}
}

func TestEvaluate_Compare_WithQuotedNumbers(t *testing.T) {
	// YAML "3" parses as string; numeric operators must still handle it.
	values := map[string]interface{}{
		"replicas": "3",
	}
	keySource := map[string]string{"replicas": "test.yaml"}
	labels := map[string]string{}

	rule := &config.Rule{ID: "test", Type: config.TypeCompare, Key: "replicas", Op: ">", TargetValue: "2"}
	if issue := Evaluate(rule, values, keySource, labels, "test-instance"); issue != nil {
		t.Errorf("quoted number should compare numerically: %v", issue.Message)
	}

	rule = &config.Rule{ID: "test", Type: config.TypeCompare, Key: "replicas", Op: "<", TargetValue: 4}
	if issue := Evaluate(rule, values, keySource, labels, "test-instance"); issue != nil {
		t.Errorf("quoted number should compare numerically: %v", issue.Message)
	}

	rule = &config.Rule{ID: "test", Type: config.TypeCompare, Key: "replicas", Op: ">", TargetValue: 5}
	if issue := Evaluate(rule, values, keySource, labels, "test-instance"); issue == nil {
		t.Errorf("3 > 5 should fail")
	}
}
