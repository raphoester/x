package xs3

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type Config struct {
	Region   string `yaml:"region"`
	Endpoint string `yaml:"endpoint"`
	ID       string `yaml:"id"`
	Secret   string `yaml:"secret"`
}

func (c *Config) ResetToDefault() {
	c.Region = "fr-par"
	c.Endpoint = "https://s3.fr-par.scw.cloud"
	c.ID = "YOUR_ID"
	c.Secret = "SECRET"
}

func (c *Config) GetCredentials() *credentials.Credentials {
	return credentials.NewStaticCredentials(c.ID, c.Secret, "")
}

func New(c Config) (*s3.S3, error) {
	s, err := c.Session()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return s3.New(s), nil
}

func (c *Config) Session() (*session.Session, error) {
	return session.NewSession(
		&aws.Config{
			Credentials: c.GetCredentials(),
			Endpoint:    &c.Endpoint,
			Region:      &c.Region,
		},
	)
}
