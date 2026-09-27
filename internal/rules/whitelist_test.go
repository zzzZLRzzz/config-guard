package rules

import (
	"config-guard/internal/config"
	"testing"
)

func TestWhitelistMatcher_Correctness(t *testing.T) {
	wl := config.Whitelist{
		Correctness: []config.CorrectnessWhitelist{
			{Rule: "rule1", Instance: "uat-aps1"},
			{Rule: "rule2", Instance: ""},
		},
	}

	matcher := NewWhitelistMatcher(wl)

	if !matcher.IsCorrectnessWhitelisted("rule1", "uat-aps1") {
		t.Errorf("rule1 should be whitelisted for uat-aps1")
	}

	if matcher.IsCorrectnessWhitelisted("rule1", "staging-aps1") {
		t.Errorf("rule1 should not be whitelisted for staging-aps1")
	}

	if !matcher.IsCorrectnessWhitelisted("rule2", "uat-aps1") {
		t.Errorf("rule2 should be whitelisted for all instances")
	}

	if !matcher.IsCorrectnessWhitelisted("rule2", "staging-aps1") {
		t.Errorf("rule2 should be whitelisted for all instances")
	}
}

func TestWhitelistMatcher_Diff_ExactKey(t *testing.T) {
	wl := config.Whitelist{
		Diff: []config.DiffWhitelist{
			{Key: "spring.profiles.active", Mode: config.ModeAllowDiff, Reason: "test"},
		},
	}

	matcher := NewWhitelistMatcher(wl)

	if mode, ok := matcher.IsDiffWhitelisted("spring.profiles.active", "uat-aps1", "staging-aps1"); !ok {
		t.Errorf("spring.profiles.active should be whitelisted")
	} else if mode != config.ModeAllowDiff {
		t.Errorf("mode should be allow_diff, got %s", mode)
	}

	if _, ok := matcher.IsDiffWhitelisted("spring.datasource.url", "uat-aps1", "staging-aps1"); ok {
		t.Errorf("spring.datasource.url should not be whitelisted")
	}
}

func TestWhitelistMatcher_Diff_PrefixMatch(t *testing.T) {
	wl := config.Whitelist{
		Diff: []config.DiffWhitelist{
			{Key: "resources", Mode: config.ModeAllowDiff, Reason: "test"},
			{Key: "logging.level", Mode: config.ModeAllowDiff, Reason: "test"},
		},
	}

	matcher := NewWhitelistMatcher(wl)

	tests := []struct {
		key      string
		expected bool
	}{
		{"resources.requests.cpu", true},
		{"resources.limits.memory", true},
		{"resources", true},
		{"logging.level.root", true},
		{"logging.level.com.example", true},
		{"logging.file", false},
		{"other.key", false},
	}

	for _, tt := range tests {
		if _, ok := matcher.IsDiffWhitelisted(tt.key, "uat-aps1", "staging-aps1"); ok != tt.expected {
			t.Errorf("key %s: expected whitelisted=%v, got %v", tt.key, tt.expected, ok)
		}
	}
}

func TestWhitelistMatcher_Diff_WithPair(t *testing.T) {
	wl := config.Whitelist{
		Diff: []config.DiffWhitelist{
			{
				Key:    "replicas",
				Mode:   "allow_diff",
				Pair:   []string{"uat-aps1", "staging-aps1"},
				Reason: "test",
			},
		},
	}

	matcher := NewWhitelistMatcher(wl)

	if _, ok := matcher.IsDiffWhitelisted("replicas", "uat-aps1", "staging-aps1"); !ok {
		t.Errorf("replicas should be whitelisted for uat-aps1 vs staging-aps1")
	}

	if _, ok := matcher.IsDiffWhitelisted("replicas", "staging-aps1", "uat-aps1"); !ok {
		t.Errorf("replicas should be whitelisted for staging-aps1 vs uat-aps1 (order independent)")
	}

	if _, ok := matcher.IsDiffWhitelisted("replicas", "uat-aps1", "prod-aps1"); ok {
		t.Errorf("replicas should not be whitelisted for uat-aps1 vs prod-aps1")
	}
}

func TestWhitelistMatcher_Diff_AllowMissing(t *testing.T) {
	wl := config.Whitelist{
		Diff: []config.DiffWhitelist{
			{Key: "optional.field", Mode: config.ModeAllowMissing, Reason: "test"},
		},
	}

	matcher := NewWhitelistMatcher(wl)

	if mode, ok := matcher.IsDiffWhitelisted("optional.field", "uat-aps1", "staging-aps1"); !ok {
		t.Errorf("optional.field should be whitelisted")
	} else if mode != config.ModeAllowMissing {
		t.Errorf("mode should be allow_missing, got %s", mode)
	}
}
