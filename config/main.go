package config

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/mr55p-dev/gonk"
	"gopkg.in/yaml.v3"
)

type NginxBlock struct {
	ExternalHost string
	Protocol     string `config:"protocol,optional"`
	IPv4         string `config:"ipv4"`
	Port         int
	Protected    bool `config:"protected,optional"`
}

type AppConfig struct {
	App     string
	Nginx   []NginxBlock `config:"nginx,optional"`
	Runtime struct {
		EnvExtensions []string       `config:"env-extensions,optional"`
		Env           map[string]any `config:"env,optional"`
	} `config:"runtime,optional"`
}

type Extensions map[string]map[string]string

func NewExtensions(src io.Reader) (Extensions, error) {
	data, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}
	ext := make(map[string]map[string]string)
	err = yaml.Unmarshal(data, ext)
	if err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}
	return ext, nil
}

func NewAppConfig(dir string) (*AppConfig, error) {
	// Load the config object
	cfg := new(AppConfig)
	loader, err := gonk.NewYamlLoader(filepath.Join(dir, "app.yml"))
	if err != nil {
		return nil, fmt.Errorf("Error creating loader: %w", err)
	}
	if err := gonk.LoadConfig(&cfg, loader); err != nil {
		return nil, fmt.Errorf("Error loading config: %w", err)
	}
	for i := 0; i < len(cfg.Nginx); i++ {
		if cfg.Nginx[i].Protocol == "" {
			cfg.Nginx[i].Protocol = "http"
		}
	}

	return cfg, nil
}
