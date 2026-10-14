package crud

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	if len(cfg.Resources) == 0 {
		return nil, fmt.Errorf("%s: no resources defined", path)
	}

	seen := make(map[string]struct{}, len(cfg.Resources))
	for _, r := range cfg.Resources {
		if err := r.Validate(); err != nil {
			return nil, err
		}
		if _, dup := seen[r.Name]; dup {
			return nil, fmt.Errorf("resource %s: duplicate name", r.Name)
		}
		seen[r.Name] = struct{}{}
	}

	return &cfg, nil
}
