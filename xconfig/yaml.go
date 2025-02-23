package xconfig

import (
	"encoding/base64"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type ByteArray []byte

func (b *ByteArray) UnmarshalYAML(value *yaml.Node) error {
	var strVal string
	if err := value.Decode(&strVal); err != nil {
		return err
	}

	decoded, err := base64.StdEncoding.DecodeString(strVal)
	if err != nil {
		return fmt.Errorf("failed to decode base64 string: %v", err)
	}

	*b = decoded
	return nil
}

func ApplyYamlFile(rcv any, absoluteFilePath string) error {
	fileContent, err := os.ReadFile(absoluteFilePath)
	if err != nil {
		return fmt.Errorf("unable to read file %q: %w", absoluteFilePath, err)
	}
	if err := loadRawYamlContents(rcv, fileContent); err != nil {
		return fmt.Errorf("unable to load yaml files: %w", err)
	}

	return nil
}

func loadRawYamlContents(rcv any, content []byte) error {
	var yamlNode yaml.Node
	if err := yaml.Unmarshal(content, &yamlNode); err != nil {
		return fmt.Errorf("unable to read yaml config file: %w", err)
	}

	if err := yamlNode.Decode(rcv); err != nil {
		return fmt.Errorf("unable to decode yaml config file %w", err)
	}

	return nil

}
