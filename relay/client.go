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
}

func NewClient(url string, timeout time.Duration) *Client {
	return &Client{
		url:     url,
		timeout: timeout,
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
	_, err = (*p.conn).Read(data)
	return data, err
}
