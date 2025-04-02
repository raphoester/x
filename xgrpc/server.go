package xgrpc

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type ServerConfig struct {
	BindAddress      string `yaml:"bind_address"`
	EnableReflection bool   `yaml:"enable_reflection"`
}

func (c *ServerConfig) ResetToDefault() {
	c.BindAddress = "localhost:8080"
	c.EnableReflection = true
}

func RegisterReflectionIfNeeded(config ServerConfig, server *grpc.Server) {
	if config.EnableReflection {
		reflection.Register(server)
	}
}

type Registrar func(srv *grpc.Server)

type Registrable interface {
	Registrar() Registrar
}
