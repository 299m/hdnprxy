package relay

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/299m/util/util"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Client struct {
	url     string
	timeout time.Duration
	conn    net.Conn

	southbuffer   []byte
	trustedcacert []string

	paramname  string
	paramvalue string

	debuglogs DebugLog
	connid    string

	usetls bool
}

func newClient(url string, timeout time.Duration, usetls bool) *Client {
	return &Client{
		url:         url,
		timeout:     timeout,
		southbuffer: make([]byte, 1024),
		usetls:      usetls,
	}
}

// // Old. Best to use the v2 version
func NewClient(url string, timeout time.Duration) *Client {
	return newClient(url, timeout, true)
}
func NewClientv2(url string, timeout time.Duration, usetls bool) *Client {
	return newClient(url, timeout, usetls)
}

// / if we are setting up a tunnel from a local proxy server ot a config proxy, use this
func NewTunnelClient(url string, timeout time.Duration, paramname string, paramvalue string) *Client {

	fmt.Println("Creating tunnel client with url ", url, " and ", paramname, " ", paramvalue[:4], "******")
	return &Client{
		url:         url,
		timeout:     timeout,
		southbuffer: make([]byte, 1024),
		paramname:   paramname,
		paramvalue:  paramvalue,
	}
}

// / Create a new client from an existing connection
func NewClientFromConn(conn net.Conn, timeout time.Duration) *Client {
	return &Client{
		conn:        conn,
		timeout:     timeout,
		southbuffer: make([]byte, 1024),
	}
}

func (p *Client) AllowCert(cert []string) {
	p.trustedcacert = cert
}

func (p *Client) EnableDebugLogs(on bool, connid string) {
	p.debuglogs.EnableDebugLogs(on, connid)
	p.connid = connid
}

func (p *Client) checkFirstResp(conn net.Conn) (valid bool, status string) {
	/// Read the 1st response from the north - then, if it's a http 200, we can start the tunnel
	///DEBUG - read the raw data - see what we get
	//buf := make([]byte, 1024)
	//n, err := conn.Read(buf)
	//fmt.Println("Raw HTTP response ", string(buf[:n]))
	//util.CheckError(err)
	firstresp, err := http.ReadResponse(bufio.NewReader(conn), nil) /// this is a
	util.CheckError(err)
	if firstresp.StatusCode != 200 {
		fmt.Println("Error response from the north", firstresp.Status)
		return false, firstresp.Status
	}
	return true, ""
}

/*
*
After this, the connection belongs to the caller of this function - they must clean it up
*/
func (p *Client) Hijack() net.Conn {
	return p.conn
}

func (p *Client) Connect() error {
	config := &tls.Config{}
	// Get the SystemCertPool, continue with an empty pool on error
	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}
	for _, certfile := range p.trustedcacert {
		p.debuglogs.LogDebug("Adding trusted certfile ", certfile)

		// Read in the certfile file
		certs, err := os.ReadFile(certfile)
		util.CheckError(err)
		// Append our certfile to the system pool
		if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
			log.Println("No certs appended, using system certs only")
		}
	}
	config.RootCAs = rootCAs

	fullurl, err := url.Parse(p.url)
	util.CheckError(err)
	p.debuglogs.LogDebug("Fullurl ", fullurl.Hostname())
	conn, err := tls.Dial("tcp", fullurl.Hostname()+":"+fullurl.Port(), config)
	if err != nil {
		return err
	}

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
		if valid, status := p.checkFirstResp(p.conn); !valid {

			return fmt.Errorf(status)
		}
	}
	return nil
}

func (p *Client) Close() {
	p.conn.Close()
}

func (p *Client) SendMsg(data []byte) error {
	p.debuglogs.LogData(string(data), "send: ")
	p.conn.SetWriteDeadline(time.Now().Add(p.timeout))
	_, err := p.conn.Write(data)
	//// FOR DEBUGGING TLS ISSUE
	util.CheckError(err)
	return err
}

func (p *Client) RecvMsg() (data []byte, err error) {
	p.conn.SetReadDeadline(time.Now().Add(p.timeout))
	data = p.southbuffer
	n, err := p.conn.Read(data)
	//// FOR DEBUGGING TLS ISSUE
	util.CheckError(err)
	p.debuglogs.LogData(string(data[:n]), "recv: ")
	return data[:n], err
}
