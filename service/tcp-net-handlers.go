package service

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/299m/util/util"
	"hdnprxy/proxy"
	relay2 "hdnprxy/relay"
	"log"
	"net"
	"net/http"
	"time"
)

func (p *Service) hijack(w http.ResponseWriter) (c net.Conn, pendingdata []byte, err error) {
	p.DebugLog("Hijacking the http connection")
	h, ok := w.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("Hijacking not supported")
	}
	conn, brw, err := h.Hijack()
	if err != nil {
		return nil, nil, err
	}
	buffered := brw.Reader.Buffered()
	pendingdata = make([]byte, buffered)
	if buffered > 0 {
		n, err := brw.Read(pendingdata)
		util.CheckError(err)
		if n != buffered {
			log.Panicln("Buffered data not read completely or something ", n, buffered)
		}
	}
	return conn, pendingdata, nil
}

func (p *Service) getTimeout(proxycfg *ProxyContent) time.Duration {
	timeout := p.timeout
	if proxycfg.Timeout != "" {
		var err error
		timeout, err = time.ParseDuration(proxycfg.Timeout)
		util.CheckError(err)
	}
	return timeout
}

// / Raw tcp proxy - north and south
func (p *Service) HandleNetProxy(w http.ResponseWriter, req *http.Request, proxycfg *ProxyContent) {
	defer util.OnPanic(w)
	fmt.Println("Handling net proxy")

	usetls := false
	if proxycfg.Type == CONNNET {
		usetls = true
	}

	north := relay2.NewClientv2(proxycfg.Proxyendpoint, p.getTimeout(proxycfg), usetls)
	north.AllowCert(p.allowedcacerts)
	err := north.Connect()
	if err != nil {
		log.Println("Unable to connect ", err)
		http.Error(w, "Server error", 500)
		return
	}
	//defer relay.Close()
	conn, pendingdata, err := p.hijack(w)
	util.CheckError(err)

	sendResponse(conn, "", 200) /// after this, go to raw tcp/tls

	//// Only accept secure connections - make sure this is a tls connection
	south := relay2.NewClientFromConn(conn.(*tls.Conn), p.getTimeout(proxycfg))
	if p.proxycfg.Lognorth { /// slightly messy - but lets see whats beign sent
		north.EnableDebugLogs(true, "svc-net-north")
	}
	p.DebugLog("Sending pending data to north", string(pendingdata))
	north.SendMsg(pendingdata)

	processor := proxy.NewEngine(north, south, p.proxycfg, p.rulesproc)
	go processor.ProcessNorthbound()
	go processor.ProcessSouthbound()
}
