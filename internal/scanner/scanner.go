package scanner

import (
	"fmt"

	"config-guard/internal/config"
	"config-guard/internal/resolver"
	"config-guard/internal/rules"
)

type CorrectnessResult struct {
	Application string         `json:"application"`
	Mode        string         `json:"mode"`
	Instances   []string       `json:"instances"`
	Issues      []rules.Issue  `json:"issues"`
	Summary     Summary        `json:"summary"`
}

type Summary struct {
	Total  int            `json:"total"`
	ByRule map[string]int `json:"by_rule,omitempty"`
}

type DiffResult struct {
	Application string      `json:"application"`
	Mode        string      `json:"mode"`
	Pair        [2]string   `json:"pair"`
	Issues      []DiffIssue `json:"issues"`
	Summary     DiffSummary `json:"summary"`
}

type DiffIssue struct {
	Key   string      `json:"key"`
	Left  interface{} `json:"left"`
	Right interface{} `json:"right"`
	Type  string      `json:"type"`
}

type DiffSummary struct {
	Total        int `json:"total"`
	ValueDiff    int `json:"value_diff"`
	MissingLeft  int `json:"missing_left"`
	MissingRight int `json:"missing_right"`
}

func RunCorrectness(rsv resolver.Resolver, app string, instances []string, merged *config.RuleFile) (*CorrectnessResult, error) {
	result := &CorrectnessResult{
		Application: app,
		Mode:        "correctness",
		Instances:   instances,
		Issues:      make([]rules.Issue, 0),
		Summary:     Summary{ByRule: make(map[string]int)},
	}

	matcher := rules.NewWhitelistMatcher(merged.Whitelist)

	for _, inst := range instances {
		resolved, err := rsv.Resolve(inst, app)
		if err != nil {
			return nil, fmt.Errorf("resolve %s/%s: %w", app, inst, err)
		}

		for _, rule := range merged.Rules {
			if matcher.IsCorrectnessWhitelisted(rule.ID, inst) {
				continue
			}
			issue := rules.Evaluate(rule, resolved.Values, resolved.KeySource, resolved.Labels, inst)
			if issue != nil {
				result.Issues = append(result.Issues, *issue)
				result.Summary.ByRule[rule.ID]++
			}
		}
	}

	result.Summary.Total = len(result.Issues)
	return result, nil
}

func RunDiff(rsv resolver.Resolver, app string, left, right string, merged *config.RuleFile) (*DiffResult, error) {
	result := &DiffResult{
		Application: app,
		Mode:        "diff",
		Pair:        [2]string{left, right},
		Issues:      make([]DiffIssue, 0),
	}

	leftResolved, err := rsv.Resolve(left, app)
	if err != nil {
		return nil, fmt.Errorf("resolve %s/%s: %w", app, left, err)
	}
	rightResolved, err := rsv.Resolve(right, app)
	if err != nil {
		return nil, fmt.Errorf("resolve %s/%s: %w", app, right, err)
	}

	matcher := rules.NewWhitelistMatcher(merged.Whitelist)

	allKeys := make(map[string]bool)
	for k := range leftResolved.Values {
		allKeys[k] = true
	}
	for k := range rightResolved.Values {
		allKeys[k] = true
	}

	for key := range allKeys {
		lv, lok := leftResolved.Values[key]
		rv, rok := rightResolved.Values[key]

		if lok && !rok {
			if mode, ok := matcher.IsDiffWhitelisted(key, left, right); ok && mode == config.ModeAllowMissing {
				continue
			}
			result.Issues = append(result.Issues, DiffIssue{Key: key, Left: lv, Right: nil, Type: "missing_right"})
			result.Summary.MissingRight++
			continue
		}
		if !lok && rok {
			if mode, ok := matcher.IsDiffWhitelisted(key, left, right); ok && mode == config.ModeAllowMissing {
				continue
			}
			result.Issues = append(result.Issues, DiffIssue{Key: key, Left: nil, Right: rv, Type: "missing_left"})
			result.Summary.MissingLeft++
			continue
		}

		ls := fmt.Sprintf("%v", lv)
		rs := fmt.Sprintf("%v", rv)
		if ls != rs {
			if mode, ok := matcher.IsDiffWhitelisted(key, left, right); ok && mode == config.ModeAllowDiff {
				continue
			}
			result.Issues = append(result.Issues, DiffIssue{Key: key, Left: lv, Right: rv, Type: "value_diff"})
			result.Summary.ValueDiff++
		}
	}

	result.Summary.Total = len(result.Issues)
	return result, nil
}
