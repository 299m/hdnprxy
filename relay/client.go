package relay

import (
	"hdnprxy/util"
	"net"
	"time"
)

type Client struct {
	url     string
	timeout time.Duration
	conn    *net.Conn

	southbuffer []byte
}

func NewClient(url string, timeout time.Duration) *Client {
	return &Client{
		url:         url,
		timeout:     timeout,
		southbuffer: make([]byte, 1024),
	}
}

// / Create a new client from an existing connection
func NewClientFromConn(conn *net.Conn, timeout time.Duration) *Client {
	return &Client{
		conn:        conn,
		timeout:     timeout,
		southbuffer: make([]byte, 1024),
	}
}

func (p *Client) Connect() error {
	conn, err := net.Dial("tcp", p.url)
	util.CheckError(err)
	p.conn = &conn
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
