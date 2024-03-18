package relay

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"hdnprxy/util"
	"log"
	"os"
	"time"
)

type Client struct {
	url     string
	timeout time.Duration
	conn    *tls.Conn

	southbuffer   []byte
	trustedcacert string
}

func NewClient(url string, timeout time.Duration) *Client {
	return &Client{
		url:         url,
		timeout:     timeout,
		southbuffer: make([]byte, 1024),
	}
}

// / Create a new client from an existing connection
func NewClientFromConn(conn *tls.Conn, timeout time.Duration) *Client {
	return &Client{
		conn:        conn,
		timeout:     timeout,
		southbuffer: make([]byte, 1024),
	}
}

func (p *Client) AllowCert(cert string) {
	p.trustedcacert = cert
}

func (p *Client) Connect() error {
	config := &tls.Config{}
	if p.trustedcacert != "" {
		if config.RootCAs == nil {
			config.RootCAs = x509.NewCertPool()
		}
		caCert, err := os.ReadFile(p.trustedcacert)
		util.CheckError(err)
		block, _ := pem.Decode(caCert)
		if block == nil {
			log.Panicln("Failed to decode parent certificate")
		}
		config.RootCAs.AppendCertsFromPEM(block.Bytes)
	}

	conn, err := tls.Dial("tcp", p.url, config)
	util.CheckError(err)
	p.conn = conn
	return nil
}

func (p *Client) Close() {
	(*p.conn).Close()
}

func (p *Client) SendMsg(data []byte) error {
	(*p.conn).SetWriteDeadline(time.Now().Add(p.timeout))
	_, err := (*p.conn).Write(data)
	return err
}

func (p *Client) RecvMsg() (data []byte, err error) {
	(*p.conn).SetReadDeadline(time.Now().Add(p.timeout))
	data = p.southbuffer
	n, err := (*p.conn).Read(data)
	return data[:n], err
}
