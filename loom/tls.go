package loom

import (
	"crypto/tls"
	"crypto/x509"
	"os"
)

func tlsConfig(c TLSConfig, nextProtos []string) (*tls.Config, error) {
	cfg := &tls.Config{
		NextProtos:         nextProtos,
		InsecureSkipVerify: c.InsecureSkipVerify,
		ServerName:         c.ServerName,
	}
	if c.CAFile != "" {
		b, err := os.ReadFile(c.CAFile)
		if err != nil {
			return nil, err
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(b)
		cfg.RootCAs = pool
	}
	if c.CertFile != "" || c.KeyFile != "" {
		crt, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
		if err != nil {
			return nil, err
		}
		cfg.Certificates = []tls.Certificate{crt}
	}
	return cfg, nil
}
