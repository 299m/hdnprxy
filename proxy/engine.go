package proxy

import (
	"github.com/299m/util/util"
	"hdnprxy/relay"
	"log"
	"time"
)

type Config struct {
	Buffersize int
	Logdebug   bool
	Lognorth   bool
	Logsouth   bool
}

func (c *Config) Expand() {
}

type Engine struct {
	north relay.Relay
	south relay.Relay

	cfg *Config
}

func (p *Engine) logDebug(message string, preffix string) {
	if p.cfg.Logdebug {
		log.Println(time.Now(), ">", preffix, ">", message)
	}
}

func NewEngine(north relay.Relay, south relay.Relay, cfg *Config) *Engine {
	return &Engine{
		north: north,
		south: south,
		cfg:   cfg,
	}
}

func (p *Engine) ProcessNorthbound() {
	defer util.OnPanicFunc()
	defer p.north.Close()
	defer p.south.Close()

	for {
		message, err := p.south.RecvMsg()
		if err != nil {
			log.Println(err)
			return
		}
		p.logDebug(string(message), "n")
		p.north.SendMsg(message)
	}
}

func (p *Engine) ProcessSouthbound() {
	defer util.OnPanicFunc()
	defer p.north.Close()
	defer p.south.Close()

	for {
		buffer, err := p.north.RecvMsg()
		util.CheckError(err)
		p.logDebug(string(buffer), "s")
		err = p.south.SendMsg(buffer)
		util.CheckError(err)
	}
}
