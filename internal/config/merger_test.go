package config

import (
	"testing"
)

func TestMergeRules_OverrideByID(t *testing.T) {
	global := []*Rule{
		{ID: "rule1", Type: TypeEnum, Key: "key1", Values: []string{"a", "b"}},
		{ID: "rule2", Type: TypeRequired, Key: "key2"},
	}
	app := []*Rule{
		{ID: "rule1", Type: TypeEnum, Key: "key1", Values: []string{"a"}},
		{ID: "rule3", Type: TypeRegex, Key: "key3", Pattern: "^test$"},
	}

	merged, err := MergeRules(global, app)
	if err != nil {
		t.Fatal(err)
	}

	if len(merged) != 3 {
		t.Errorf("expected 3 rules, got %d", len(merged))
	}

	ruleMap := make(map[string]*Rule)
	for _, r := range merged {
		ruleMap[r.ID] = r
	}

	if r, ok := ruleMap["rule1"]; !ok {
		t.Error("rule1 missing")
	} else if len(r.Values) != 1 || r.Values[0] != "a" {
		t.Errorf("rule1 should be overridden, got values %v", r.Values)
	}

	if _, ok := ruleMap["rule2"]; !ok {
		t.Error("rule2 missing")
	}

	if _, ok := ruleMap["rule3"]; !ok {
		t.Error("rule3 missing")
	}
}

func TestMergeWhitelist_Append(t *testing.T) {
	global := Whitelist{
		Correctness: []CorrectnessWhitelist{
			{Rule: "rule1", Instance: "uat-aps1"},
		},
		Diff: []DiffWhitelist{
			{Key: "key1", Mode: ModeAllowDiff, Reason: "reason1"},
		},
	}
	app := Whitelist{
		Correctness: []CorrectnessWhitelist{
			{Rule: "rule2", Instance: "staging-aps1"},
		},
		Diff: []DiffWhitelist{
			{Key: "key2", Mode: ModeAllowMissing, Reason: "reason2"},
		},
	}

	merged := MergeWhitelist(global, app)

	if len(merged.Correctness) != 2 {
		t.Errorf("expected 2 correctness whitelist entries, got %d", len(merged.Correctness))
	}

	if len(merged.Diff) != 2 {
		t.Errorf("expected 2 diff whitelist entries, got %d", len(merged.Diff))
	}
}

func TestMergeRuleFiles_FullMerge(t *testing.T) {
	global := &RuleFile{
		Rules: []*Rule{
			{ID: "rule1", Type: TypeRequired, Key: "key1"},
			{ID: "rule2", Type: TypeEnum, Key: "key2", Values: []string{"a", "b"}},
		},
		Whitelist: Whitelist{
			Diff: []DiffWhitelist{
				{Key: "wl1", Mode: ModeAllowDiff},
			},
		},
	}
	app := &RuleFile{
		Rules: []*Rule{
			{ID: "rule2", Type: TypeEnum, Key: "key2", Values: []string{"a"}},
			{ID: "rule3", Type: TypeRegex, Key: "key3", Pattern: "^test$"},
		},
		Whitelist: Whitelist{
			Diff: []DiffWhitelist{
				{Key: "wl2", Mode: ModeAllowMissing},
			},
		},
	}

	merged, err := MergeRuleFiles(global, app)
	if err != nil {
		t.Fatal(err)
	}

	if len(merged.Rules) != 3 {
		t.Errorf("expected 3 rules, got %d", len(merged.Rules))
	}

	if len(merged.Whitelist.Diff) != 2 {
		t.Errorf("expected 2 diff whitelist entries, got %d", len(merged.Whitelist.Diff))
	}

	ruleMap := make(map[string]*Rule)
	for _, r := range merged.Rules {
		ruleMap[r.ID] = r
	}

	if r, ok := ruleMap["rule2"]; !ok {
		t.Error("rule2 missing")
	} else if len(r.Values) != 1 {
		t.Errorf("rule2 should be overridden to have 1 value, got %d", len(r.Values))
	}
}
