package evaluator

import (
	"encoding/json"
	"fmt"
	"os"
)

func LoadSuiteFromJSON(path string) (EvalSuite, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return EvalSuite{}, fmt.Errorf("read suite failed: %w", err)
	}

	var suite EvalSuite
	if err := json.Unmarshal(data, &suite); err != nil {
		return EvalSuite{}, fmt.Errorf("unmarshal suite failed: %w", err)
	}
	if suite.SuiteID == "" {
		return EvalSuite{}, fmt.Errorf("suite_id is required")
	}
	if len(suite.Groups) == 0 {
		return EvalSuite{}, fmt.Errorf("groups is required")
	}
	return suite, nil
}
