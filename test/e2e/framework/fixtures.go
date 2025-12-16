package framework

import (
	"os"

	"gopkg.in/yaml.v3"
	"github.com/kubestack-ai/kubestack-ai/internal/storage/graph"
)

// Fixture defines test data structure.
type Fixture struct {
	Nodes []graph.Node `yaml:"nodes"`
	Edges []graph.Edge `yaml:"edges"`
}

// LoadFixtureFromFile loads a fixture from a YAML file.
func LoadFixtureFromFile(path string) (*Fixture, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f Fixture
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return &f, nil
}
