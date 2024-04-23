package relay

import "net"

type Relay interface {
	Connect() error
	Close()
	SendMsg(data []byte) error
	RecvMsg() (data []byte, from net.Addr, err error)
	EnableDebugLogs(bool, string)
}
