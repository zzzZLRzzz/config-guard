package config

import "fmt"

func MergeRules(base, override []*Rule) ([]*Rule, error) {
	byID := make(map[string]*Rule)
	order := make([]string, 0)

	for _, r := range base {
		byID[r.ID] = r
		order = append(order, r.ID)
	}

	for _, r := range override {
		if _, exists := byID[r.ID]; exists {
			byID[r.ID] = r
		} else {
			byID[r.ID] = r
			order = append(order, r.ID)
		}
	}

	result := make([]*Rule, 0, len(order))
	for _, id := range order {
		result = append(result, byID[id])
	}
	return result, nil
}

func MergeWhitelist(base, override Whitelist) Whitelist {
	return Whitelist{
		Correctness: append(base.Correctness, override.Correctness...),
		Diff:        append(base.Diff, override.Diff...),
	}
}

func MergeRuleFiles(base, override *RuleFile) (*RuleFile, error) {
	rules, err := MergeRules(base.Rules, override.Rules)
	if err != nil {
		return nil, fmt.Errorf("merge rules: %w", err)
	}
	wl := MergeWhitelist(base.Whitelist, override.Whitelist)
	return &RuleFile{Rules: rules, Whitelist: wl}, nil
}
