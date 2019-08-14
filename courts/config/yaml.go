package config

import (
	"bytes"
	"io/ioutil"
	"text/template"

	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

func ReadYaml(filePath string, dst interface{}, vars ...map[string]interface{}) error {
	yamlBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return errors.Wrap(err, "error reading file "+filePath)
	}

	return ParseYaml(yamlBytes, dst, vars...)
}

func ParseYaml(yamlBytes []byte, dst interface{}, vars ...map[string]interface{}) error {
	if len(vars) > 0 {
		yamlString, err := ReplaceVariables(string(yamlBytes), vars...)
		if err != nil {
			return errors.Wrap(err, "error replacing vars")
		}
		yamlBytes = []byte(yamlString)
	}

	err := yaml.Unmarshal(yamlBytes, dst)
	if err != nil {
		return errors.Wrap(err, "error unmarshalling yaml")
	}

	return nil
}

func ReplaceVariables(templateStr string, vars ...map[string]interface{}) (string, error) {
	if len(vars) == 0 {
		return templateStr, nil
	}

	mergedVars := make(map[string]interface{})

	for _, varMap := range vars {
		if err := mergo.Merge(&mergedVars, varMap); err != nil {
			return templateStr, errors.Wrap(err, "error merging vars")
		}
	}

	tmpl, err := template.New("config").Parse(templateStr)
	if err != nil {
		return templateStr, errors.Wrap(err, "error parsing template")
	}

	var outBuff bytes.Buffer
	err = tmpl.Execute(&outBuff, mergedVars)
	if err != nil {
		return templateStr, errors.Wrap(err, "error processing template")
	}

	return outBuff.String(), nil
}
