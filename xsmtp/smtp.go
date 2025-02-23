package xsmtp

import (
	"bytes"
	"fmt"
	"net"
	"net/smtp"

	"github.com/raphoester/x/xid"
)

type ScalewayConfig struct {
	Region        string `yaml:"region"`
	Secret        string `yaml:"secret"`
	Username      string `yaml:"username"`
	RemoteAddress string `yaml:"remote_address"`
	SendFrom      string `yaml:"send_from"`
	host          string `yaml:"-"`
}

func (s *ScalewayConfig) ResetToDefault() {
	s.Region = "fr-par"
	s.Secret = xid.NewDefaultFixedGenerator().Generate()
	s.Username = xid.NewDefaultFixedGenerator().Generate()
	s.RemoteAddress = "smtp.tem.scw.cloud:2587"
	s.SendFrom = "noreply@example.test"

}

type ScalewaySender struct {
	config ScalewayConfig
}

func NewScalewaySender(config ScalewayConfig) (*ScalewaySender, error) {
	host, port, err := net.SplitHostPort(config.RemoteAddress)
	if err != nil {
		return nil, fmt.Errorf("unable to parse remote address: %w", err)
	}

	config.RemoteAddress = net.JoinHostPort(host, port)
	config.host = host

	return &ScalewaySender{config: config}, nil
}

func (s *ScalewaySender) Send(to []string, content bytes.Buffer) error {
	auth := smtp.PlainAuth("", s.config.Username, s.config.Secret, s.config.host)
	err := smtp.SendMail(s.config.RemoteAddress, auth, s.config.SendFrom, to, content.Bytes())
	if err != nil {
		return fmt.Errorf("could not send email: %w", err)
	}
	return nil
}

func (s *ScalewaySender) From() string {
	return s.config.SendFrom
}
