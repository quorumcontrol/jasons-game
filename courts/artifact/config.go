package artifact

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/courts/config"
)

type Artifact struct {
	OriginAuth   string `yaml:"origin_auth"`
	Inscriptions struct {
		Type     string `yaml:"type"`
		Material string `yaml:"material"`
		Age      string `yaml:"age"`
		Weight   string `yaml:"weight"`
		ForgedBy string `yaml:"forged by"`
	}
}

type ArtifactsConfig struct {
	Artifacts      []*Artifact
	NamesPool      []string
	ObjectTemplate []byte
}

var inscribeableKeys = []string{"Type", "Material", "Age", "Weight"}

func NewArtifactsConfig(path string) (*ArtifactsConfig, error) {
	cfg := &ArtifactsConfig{}
	err := config.ReadYaml(filepath.Join(path, "artifacts.yml"), cfg)
	if err != nil {
		return nil, errors.Wrap(err, "error processing artifacts.yml")
	}

	cfg.ObjectTemplate, err = ioutil.ReadFile(filepath.Join(path, "template.yml"))
	if err != nil {
		return nil, errors.Wrap(err, "error fetching template")
	}

	names := make(map[string][]string)
	err = config.ReadYaml(filepath.Join(path, "names.yml"), names)
	if err != nil {
		return nil, errors.Wrap(err, "error processing names.yml")
	}
	cfg.NamesPool = names["names"]

	err = cfg.validate()
	if err != nil {
		return nil, errors.Wrap(err, "invalid config")
	}

	return cfg, nil
}

func (c *ArtifactsConfig) validate() error {
	var err error

	for i, artifact := range c.Artifacts {
		if len(artifact.Inscriptions.Type) == 0 {
			return fmt.Errorf("must set Type on artifact #%d", i)
		}
		if len(artifact.Inscriptions.Material) == 0 {
			return fmt.Errorf("must set Material on artifact #%d", i)
		}
		if len(artifact.Inscriptions.Age) == 0 {
			return fmt.Errorf("must set Age on artifact #%d", i)
		}
		if len(artifact.Inscriptions.Weight) == 0 {
			return fmt.Errorf("must set Weight on artifact #%d", i)
		}
		if len(artifact.Inscriptions.ForgedBy) == 0 {
			return fmt.Errorf("must set ForgedBy on artifact #%d", i)
		}
	}

	// just check to make sure yaml parses
	err = config.ParseYaml(c.ObjectTemplate, new(interface{}))
	if err != nil {
		return errors.Wrap(err, "error parsing yaml on artifact ObjectTemplate")
	}

	if len(c.NamesPool) == 0 {
		return fmt.Errorf("must set names in names.yml")
	}

	return nil
}

func (c *ArtifactsConfig) inscribeableKeys() []string {
	return inscribeableKeys
}

func (c *ArtifactsConfig) inscribeableValuesFor(key string) []string {
	values := make([]string, len(c.Artifacts))
	for i, artifact := range c.Artifacts {
		switch key {
		case "Type":
			values[i] = artifact.Inscriptions.Type
		case "Material":
			values[i] = artifact.Inscriptions.Material
		case "Age":
			values[i] = artifact.Inscriptions.Age
		case "Weight":
			values[i] = artifact.Inscriptions.Weight
		case "ForgedBy":
			values[i] = artifact.Inscriptions.ForgedBy
		default:
			return []string{}
		}
	}
	return values
}
