package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	TDLib          TDLibConfig    `yaml:"tdlib"`
	Logger         LoggerConfig   `yaml:"logger"`
	DatabaseConfig DatabaseConfig `yaml:"database"`
}

func LoadConfig(path string) (Config, error) {
	cfg := Config{}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
