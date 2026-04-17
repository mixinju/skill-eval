package eval

import (
	"encoding/json"
	"fmt"
	"os"
)

type Case struct {
	ID       string `json:"id" yaml:"id"`
	Name     string `json:"name" yaml:"name"`
	Input    string `json:"input" yaml:"input"`
	Expected string `json:"expected,omitempty" yaml:"expected,omitempty"`
}

func LoadCases(path string) ([]Case, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read cases file %s: %w", path, err)
	}

	var cases []Case
	if err := json.Unmarshal(data, &cases); err != nil {
		return nil, fmt.Errorf("parse cases: %w", err)
	}

	for i, c := range cases {
		if c.ID == "" {
			cases[i].ID = fmt.Sprintf("case_%d", i+1)
		}
	}

	return cases, nil
}
