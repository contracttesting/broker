package publish_contract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/contracttesting/broker/internal/dsl"
	"github.com/goccy/go-yaml"
)

// parseFragment picks the parser by the extension of the source. YAML goes through
// YAMLToJSON so both formats land on the same JSON unmarshal into dsl.Contract.
func parseFragment(fragment ContractFragment) (*dsl.Contract, error) {
	extension := strings.ToLower(filepath.Ext(fragment.Source))
	if extension != ".yaml" && extension != ".yml" && extension != ".json" {
		return nil, fmt.Errorf(
			"unsupported contract file: %s (expected .yaml, .yml or .json)",
			fragment.Source,
		)
	}

	contract := &dsl.Contract{}

	content := bytes.TrimSpace([]byte(fragment.Content))
	if len(content) == 0 {
		return contract, nil
	}

	if extension != ".json" {
		converted, err := yaml.YAMLToJSON(content)
		if err != nil {
			return nil, malformedContractFile(fragment.Source, err)
		}

		content = converted
	}

	if err := json.Unmarshal(content, contract); err != nil {
		return nil, malformedContractFile(fragment.Source, err)
	}

	return contract, nil
}

func malformedContractFile(source string, err error) error {
	return fmt.Errorf("malformed contract file: %s: %v", source, err)
}
