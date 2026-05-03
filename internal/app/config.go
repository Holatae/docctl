package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// Config ============================================
// DATAMODELLER FÖR CONFIG
// ============================================
var configMutex sync.Mutex

type Config struct {
	Foreningar map[string]Association `yaml:"foreningar"`
	Settings   Settings               `yaml:"settings"`
}

type Association struct {
	Id        string   `yaml:"-"`
	Name      string   `yaml:"namn"`
	OrgNummer string   `yaml:"org_nummer"`
	Body      []string `yaml:"organ"`
}

type Settings struct {
	CreateZIP         bool `yaml:"create_zip"`
	UseOpenTimeStamps bool `yaml:"use_open_time_stamps"`
}

func LoadConfig(projRoot string) (Config, error) {
	configMutex.Lock()
	defer configMutex.Unlock()

	configPath := filepath.Join(projRoot, ".tooling", "config.yaml")
	var cfg Config

	data, err := os.ReadFile(configPath)
	if err != nil {
		cfg = Config{
			Foreningar: map[string]Association{},
			Settings:   Settings{CreateZIP: true, UseOpenTimeStamps: false},
		}
		if err := saveConfigLocked(projRoot, cfg); err != nil {
			return cfg, err
		}
		return cfg, nil
	}
	_ = yaml.Unmarshal(data, &cfg)

	for key, org := range cfg.Foreningar {
		org.Id = key

		cfg.Foreningar[key] = org
	}
	return cfg, nil
}

func SaveConfig(projRoot string, cfg Config) error {
	configMutex.Lock()
	defer configMutex.Unlock()
	return saveConfigLocked(projRoot, cfg)
}

func saveConfigLocked(projRoot string, cfg Config) error {
	configPath := filepath.Join(projRoot, ".tooling", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fmt.Errorf("could not make configpath: %w", err)
	}
	data, _ := yaml.Marshal(&cfg)
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return fmt.Errorf("could not write config: %w", err)
	}
	return nil
}
