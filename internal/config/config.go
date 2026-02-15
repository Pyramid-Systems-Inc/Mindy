package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	WatchPaths []string `yaml:"watch_paths"`
	HttpPort   int      `yaml:"http_port"`
	DataDir    string   `yaml:"data_dir"`
}

func Default() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		WatchPaths: []string{},
		HttpPort:   9090,
		DataDir:    filepath.Join(home, ".mindy", "data"),
	}
}

func Load(path string) (*Config, error) {
	cfg := Default()

	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) EnsureDataDir() error {
	return os.MkdirAll(c.DataDir, 0755)
}
