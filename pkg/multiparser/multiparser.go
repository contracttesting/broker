package multiparser

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
)

var ErrUnsupportedFileExtension = errors.New("unsupported contract file extension")

func AnyToJSON(path string, raw []byte) ([]byte, error) {
	extension := strings.ToLower(filepath.Ext(path))

	switch extension {
	case ".json":
		return raw, nil
	case ".yaml", ".yml":
		return yaml.YAMLToJSON(raw)
	default:
		return nil, ErrUnsupportedFileExtension
	}
}
