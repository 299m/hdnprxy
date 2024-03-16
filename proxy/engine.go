package proxy

import (
	"hdnprxy/relay"
	"hdnprxy/util"
	"log"
	"net"
)

type Config struct {
	Buffersize int
}

type Engine struct {
	relayer *relay.Client
	conn    net.Conn

	cfg *Config
}

func NewEngine(relayer *relay.Client, conn net.Conn, cfg *Config) *Engine {
	return &Engine{
		relayer: relayer,
		conn:    conn,
		cfg:     cfg,
	}
}

func (p *Engine) ProcessNorthbound() {
	defer p.relayer.Close()
	defer p.conn.Close()

	for {
		message, err := p.relayer.RecvMsg()
		if err != nil {
			log.Println(err)
			return
		}
		p.conn.Write(message)
	}
}

func (p *Engine) ProcessSouthbound() {
	defer p.relayer.Close()
	defer p.conn.Close()
	buffer := make([]byte, p.cfg.Buffersize)

	for {
		n, err := p.conn.Read(buffer)
		util.CheckError(err)
		p.relayer.SendMsg(buffer[:n])
	}
}
