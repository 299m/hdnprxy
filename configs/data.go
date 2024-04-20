package configs

import "os"

type TlsConfig struct {
	Cert string
	Key  string
	Port string

	IsHttps    bool /// one of these must be set
	IsProxy    bool
	IsTlsProxy bool /// it seems we're getting data prior to the tls handshake
	IsTcpProxy bool
	IsUdpProxy bool
}

func (t *TlsConfig) Expand() {
	t.Cert = os.ExpandEnv(t.Cert)
	t.Key = os.ExpandEnv(t.Key)
}
