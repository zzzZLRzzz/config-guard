package rules

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"config-guard/internal/config"
	"config-guard/internal/resolver"
)

type Issue struct {
	Instance string `json:"instance"`
	Source   string `json:"source"`
	Key      string `json:"key"`
	Current  string `json:"current,omitempty"`
	Expected string `json:"expected,omitempty"`
	RuleID   string `json:"rule"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

func Evaluate(rule *config.Rule, values map[string]interface{}, keySource map[string]string, labels map[string]string, instanceName string) *Issue {
	switch rule.Type {
	case config.TypeRequired:
		return evalRequired(rule, values, keySource, instanceName)
	case config.TypeEnum:
		return evalEnum(rule, values, keySource, instanceName)
	case config.TypeRegex:
		return evalRegex(rule, values, keySource, instanceName)
	case config.TypeEquals:
		return evalEquals(rule, values, keySource, instanceName)
	case config.TypeCompare:
		return evalCompare(rule, values, keySource, labels, instanceName)
	case config.TypeDerive:
		return evalDerive(rule, values, keySource, labels, instanceName)
	default:
		return nil
	}
}

func evalRequired(rule *config.Rule, values map[string]interface{}, keySource map[string]string, inst string) *Issue {
	if _, ok := values[rule.Key]; !ok {
		return &Issue{
			Instance: inst,
			Key:      rule.Key,
			RuleID:   rule.ID,
			Severity: "error",
			Message:  fmt.Sprintf("key %s is required but missing", rule.Key),
		}
	}
	return nil
}

func evalEnum(rule *config.Rule, values map[string]interface{}, keySource map[string]string, inst string) *Issue {
	val, ok := values[rule.Key]
	if !ok {
		return nil
	}
	s := fmt.Sprintf("%v", val)
	for _, allowed := range rule.Values {
		if s == allowed {
			return nil
		}
	}
	return &Issue{
		Instance: inst,
		Source:   keySource[rule.Key],
		Key:      rule.Key,
		Current:  s,
		Expected: strings.Join(rule.Values, ", "),
		RuleID:   rule.ID,
		Severity: "error",
		Message:  fmt.Sprintf("key %s value %s not in allowed values [%s]", rule.Key, s, strings.Join(rule.Values, ", ")),
	}
}

func evalRegex(rule *config.Rule, values map[string]interface{}, keySource map[string]string, inst string) *Issue {
	val, ok := values[rule.Key]
	if !ok {
		return nil
	}
	s := fmt.Sprintf("%v", val)

	// CompiledPattern is normally populated by Rule.Validate(), but rules
	// constructed in code may skip validation, so compile lazily here to
	// avoid a nil-pointer panic.
	cp := rule.CompiledPattern
	if cp == nil {
		var err error
		cp, err = regexp.Compile(rule.Pattern)
		if err != nil {
			return &Issue{
				Instance: inst,
				Key:      rule.Key,
				RuleID:   rule.ID,
				Severity: "error",
				Message:  fmt.Sprintf("invalid regex pattern %s: %v", rule.Pattern, err),
			}
		}
	}
	if !cp.MatchString(s) {
		return &Issue{
			Instance: inst,
			Source:   keySource[rule.Key],
			Key:      rule.Key,
			Current:  s,
			Expected: rule.Description,
			RuleID:   rule.ID,
			Severity: "error",
			Message:  fmt.Sprintf("key %s value %s does not match pattern %s", rule.Key, s, rule.Pattern),
		}
	}
	return nil
}

func evalEquals(rule *config.Rule, values map[string]interface{}, keySource map[string]string, inst string) *Issue {
	val, ok := values[rule.Key]
	if !ok {
		return nil
	}
	expected := fmt.Sprintf("%v", rule.Value)
	actual := fmt.Sprintf("%v", val)
	if actual != expected {
		return &Issue{
			Instance: inst,
			Source:   keySource[rule.Key],
			Key:      rule.Key,
			Current:  actual,
			Expected: expected,
			RuleID:   rule.ID,
			Severity: "error",
			Message:  fmt.Sprintf("key %s expected %v but got %v", rule.Key, rule.Value, val),
		}
	}
	return nil
}

func evalCompare(rule *config.Rule, values map[string]interface{}, keySource map[string]string, labels map[string]string, inst string) *Issue {
	leftVal, ok := values[rule.Key]
	if !ok {
		return nil
	}

	rightVal, err := resolveCompareRight(rule, values, labels, inst)
	if err != nil {
		return err
	}

	return compareValues(rule, leftVal, rightVal, keySource, inst)
}

func resolveCompareRight(rule *config.Rule, values map[string]interface{}, labels map[string]string, inst string) (interface{}, *Issue) {
	if rule.Target != "" {
		v, ok := values[rule.Target]
		if !ok {
			return nil, &Issue{
				Instance: inst,
				Key:      rule.Key,
				RuleID:   rule.ID,
				Severity: "error",
				Message:  fmt.Sprintf("compare target %s not found", rule.Target),
			}
		}
		return v, nil
	}
	if rule.TargetValue != nil {
		return rule.TargetValue, nil
	}
	v, ok := labels[rule.TargetLabel]
	if !ok {
		return nil, &Issue{
			Instance: inst,
			Key:      rule.Key,
			RuleID:   rule.ID,
			Severity: "error",
			Message:  fmt.Sprintf("label %s not found", rule.TargetLabel),
		}
	}
	return v, nil
}

func compareValues(rule *config.Rule, left, right interface{}, keySource map[string]string, inst string) *Issue {
	leftStr := fmt.Sprintf("%v", left)
	rightStr := fmt.Sprintf("%v", right)

	switch rule.Op {
	case "==":
		if leftStr == rightStr {
			return nil
		}
	case "!=":
		if leftStr != rightStr {
			return nil
		}
	case ">", ">=", "<", "<=":
		lf, lok := toFloat64(left)
		rf, rok := toFloat64(right)
		if !lok || !rok {
			return &Issue{
				Instance: inst,
				Source:   keySource[rule.Key],
				Key:      rule.Key,
				Current:  leftStr,
				RuleID:   rule.ID,
				Severity: "error",
				Message:  fmt.Sprintf("cannot compare non-numeric values: %v %s %v", left, rule.Op, right),
			}
		}
		rf += rule.Offset
		switch rule.Op {
		case ">":
			if lf > rf {
				return nil
			}
		case ">=":
			if lf >= rf {
				return nil
			}
		case "<":
			if lf < rf {
				return nil
			}
		case "<=":
			if lf <= rf {
				return nil
			}
		}
	}

	offsetStr := ""
	if rule.Offset != 0 {
		offsetStr = fmt.Sprintf(" + %v", rule.Offset)
	}
	return &Issue{
		Instance: inst,
		Source:   keySource[rule.Key],
		Key:      rule.Key,
		Current:  leftStr,
		Expected: fmt.Sprintf("%s %s %v%s", rule.Key, rule.Op, right, offsetStr),
		RuleID:   rule.ID,
		Severity: "error",
		Message:  fmt.Sprintf("compare failed: %v %s %v%s", left, rule.Op, right, offsetStr),
	}
}

func evalDerive(rule *config.Rule, values map[string]interface{}, keySource map[string]string, labels map[string]string, inst string) *Issue {
	fromVal, err := resolver.ResolveFrom(rule.From, labels)
	if err != nil {
		return &Issue{
			Instance: inst,
			Key:      rule.Key,
			RuleID:   rule.ID,
			Severity: "error",
			Message:  fmt.Sprintf("derive from %s failed: %v", rule.From, err),
		}
	}

	fromStr := fmt.Sprintf("%v", fromVal)
	expected, ok := rule.Map[fromStr]
	if !ok {
		return &Issue{
			Instance: inst,
			Key:      rule.Key,
			RuleID:   rule.ID,
			Severity: "error",
			Message:  fmt.Sprintf("derive map has no entry for %s=%s", rule.From, fromStr),
		}
	}

	actual, ok := values[rule.Key]
	if !ok {
		return &Issue{
			Instance: inst,
			Key:      rule.Key,
			Expected: fmt.Sprintf("%v", expected),
			RuleID:   rule.ID,
			Severity: "error",
			Message:  fmt.Sprintf("key %s is missing, expected %v (derived from %s=%s)", rule.Key, expected, rule.From, fromStr),
		}
	}

	actualStr := fmt.Sprintf("%v", actual)
	expectedStr := fmt.Sprintf("%v", expected)
	if actualStr != expectedStr {
		return &Issue{
			Instance: inst,
			Source:   keySource[rule.Key],
			Key:      rule.Key,
			Current:  actualStr,
			Expected: expectedStr,
			RuleID:   rule.ID,
			Severity: "error",
			Message:  fmt.Sprintf("key %s expected %v (derived from %s=%s) but got %v", rule.Key, expected, rule.From, fromStr, actual),
		}
	}
	return nil
}

func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case string:
		f, err := strconv.ParseFloat(n, 64)
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}
