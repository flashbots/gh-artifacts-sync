package config

import (
	"errors"
	"fmt"
	"net"
)

type Server struct {
	ListenAddress string `yaml:"listen_address" json:"listen_address"`
}

var (
	errServerInvalidListenAddress = errors.New("invalid server listen address")
)

func (cfg *Server) Validate() error {
	if _, err := net.ResolveTCPAddr("tcp", cfg.ListenAddress); err != nil {
		return fmt.Errorf("%w: %w",
			errServerInvalidListenAddress, err,
		)
	}

	return nil
}
