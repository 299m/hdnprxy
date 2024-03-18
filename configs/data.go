package configs

import "os"

type TlsConfig struct {
	Cert string
	Key  string
	Port string

	IsHttps bool /// one of these must be set
	IsProxy bool
}

func (t *TlsConfig) Expand() {
	t.Cert = os.ExpandEnv(t.Cert)
	t.Key = os.ExpandEnv(t.Key)
}
