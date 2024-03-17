package relay

import (
	"github.com/gorilla/websocket"
	"hdnprxy/util"
	"log"
	"time"
)

// /Obey the client interface (in schema) but to the north have a web socket and to the south have a tcp connection
type WebSockRelay struct {
	url     string
	timeout time.Duration
	conn    *websocket.Conn
}

// // Use this to create a new north bound relay, which can then be connected
func NewWebSockRelay(url string, timeout time.Duration) *WebSockRelay {
	return &WebSockRelay{
		url:     url,
		timeout: timeout,
	}
}

// / Use this for an incoming south side connection that's already been accepted
func NewWebSockRelayFromConn(conn *websocket.Conn, timeout time.Duration) *WebSockRelay {
	return &WebSockRelay{
		conn:    conn,
		timeout: timeout,
	}
}

func (p *WebSockRelay) Connect() error {
	if p.conn != nil {
		log.Panicln("WebSockRelay: Connect: Already connected") /// somethings wrong - has the web socket been passed into the constructor
	}
	/// Connect to the web socket
	c, _, err := websocket.DefaultDialer.Dial(p.url, nil)
	util.CheckError(err)
	p.conn = c
	return nil
}

func (p *WebSockRelay) Close() {
	p.conn.Close()
}

func (p *WebSockRelay) SendMsg(data []byte) error {
	/// Send data to the web socket
	return p.conn.WriteMessage(websocket.BinaryMessage, data)
}

func (p *WebSockRelay) RecvMsg() (data []byte, err error) {
	/// Receive data from the web socket
	_, data, err = p.conn.ReadMessage()
	return data, err
}
