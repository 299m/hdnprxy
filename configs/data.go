package configs

import "os"

type TlsConfig struct {
	Cert string
	Key  string
	Port string
}

func (t *TlsConfig) Expand() {
	t.Cert = os.ExpandEnv(t.Cert)
	t.Key = os.ExpandEnv(t.Key)
}
