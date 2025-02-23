package xconfig

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/raphoester/x/basicutil"
)

type stringArray []string

func (s *stringArray) String() string {
	return strings.Join(*s, ", ")
}
func (s *stringArray) Set(value string) error {
	*s = append(*s, value)
	return nil
}

type ConfigLoader struct {
	sa stringArray
}

func NewFromDefaultFiles() *ConfigLoader {
	loader := ConfigLoader{}
	loader.sa = []string{}
	inParent, err := basicutil.FindInParentDirectories("config.yaml", "go.mod")
	if err == nil {
		loader.sa = append(loader.sa, inParent...)
	}

	return &loader
}

// New creates a config loader with ability to give multiple config files
func New() *ConfigLoader {
	LoadDefaultEnvFile()
	loader := ConfigLoader{}
	flag.Var(&loader.sa, "config", "config files to be loaded (repeatable) (ordered: last overrides)")
	flag.Parse()
	return &loader
}

type Config interface {
	ResetToDefault()
}

func (l *ConfigLoader) ApplyConfig(rcv Config) error {
	rcv.ResetToDefault()

	var retErr error

	for _, s := range l.sa {
		if err := ApplyYamlFile(rcv, s); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("failed to override config with yaml: %w", err))
		}
	}

	if err := ApplyEnv(rcv); err != nil {
		retErr = errors.Join(retErr, fmt.Errorf("failed to load env: %w", err))
	}

	return retErr
}
