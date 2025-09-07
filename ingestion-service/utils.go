package main

import (
	"crypto/tls"
	"fmt"

	"github.com/nats-io/nats.go"
)

func ConnectWithPassword(username, password string) (*nats.Conn, error) {
	// The NATS server must be configured with an authentication provider
	// that validates this user/password pair against your database.
	opts := []nats.Option{
		nats.Name("IoT Ingestion"),
		nats.UserInfo(username, password),
	}
	nc, err := nats.Connect(nats.DefaultURL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	return nc, nil
}

func ConnectWithCert(certFile, keyFile, caFile string) (*nats.Conn, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load key pair: %w", err)
	}

	opts := []nats.Option{
		nats.Name("IoT Ingestion"),
		nats.Secure(&tls.Config{
			Certificates: []tls.Certificate{cert},
		}),
		nats.RootCAs(caFile),
	}
	nc, err := nats.Connect("tls://localhost:4443", opts...) // NATS with TLS
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	return nc, nil
}
