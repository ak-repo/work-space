package store

import (
	"encoding/json"
	"os"

	"github.com/ak-repo/discount-engine/internal/engine"
)

func LoadRules(path string) ([]engine.Rule, error) {

	data, err := os.ReadFile(path)

	if err != nil {
		return nil, err
	}

	var rules []engine.Rule
	err = json.Unmarshal(data, &rules)
	return rules, err

}
