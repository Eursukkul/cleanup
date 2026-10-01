package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

type Settings struct {
	Directories     []string `yaml:"directories"`
	RetentionMonths int      `yaml:"retention_months"`
	DryRun          bool     `yaml:"dry_run"`
}

// Load reads a fresh configuration for each batch invocation.
func Load(filename string) (Settings, error) {
	settings := Settings{RetentionMonths: 3, DryRun: true}
	file, err := os.Open(filename)
	if err != nil {
		return Settings{}, fmt.Errorf("open config: %w", err)
	}
	defer file.Close()
	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)
	if err := decoder.Decode(&settings); err != nil {
		return Settings{}, fmt.Errorf("decode config: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err != nil {
			return Settings{}, fmt.Errorf("decode config: %w", err)
		}
		return Settings{}, errors.New("config must contain exactly one YAML document")
	}
	if len(settings.Directories) == 0 {
		return Settings{}, errors.New("config directories must contain at least one folder")
	}
	if settings.RetentionMonths < 1 || settings.RetentionMonths > 1200 {
		return Settings{}, errors.New("retention_months must be between 1 and 1200")
	}
	for i, directory := range settings.Directories {
		if strings.TrimSpace(directory) == "" {
			return Settings{}, fmt.Errorf("directories[%d] must not be blank", i)
		}
		if !filepath.IsAbs(directory) {
			settings.Directories[i] = filepath.Join(filepath.Dir(filename), directory)
		}
	}
	return settings, nil
}
