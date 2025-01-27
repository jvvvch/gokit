package parseconf

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/kelseyhightower/envconfig"
)

func FromJSON[T any](config *T, filePath string) error {
	filePath = filepath.Clean(filePath)
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("parseconf: failed to open file %v: %w", filePath, err)
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("parseconf: failed to read file %v: %w", filePath, err)
	}

	if err := json.Unmarshal(bytes, config); err != nil {
		return fmt.Errorf("parseconf: failed to unmarshal json from file %v: %w", filePath, err)
	}
	return nil
}

func FromEnv[T any](config *T) error {
	err := envconfig.Process("", config)
	if err != nil {
		return fmt.Errorf("parseconf: failed to parse config from env: %w", err)
	}
	return nil
}
