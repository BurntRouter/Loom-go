package loom

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

type ClientOptions struct {
	Addr      string // host:port
	Transport Transport
	TLS       TLSConfig

	Name  string
	Room  string
	Token string
}
