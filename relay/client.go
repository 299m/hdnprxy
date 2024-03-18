package relay

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"hdnprxy/util"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Client struct {
	url     string
	timeout time.Duration
	conn    *tls.Conn

	southbuffer   []byte
	trustedcacert string

	paramname  string
	paramvalue string
}

func NewClient(url string, timeout time.Duration) *Client {
	return &Client{
		url:         url,
		timeout:     timeout,
		southbuffer: make([]byte, 1024),
	}
}

// / if we are setting up a tunnel from a local proxy server ot a remote proxy, use this
func NewTunnelClient(url string, timeout time.Duration, paramname string, paramvalue string) *Client {
	return &Client{
		url:         url,
		timeout:     timeout,
		southbuffer: make([]byte, 1024),
		paramname:   paramname,
		paramvalue:  paramvalue,
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
	fullurl, err := url.Parse(p.url)
	util.CheckError(err)
	conn, err := tls.Dial("tcp", fullurl.Host, config)
	util.CheckError(err)
	p.conn = conn
	if p.paramname != "" {
		data, err := json.Marshal(map[string]string{p.paramname: p.paramvalue})
		util.CheckError(err)
		buf := &bytes.Buffer{}
		buf.Write(data)
		req, err := http.NewRequest(http.MethodPost, p.url, buf)
		/// This should trigger the tunnel setup - after that we should be on a tls/tcp protocol
		err = req.Write(p.conn)
		util.CheckError(err)
	}
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
