package report

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Reporter interface {
	Report(result interface{}) error
}

type JSONReporter struct {
	Writer io.Writer
}

func NewJSONReporter(path string) (*JSONReporter, error) {
	var w io.Writer
	if path == "" || path == "-" {
		w = os.Stdout
	} else {
		f, err := os.Create(path)
		if err != nil {
			return nil, fmt.Errorf("create report file %s: %w", path, err)
		}
		w = f
	}
	return &JSONReporter{Writer: w}, nil
}

func (r *JSONReporter) Report(result interface{}) error {
	enc := json.NewEncoder(r.Writer)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
