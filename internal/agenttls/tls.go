package agenttls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// ListenAndServe starts the agent HTTP or HTTPS server based on TLS env vars.
func ListenAndServe(addr string, handler http.Handler) error {
	certFile := os.Getenv("STRATABENCH_AGENT_TLS_CERT")
	keyFile := os.Getenv("STRATABENCH_AGENT_TLS_KEY")
	if certFile == "" && keyFile == "" {
		return http.ListenAndServe(addr, handler)
	}
	if certFile == "" || keyFile == "" {
		return fmt.Errorf("both STRATABENCH_AGENT_TLS_CERT and STRATABENCH_AGENT_TLS_KEY are required for TLS")
	}
	cfg, err := serverTLSConfig()
	if err != nil {
		return err
	}
	srv := &http.Server{Addr: addr, Handler: handler, TLSConfig: cfg}
	return srv.ListenAndServeTLS(certFile, keyFile)
}

func serverTLSConfig() (*tls.Config, error) {
	cfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if caFile := os.Getenv("STRATABENCH_AGENT_TLS_CA"); caFile != "" {
		caPEM, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("read agent TLS CA: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, fmt.Errorf("invalid STRATABENCH_AGENT_TLS_CA PEM")
		}
		cfg.ClientCAs = pool
		cfg.ClientAuth = tls.RequireAndVerifyClientCert
	}
	return cfg, nil
}

// ConfigureClient applies optional mTLS settings to outbound agent HTTP clients.
func ConfigureClient(baseURL string, client *http.Client) {
	if client == nil {
		return
	}
	if !strings.HasPrefix(strings.ToLower(baseURL), "https://") {
		return
	}
	caFile := os.Getenv("STRATABENCH_AGENT_TLS_CA")
	certFile := os.Getenv("STRATABENCH_AGENT_TLS_CLIENT_CERT")
	keyFile := os.Getenv("STRATABENCH_AGENT_TLS_CLIENT_KEY")
	if caFile == "" && certFile == "" && keyFile == "" {
		return
	}
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if caFile != "" {
		caPEM, err := os.ReadFile(caFile)
		if err == nil {
			pool := x509.NewCertPool()
			if pool.AppendCertsFromPEM(caPEM) {
				tlsCfg.RootCAs = pool
			}
		}
	}
	if certFile != "" && keyFile != "" {
		if cert, err := tls.LoadX509KeyPair(certFile, keyFile); err == nil {
			tlsCfg.Certificates = []tls.Certificate{cert}
		}
	}
	if client.Transport == nil {
		client.Transport = &http.Transport{TLSClientConfig: tlsCfg}
		return
	}
	if tr, ok := client.Transport.(*http.Transport); ok {
		tr.TLSClientConfig = tlsCfg
	}
}
