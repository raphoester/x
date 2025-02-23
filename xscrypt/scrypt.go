package xscrypt

import (
	"fmt"

	"github.com/Aoang/firebase-scrypt"
)

type Config struct {
	SignerKeyB64     string `yaml:"signer_key_b64"`
	SaltSeparatorB64 string `yaml:"salt_separator_b64"`
	Rounds           int    `yaml:"rounds"`
	MemoryCost       int    `yaml:"memory_cost"`
}

func (c *Config) ResetToDefault() {
	c.SignerKeyB64 = "aGVsbG8gd29ybGQK"
	c.SaltSeparatorB64 = "aGVsbG8gd29ybGQK"
	c.Rounds = 8
	c.MemoryCost = 14
}

func New(config Config) (*scrypt.App, error) {
	app, err := scrypt.New(config.SignerKeyB64, config.SaltSeparatorB64, config.Rounds, config.MemoryCost)
	if err != nil {
		return nil, fmt.Errorf("could not create scrypt app: %w", err)
	}

	return app, nil
}
