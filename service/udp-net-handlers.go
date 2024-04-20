package service

import (
	"crypto/tls"
	"fmt"
	"github.com/299m/util/util"
	"hdnprxy/proxy"
	relay2 "hdnprxy/relay"
	"net"
	"net/http"
)

// Listen for incoming UDP packets
func (p *Service) HandleLocalUdp(udpconn *net.UDPConn, proxycontent *ProxyContent, tunnel *Tunnel) {
	defer util.OnPanicFunc()
	// make a tcp/tls tunnel to the remote server
	// using the relay client
	// and then use a udp relay to receive from local
	// If the udp client has an error receiving, then return, else reconnect to the remote with tcp/tls

	north := relay2.NewTunnelClient(proxycontent.Proxyendpoint, p.timeout, tunnel.Paramname, tunnel.Paramval)
	north.AllowCert(p.allowedcacerts)
	err := north.Connect()
	util.CheckError(err)
	south := relay2.NewUdpRelay(udpconn, p.buffersize)
	processor := proxy.NewEngine(north, south, p.proxycfg, p.rulesproc)
	fmt.Println("UDP Tunnel setup ... start the engine")

	go processor.ProcessNorthbound()
	//// Don't start another goroutine for the southbound - the tunnel should remain open
	processor.ProcessSouthbound()

}

func (p *Service) HandleRemoteUdp(w http.ResponseWriter, req *http.Request, proxycfg *ProxyContent) {
	defer util.OnPanicFunc()

	/// Create the north UPD relay
	north := relay2.NewUdpClient(proxycfg.Proxyendpoint, p.getTimeout(proxycfg))
	if p.proxycfg.Lognorth {
		north.EnableDebugLogs(true, "svc-net-north")
	}

	conn, pendingdata, err := p.hijack(w)
	util.CheckError(err)

	sendResponse(conn, "", 200) /// after this, go to raw tcp/tls

	//// Only accept secure connections - make sure this is a tls connection
	south := relay2.NewClientFromConn(conn.(*tls.Conn), p.getTimeout(proxycfg))
	if p.proxycfg.Logsouth { /// slightly messy - but lets see whats beign sent
		south.EnableDebugLogs(true, "svc-net-north")
	}
	err = north.SendMsg(pendingdata)
	util.CheckError(err)

	processor := proxy.NewEngine(north, south, p.proxycfg, p.rulesproc)
	go processor.ProcessNorthbound()
	//// Don't start another goroutine for the southbound - the tunnel should remain open
	processor.ProcessSouthbound()
}
