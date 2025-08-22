package resourcestate

import (
	"os"

	"gopkg.in/yaml.v3"

	crs "k8s.io/kube-state-metrics/v2/pkg/customresourcestate"
)

// Config wraps a list of customresourcestate.Resource entries for core resources.
type Config struct {
	Kind string `yaml:"kind"` // expect "ResourceMetricsConfig"
	Spec struct {
		Resources []crs.Resource `yaml:"resources"`
	} `yaml:"spec"`
}

func LoadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	if c.Kind == "" {
		c.Kind = "ResourceMetricsConfig"
	}
	return &c, nil
}
