package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	if len(cfg.Applications) == 0 {
		return nil, fmt.Errorf("config %s: no applications defined", path)
	}
	if cfg.RuleFile == "" {
		return nil, fmt.Errorf("config %s: rule_file is required", path)
	}
	return &cfg, nil
}

func LoadRuleFile(path string) (*RuleFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read rule file %s: %w", path, err)
	}
	var rf ruleFileYAML
	if err := yaml.Unmarshal(data, &rf); err != nil {
		return nil, fmt.Errorf("parse rule file %s: %w", path, err)
	}
	if err := rf.Whitelist.Validate(); err != nil {
		return nil, fmt.Errorf("validate whitelist in %s: %w", path, err)
	}
	return rf.toRuleFile(), nil
}

func (g *ruleFileYAML) toRuleFile() *RuleFile {
	rf := &RuleFile{
		Whitelist: g.Whitelist,
	}
	for _, r := range g.Rules.Required {
		rf.Rules = append(rf.Rules, &Rule{ID: r.ID, Type: TypeRequired, Key: r.Key})
	}
	for _, r := range g.Rules.Enum {
		rf.Rules = append(rf.Rules, &Rule{ID: r.ID, Type: TypeEnum, Key: r.Key, Values: r.Values})
	}
	for _, r := range g.Rules.Regex {
		rf.Rules = append(rf.Rules, &Rule{ID: r.ID, Type: TypeRegex, Key: r.Key, Pattern: r.Pattern, Description: r.Description})
	}
	for _, r := range g.Rules.Equals {
		rf.Rules = append(rf.Rules, &Rule{ID: r.ID, Type: TypeEquals, Key: r.Key, Value: r.Value, Description: r.Description})
	}
	for _, r := range g.Rules.Compare {
		rf.Rules = append(rf.Rules, &Rule{
			ID: r.ID, Type: TypeCompare, Key: r.Key, Op: r.Op,
			Target: r.Target, TargetValue: r.TargetValue, TargetLabel: r.TargetLabel,
			Offset: r.Offset, Description: r.Description,
		})
	}
	for _, r := range g.Rules.Derive {
		rf.Rules = append(rf.Rules, &Rule{ID: r.ID, Type: TypeDerive, Key: r.Key, From: r.From, Map: r.Map, Description: r.Description})
	}
	return rf
}

func ValidateRules(rules []*Rule) error {
	seen := make(map[string]bool)
	for _, r := range rules {
		if err := r.Validate(); err != nil {
			return err
		}
		if seen[r.ID] {
			return fmt.Errorf("duplicate rule id %q", r.ID)
		}
		seen[r.ID] = true
	}
	return nil
}
