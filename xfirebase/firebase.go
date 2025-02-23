package xfirebase

import (
	"context"
	"encoding/json"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

func NewApp(ctx context.Context, config Config) (*firebase.App, error) {
	jsonCredentials := config.JSONCredentials.AsJSON()

	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsJSON(jsonCredentials))
	if err != nil {
		return nil, fmt.Errorf("failed to create firebase app: %w", err)
	}

	return app, nil
}

type Config struct {
	JSONCredentials JSONCredentials `yaml:"json_credentials"`
}

func (c *Config) ResetToDefault() {
	c.JSONCredentials.ResetToDefault()
}

type JSONCredentials struct {
	Type                    string `json:"type" yaml:"type"`
	ProjectID               string `json:"project_id" yaml:"project_id"`
	PrivateKeyID            string `json:"private_key_id" yaml:"private_key_id"`
	PrivateKey              string `json:"private_key" yaml:"private_key"`
	ClientEmail             string `json:"client_email" yaml:"client_email"`
	ClientID                string `json:"client_id" yaml:"client_id"`
	AuthURI                 string `json:"auth_uri" yaml:"auth_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url" yaml:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url" yaml:"client_x509_cert_url"`
}

func (c *JSONCredentials) ResetToDefault() {
	c.Type = "service_account"
	c.ProjectID = "project-id"
	c.PrivateKeyID = "abcdef-cafe-1234"
	c.PrivateKey = "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDQ8Zz4z8z\n-----END PRIVATE KEY"
	c.ClientEmail = "firebase-adminsdk@default.iam.gservicaccount.com"
	c.ClientID = "1234567890"
	c.AuthURI = "https://accounts.google.com/o/oauth2/auth"
	c.AuthProviderX509CertURL = "https://www.googleapis.com/oauth2/v1/certs"
	c.ClientX509CertURL = "https://www.googleapis.com/robot/v1/metadata/x509/firebase-adminsdk%40default.iam.gserviceaccount.com"
}

func (c *JSONCredentials) AsJSON() []byte {
	serialized, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}

	return serialized
}
