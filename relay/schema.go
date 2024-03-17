package relay

type Relay interface {
	Connect() error
	Close()
	SendMsg(data []byte) error
	RecvMsg() (data []byte, err error)
}
