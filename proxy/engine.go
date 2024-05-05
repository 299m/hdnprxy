package proxy

import (
	"fmt"
	"github.com/299m/util/util"
	"hdnprxy/relay"
	"hdnprxy/routing"
	"hdnprxy/rules"
	"net"
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
	north     relay.Relay
	south     relay.Relay
	logdebug  relay.DebugLog
	rulesproc *rules.Processor

	cfg      *Config
	engineid int64

	udpbuffsize int
	router      routing.UdpRoutes
	islocal     bool
	isudp       bool
	udpmsg      *TunnelMessage
}

func NewEngine(north relay.Relay, south relay.Relay, cfg *Config, rulesproc *rules.Processor) *Engine {
	e := &Engine{
		north:     north,
		south:     south,
		cfg:       cfg,
		engineid:  atomic.AddInt64(&engineid, 1),
		rulesproc: rulesproc,
	}
	if cfg.Logdebug {
		e.logdebug.EnableDebugLogs(true, fmt.Sprint("e-", e.engineid))
	}
	return e
}

func NewUdpEngine(north relay.Relay, south relay.Relay, cfg *Config, rulesproc *rules.Processor, islocal bool, bufsize int) *Engine {
	e := &Engine{
		north:       north,
		south:       south,
		cfg:         cfg,
		engineid:    atomic.AddInt64(&engineid, 1),
		rulesproc:   rulesproc,
		islocal:     islocal,
		isudp:       true,
		udpbuffsize: bufsize,
		udpmsg:      NewTunnelMessage(bufsize),
	}
	if cfg.Logdebug {
		e.logdebug.EnableDebugLogs(true, fmt.Sprint("e-", e.engineid))
	}
	return e
}

func (p *Engine) HandleUdpSend(message []byte, addr net.Addr) (fullmsg []byte) {
	if p.islocal {
		/// If local we expect to get the full message - this is the raw input
		/// we need to put a header in to say where the message came from
		fullmsg = p.udpmsg.Write(message, addr.(*net.UDPAddr))
		p.router.FindOrAddRouteByAddr(addr.(*net.UDPAddr))
	} else { // REMOTE SIDE
		/// If we are not local, we need to lookup the return addr (the UDP address from the local side) and put that in the message
		/// The send UDP side should have filled this in for us based on sending port
		addratlocal := p.router.FindRouteById(int64(addr.(*net.UDPAddr).Port))
		fullmsg = p.udpmsg.Write(message, addratlocal)
	}
	return fullmsg
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
		message, addr, err := p.south.RecvMsg()
		util.CheckError(err)
		rule := p.rulesproc.Allow(message)
		if rule == rules.ALLOW {
			p.logdebug.LogData(string(message), "n")

			/// UPD type messages need a header and route record
			if p.isudp {
				message = p.HandleUdpSend(message, addr)
			}
			err := p.north.SendMsg(message)
			util.CheckError(err)
		} else {
			p.logdebug.LogDebug(fmt.Sprint("Rule blocked message. ", string(message)), "n")
			if rule == rules.DROPFLAT {
				p.logdebug.LogDebug("Dropping message without response and closing the connection", "n")
				break
			} else {
				p.logdebug.LogDebug("Responding with 403 and closing the connection", "n")
				p.north.SendMsg([]byte("HTTP/1.1 403 Forbidden\r\n\r\n"))
				break
			}
		}
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
		buffer, addr, err := p.north.RecvMsg()
		util.CheckError(err)
		p.logdebug.LogData(string(buffer), "s")
		if p.isudp {
			if !p.islocal {
				/// Find the address at the local side and put into the message
				addratlocal := p.router.FindRouteById(int64(addr.(*net.UDPAddr).Port))
				buffer = p.udpmsg.Write(buffer, addratlocal)
			} else {
				/// we are local - get the addr of the buffer and send to that address/port
				msgdata, needmore, localaddr, nextmsgoffset, err := p.udpmsg.Read(buffer)
				util.CheckError(err)
				if needmore {

				}
			}
		}
		err = p.south.SendMsg(buffer)
		util.CheckError(err)
	}
}
