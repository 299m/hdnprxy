package proxy

import (
	"fmt"
	"github.com/299m/util/util"
	"hdnprxy/relay"
	"sync/atomic"
)

type Config struct {
	Buffersize int
	Logdebug   bool
	Lognorth   bool
	Logsouth   bool
}

func (c *Config) Expand() {
}

var engineid int64

type Engine struct {
	north    relay.Relay
	south    relay.Relay
	logdebug relay.DebugLog

	cfg      *Config
	engineid int64
}

func NewEngine(north relay.Relay, south relay.Relay, cfg *Config) *Engine {
	e := &Engine{
		north:    north,
		south:    south,
		cfg:      cfg,
		engineid: atomic.AddInt64(&engineid, 1),
	}
	if cfg.Logdebug {
		e.logdebug.EnableDebugLogs(true, fmt.Sprint("e-", e.engineid))
	}
	return e
}

func (p *Engine) ProcessNorthbound() {
	defer util.OnPanicFunc()
	defer p.north.Close()
	defer p.south.Close()
	if p.cfg.Lognorth {
		p.north.EnableDebugLogs(true, fmt.Sprint("e-", p.engineid, "-n"))
		p.logdebug.LogDebug("Northbound logging enabled", "n")
	}

	for {
		p.logdebug.LogDebug("Waiting for message from south", "n")
		message, err := p.south.RecvMsg()
		util.CheckError(err)
		p.logdebug.LogData(string(message), "n")
		p.north.SendMsg(message)
	}
}

func (p *Engine) ProcessSouthbound() {
	defer util.OnPanicFunc()
	defer p.north.Close()
	defer p.south.Close()
	if p.cfg.Logsouth {
		p.north.EnableDebugLogs(true, fmt.Sprint("e-", p.engineid, "-s"))
		p.logdebug.LogDebug("Southbound logging enabled", "s")
	}

	for {
		p.logdebug.LogDebug("Waiting for message from north", "s")
		buffer, err := p.north.RecvMsg()
		util.CheckError(err)
		p.logdebug.LogData(string(buffer), "s")
		err = p.south.SendMsg(buffer)
		util.CheckError(err)
	}
}
