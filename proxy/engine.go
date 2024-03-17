package proxy

import (
	"hdnprxy/relay"
	"hdnprxy/util"
	"log"
)

type Config struct {
	Buffersize int
}

type Engine struct {
	north relay.Relay
	south relay.Relay

	cfg *Config
}

func NewEngine(north relay.Relay, south relay.Relay, cfg *Config) *Engine {
	return &Engine{
		north: north,
		south: south,
		cfg:   cfg,
	}
}

func (p *Engine) ProcessNorthbound() {
	defer p.north.Close()
	defer p.south.Close()

	for {
		message, err := p.south.RecvMsg()
		if err != nil {
			log.Println(err)
			return
		}
		p.north.SendMsg(message)
	}
}

func (p *Engine) ProcessSouthbound() {
	defer p.north.Close()
	defer p.south.Close()

	for {
		buffer, err := p.north.RecvMsg()
		util.CheckError(err)
		err = p.south.SendMsg(buffer)
		util.CheckError(err)
	}
}
