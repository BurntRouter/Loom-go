package loom

import "time"

type Transport string

const (
	TransportQUIC Transport = "quic"
	TransportH3   Transport = "h3"
)

type TLSConfig struct {
	ServerName         string
	InsecureSkipVerify bool
	CAFile             string // optional (QUIC + H3)
	CertFile           string // optional mTLS
	KeyFile            string // optional mTLS
}

type ReconnectOptions struct {
	Enabled    bool
	Delay      time.Duration
	MaxRetries int // 0 = forever
}

type ClientOptions struct {
	Addr      string // host:port
	Transport Transport
	TLS       TLSConfig

	Name  string
	Room  string
	Token string

	// Reconnect enables best-effort automatic reconnect on disconnects/timeouts.
	// Nil means enabled with defaults.
	Reconnect *ReconnectOptions
}

func (o ClientOptions) reconnect() (enabled bool, delay time.Duration, maxRetries int) {
	if o.Reconnect == nil {
		return true, time.Second, 0
	}
	if !o.Reconnect.Enabled {
		return false, 0, 0
	}
	delay = o.Reconnect.Delay
	if delay <= 0 {
		delay = time.Second
	}
	return true, delay, o.Reconnect.MaxRetries
}
