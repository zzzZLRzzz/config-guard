package config

import (
	"fmt"
	"regexp"
)

const (
	TypeRequired = "required"
	TypeEnum     = "enum"
	TypeRegex    = "regex"
	TypeEquals   = "equals"
	TypeCompare  = "compare"
	TypeDerive   = "derive"
)

const (
	ModeCorrectness = "correctness"
	ModeDiff        = "diff"
)

const (
	ModeAllowDiff    = "allow_diff"
	ModeAllowMissing = "allow_missing"
)

type Config struct {
	BasePath     string   `yaml:"base_path"`
	Applications []string `yaml:"applications"`
	RuleFile     string   `yaml:"rule_file"`
}

type RuleFile struct {
	Rules     []*Rule   `yaml:"-"`
	Whitelist Whitelist `yaml:"whitelist"`
}

type ruleFileYAML struct {
	Rules     groupedRules `yaml:"rules"`
	Whitelist Whitelist    `yaml:"whitelist"`
}

type groupedRules struct {
	Required []requiredRule `yaml:"required"`
	Enum     []enumRule     `yaml:"enum"`
	Regex    []regexRule    `yaml:"regex"`
	Equals   []equalsRule   `yaml:"equals"`
	Compare  []compareRule  `yaml:"compare"`
	Derive   []deriveRule   `yaml:"derive"`
}

type requiredRule struct {
	ID  string `yaml:"id"`
	Key string `yaml:"key"`
}

type enumRule struct {
	ID     string   `yaml:"id"`
	Key    string   `yaml:"key"`
	Values []string `yaml:"values"`
}

type regexRule struct {
	ID          string `yaml:"id"`
	Key         string `yaml:"key"`
	Pattern     string `yaml:"pattern"`
	Description string `yaml:"description,omitempty"`
}

type equalsRule struct {
	ID          string      `yaml:"id"`
	Key         string      `yaml:"key"`
	Value       interface{} `yaml:"value"`
	Description string      `yaml:"description,omitempty"`
}

type compareRule struct {
	ID          string      `yaml:"id"`
	Key         string      `yaml:"key"`
	Op          string      `yaml:"op"`
	Target      string      `yaml:"target,omitempty"`
	TargetValue interface{} `yaml:"target_value,omitempty"`
	TargetLabel string      `yaml:"target_label,omitempty"`
	Offset      float64     `yaml:"offset,omitempty"`
	Description string      `yaml:"description,omitempty"`
}

type deriveRule struct {
	ID          string                 `yaml:"id"`
	Key         string                 `yaml:"key"`
	From        string                 `yaml:"from"`
	Map         map[string]interface{} `yaml:"map"`
	Description string                 `yaml:"description,omitempty"`
}

type Rule struct {
	ID   string `yaml:"id"`
	Type string `yaml:"type"`
	Key  string `yaml:"key"`
	// Values: allowed values for enum rules.
	Values []string `yaml:"values,omitempty"`
	// Pattern: regex pattern for regex rules.
	Pattern     string `yaml:"pattern,omitempty"`
	Description string `yaml:"description,omitempty"`
	// Value: expected value for equals rules.
	Value interface{} `yaml:"value,omitempty"`
	// Op: comparison operator (==, !=, >, >=, <, <=) for compare rules.
	Op string `yaml:"op,omitempty"`
	// Target: another config key to compare against (compare rules, mutually exclusive with TargetValue/TargetLabel).
	Target string `yaml:"target,omitempty"`
	// TargetValue: literal value to compare against (compare rules, mutually exclusive with Target/TargetLabel).
	TargetValue interface{} `yaml:"target_value,omitempty"`
	// TargetLabel: instance label to compare against (compare rules, mutually exclusive with Target/TargetValue).
	TargetLabel string `yaml:"target_label,omitempty"`
	// Offset: numeric offset added to the right-hand side in compare rules.
	Offset float64 `yaml:"offset,omitempty"`
	// From: source namespace for derive rules (e.g. instance.labels.region).
	From string `yaml:"from,omitempty"`
	// Map: mapping from resolved value to expected config value for derive rules.
	Map map[string]interface{} `yaml:"map,omitempty"`

	CompiledPattern *regexp.Regexp `yaml:"-"`
}

type Whitelist struct {
	Correctness []CorrectnessWhitelist `yaml:"correctness"`
	Diff        []DiffWhitelist        `yaml:"diff"`
}

type CorrectnessWhitelist struct {
	Rule     string `yaml:"rule"`
	Instance string `yaml:"instance,omitempty"`
	Reason   string `yaml:"reason"`
}

type DiffWhitelist struct {
	Key    string   `yaml:"key"`
	Pair   []string `yaml:"pair,omitempty"`
	Mode   string   `yaml:"mode"`
	Reason string   `yaml:"reason"`
}

type ScanRequest struct {
	Application string   `yaml:"application"`
	Mode        string   `yaml:"mode"`
	Instances   []string `yaml:"instances"`
}

func (r *Rule) Validate() error {
	if r.ID == "" {
		return fmt.Errorf("rule id is required")
	}
	if r.Key == "" {
		return fmt.Errorf("rule %q: key is required", r.ID)
	}
	switch r.Type {
	case TypeRequired:
		// no extra fields
	case TypeEnum:
		if len(r.Values) == 0 {
			return fmt.Errorf("rule %q: enum requires values", r.ID)
		}
	case TypeRegex:
		if r.Pattern == "" {
			return fmt.Errorf("rule %q: regex requires pattern", r.ID)
		}
		compiled, err := regexp.Compile(r.Pattern)
		if err != nil {
			return fmt.Errorf("rule %q: invalid regex pattern: %w", r.ID, err)
		}
		r.CompiledPattern = compiled
	case TypeEquals:
		if r.Value == nil {
			return fmt.Errorf("rule %q: equals requires value", r.ID)
		}
	case TypeCompare:
		if r.Op == "" {
			return fmt.Errorf("rule %q: compare requires op", r.ID)
		}
		targetCount := 0
		if r.Target != "" {
			targetCount++
		}
		if r.TargetValue != nil {
			targetCount++
		}
		if r.TargetLabel != "" {
			targetCount++
		}
		if targetCount != 1 {
			return fmt.Errorf("rule %q: compare requires exactly one of target/target_value/target_label", r.ID)
		}
	case TypeDerive:
		if r.From == "" {
			return fmt.Errorf("rule %q: derive requires from", r.ID)
		}
		if len(r.Map) == 0 {
			return fmt.Errorf("rule %q: derive requires map", r.ID)
		}
	default:
		return fmt.Errorf("rule %q: invalid type %q", r.ID, r.Type)
	}
	return nil
}

func (w *Whitelist) Validate() error {
	for i, entry := range w.Diff {
		if entry.Key == "" {
			return fmt.Errorf("diff whitelist[%d]: key is required", i)
		}
		switch entry.Mode {
		case ModeAllowDiff, ModeAllowMissing:
		default:
			return fmt.Errorf("diff whitelist[%d]: invalid mode %q, must be %s or %s", i, entry.Mode, ModeAllowDiff, ModeAllowMissing)
		}
		if len(entry.Pair) != 0 && len(entry.Pair) != 2 {
			return fmt.Errorf("diff whitelist[%d]: pair must be empty or have exactly 2 elements, got %d", i, len(entry.Pair))
		}
	}
	for i, entry := range w.Correctness {
		if entry.Rule == "" {
			return fmt.Errorf("correctness whitelist[%d]: rule is required", i)
		}
	}
	return nil
}
