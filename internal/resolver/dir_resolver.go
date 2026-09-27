package resolver

import (
	"fmt"
	"path/filepath"
)

type DirResolver struct {
	BasePath string
}

func NewDirResolver(basePath string) *DirResolver {
	return &DirResolver{BasePath: basePath}
}

func (d *DirResolver) Resolve(instance string, app string) (*ResolvedConfig, error) {
	dir := filepath.Join(d.BasePath, app, instance)
	flat, err := FlattenDir(dir)
	if err != nil {
		return nil, fmt.Errorf("flatten %s/%s: %w", app, instance, err)
	}

	labels, err := ParseLabels(instance)
	if err != nil {
		return nil, fmt.Errorf("parse labels for %s: %w", instance, err)
	}
	labels["app"] = app

	return &ResolvedConfig{
		Labels:    labels,
		Values:    flat.Values,
		KeySource: flat.KeySource,
	}, nil
}
