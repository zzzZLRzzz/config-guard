package resolver

import (
	"testing"
)

func TestParseLabels_ValidInstance(t *testing.T) {
	tests := []struct {
		instance string
		env      string
		region   string
	}{
		{"uat-aps1", "uat", "aps1"},
		{"staging-use1", "staging", "use1"},
		{"prod-euw1", "prod", "euw1"},
		{"dev-aps2", "dev", "aps2"},
	}

	for _, tt := range tests {
		labels, err := ParseLabels(tt.instance)
		if err != nil {
			t.Errorf("ParseLabels(%s) error: %v", tt.instance, err)
			continue
		}
		if labels["env"] != tt.env {
			t.Errorf("ParseLabels(%s) env = %s, want %s", tt.instance, labels["env"], tt.env)
		}
		if labels["region"] != tt.region {
			t.Errorf("ParseLabels(%s) region = %s, want %s", tt.instance, labels["region"], tt.region)
		}
	}
}

func TestParseLabels_InvalidInstance(t *testing.T) {
	tests := []string{
		"uat",
		"uataps1",
		"",
	}

	for _, instance := range tests {
		_, err := ParseLabels(instance)
		if err == nil {
			t.Errorf("ParseLabels(%s) should return error", instance)
		}
	}
}

func TestResolveFrom_InstanceLabels(t *testing.T) {
	labels := map[string]string{
		"env":    "uat",
		"region": "aps1",
		"app":    "svcA",
	}

	tests := []struct {
		from     string
		expected interface{}
		hasError bool
	}{
		{"instance.labels.env", "uat", false},
		{"instance.labels.region", "aps1", false},
		{"instance.labels.app", "svcA", false},
		{"instance.labels.nonexistent", nil, true},
	}

	for _, tt := range tests {
		val, err := ResolveFrom(tt.from, labels)
		if tt.hasError && err == nil {
			t.Errorf("ResolveFrom(%s) should return error", tt.from)
		}
		if !tt.hasError && err != nil {
			t.Errorf("ResolveFrom(%s) unexpected error: %v", tt.from, err)
		}
		if val != tt.expected {
			t.Errorf("ResolveFrom(%s) = %v, want %v", tt.from, val, tt.expected)
		}
	}
}
