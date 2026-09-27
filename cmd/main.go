package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"config-guard/internal/config"
	"config-guard/internal/report"
	"config-guard/internal/resolver"
	"config-guard/internal/scanner"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "scan" {
		fmt.Fprintf(os.Stderr, "Usage: config-guard scan [options]\n")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		fmt.Fprintf(os.Stderr, "  --config       path to config.yaml (required)\n")
		fmt.Fprintf(os.Stderr, "  --application  application name (required)\n")
		fmt.Fprintf(os.Stderr, "  --mode         correctness | diff (required)\n")
		fmt.Fprintf(os.Stderr, "  --instances    comma-separated instance names (required)\n")
		fmt.Fprintf(os.Stderr, "  --output       output file path (default: stdout)\n")
		os.Exit(1)
	}

	args := parseArgs(os.Args[2:])

	configPath := args["config"]
	appName := args["application"]
	mode := args["mode"]
	instancesStr := args["instances"]
	outputPath := args["output"]

	if configPath == "" || appName == "" || mode == "" || instancesStr == "" {
		fmt.Fprintf(os.Stderr, "Error: --config, --application, --mode, --instances are all required\n")
		os.Exit(1)
	}

	instances := strings.Split(instancesStr, ",")
	if mode == config.ModeDiff && len(instances) != 2 {
		fmt.Fprintf(os.Stderr, "Error: diff mode requires exactly 2 instances\n")
		os.Exit(1)
	}

	configDir := filepath.Dir(configPath)

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if cfg.BasePath == "" {
		fmt.Fprintf(os.Stderr, "Error: base_path is required in config\n")
		os.Exit(1)
	}

	found := false
	for _, a := range cfg.Applications {
		if a == appName {
			found = true
			break
		}
	}
	if !found {
		fmt.Fprintf(os.Stderr, "Error: application %q not found in config\n", appName)
		os.Exit(1)
	}

	ruleFilePath := filepath.Join(configDir, cfg.RuleFile)
	baseRules, err := config.LoadRuleFile(ruleFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading rules: %v\n", err)
		os.Exit(1)
	}

	appRulePath := filepath.Join(configDir, appName+".yaml")
	var merged *config.RuleFile
	if _, err := os.Stat(appRulePath); err == nil {
		appRules, err := config.LoadRuleFile(appRulePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading app rules: %v\n", err)
			os.Exit(1)
		}
		merged, err = config.MergeRuleFiles(baseRules, appRules)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error merging rules: %v\n", err)
			os.Exit(1)
		}
	} else {
		merged = baseRules
	}

	if err := config.ValidateRules(merged.Rules); err != nil {
		fmt.Fprintf(os.Stderr, "Error validating rules: %v\n", err)
		os.Exit(1)
	}

	rsv := resolver.NewDirResolver(cfg.BasePath)

	var result interface{}
	switch mode {
	case config.ModeCorrectness:
		result, err = scanner.RunCorrectness(rsv, appName, instances, merged)
	case config.ModeDiff:
		result, err = scanner.RunDiff(rsv, appName, instances[0], instances[1], merged)
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown mode %q (use correctness or diff)\n", mode)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning: %v\n", err)
		os.Exit(1)
	}

	reporter, err := report.NewJSONReporter(outputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating reporter: %v\n", err)
		os.Exit(1)
	}
	if err := reporter.Report(result); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing report: %v\n", err)
		os.Exit(1)
	}

	hasError := false
	switch r := result.(type) {
	case *scanner.CorrectnessResult:
		hasError = r.Summary.Total > 0
	case *scanner.DiffResult:
		hasError = r.Summary.Total > 0
	}
	if hasError {
		os.Exit(1)
	}
}

func parseArgs(args []string) map[string]string {
	result := make(map[string]string)
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--") && i+1 < len(args) {
			key := strings.TrimPrefix(args[i], "--")
			result[key] = args[i+1]
			i++
		}
	}
	return result
}
