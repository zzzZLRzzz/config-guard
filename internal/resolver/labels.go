package resolver

import (
	"fmt"
	"strings"
)

func ParseLabels(instanceName string) (map[string]string, error) {
	parts := strings.SplitN(instanceName, "-", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("cannot parse instance name %s: expected format {env}-{region}", instanceName)
	}
	return map[string]string{
		"env":    parts[0],
		"region": parts[1],
	}, nil
}

func ResolveFrom(from string, labels map[string]string) (interface{}, error) {
	if !strings.HasPrefix(from, "instance.labels.") {
		return nil, fmt.Errorf("unknown from source %s", from)
	}
	labelName := strings.TrimPrefix(from, "instance.labels.")
	v, ok := labels[labelName]
	if !ok {
		return nil, fmt.Errorf("label %s not found", labelName)
	}
	return v, nil
}
