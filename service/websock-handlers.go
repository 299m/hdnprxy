package service

import (
	"crypto/tls"
	"fmt"
	"github.com/299m/util/util"
	"hdnprxy/proxy"
	relay2 "hdnprxy/relay"
	"log"
	"net/http"
)

///WARNING - these hanven't been tested yet

// / Raw websocket proxy - north and south
func (p *Service) HandleWsProxy(w http.ResponseWriter, req *http.Request, proxycfg *ProxyContent) {
	defer util.OnPanic(w)
	fmt.Println("Handling ws proxy")
	north := relay2.NewWebSockRelay(proxycfg.Proxyendpoint, p.getTimeout(proxycfg))
	err := north.Connect()
	if err != nil {
		log.Println("Unable to connect ", err)
		http.Error(w, "Server error", 500)
		return
	}
	//defer relay.Close()
	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Println(err)
		return
	}
	south := relay2.NewWebSockRelayFromConn(conn, p.timeout)
	processor := proxy.NewEngine(north, south, p.proxycfg, p.rulesproc)
	go processor.ProcessNorthbound()
	go processor.ProcessSouthbound()
}

// Websocket to the north - raw tcp to the south
func (p *Service) HandleNetWSProxy(w http.ResponseWriter, req *http.Request, proxycfg *ProxyContent) {
	defer util.OnPanic(w)
	fmt.Println("Handling net ws proxy")
	north := relay2.NewWebSockRelay(proxycfg.Proxyendpoint, p.getTimeout(proxycfg))
	err := north.Connect()
	if err != nil {
		log.Println("Unable to connect ", err)
		http.Error(w, "Server error", 500)
		return

	}
	//defer relay.Close()
	conn, pendingdata, err := p.hijack(w)
	util.CheckError(err)
	// Only accept secure connections - make sure this is a tls connection
	south := relay2.NewClientFromConn(conn.(*tls.Conn), p.getTimeout(proxycfg))
	north.SendMsg(pendingdata)
	processor := proxy.NewEngine(north, south, p.proxycfg, p.rulesproc)
	go processor.ProcessNorthbound()
	go processor.ProcessSouthbound()
}

// Raw tcp to the north - websocket to the south
func (p *Service) HandleWSNetProxy(w http.ResponseWriter, req *http.Request, proxycfg *ProxyContent) {
	defer util.OnPanic(w)
	fmt.Println("Handling ws net proxy")
	conn, err := upgrader.Upgrade(w, req, nil)
	util.CheckError(err)
	north := relay2.NewClient(proxycfg.Proxyendpoint, p.getTimeout(proxycfg))
	north.AllowCert(p.allowedcacerts)
	err = north.Connect()
	util.CheckError(err)
	south := relay2.NewWebSockRelayFromConn(conn, p.getTimeout(proxycfg))
	processor := proxy.NewEngine(north, south, p.proxycfg, p.rulesproc)
	go processor.ProcessNorthbound()
	go processor.ProcessSouthbound()
}
