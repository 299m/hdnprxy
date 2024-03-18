package service

import (
	"crypto/tls"
	"errors"
	"github.com/gorilla/websocket"
	"hdnprxy/configs"
	"hdnprxy/proxy"
	relay2 "hdnprxy/relay"
	"hdnprxy/util"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Service struct {
	content *Content
	proxies *Proxies

	proxyparam string
	proxyroute string
	timeout    time.Duration
	buffersize int
}

type Content struct {
	Homefile string
	Basedir  string
}

// Key - if key is 123 and we see /123/ then we will proxy the request to the proxyendpoint
type ProxyContent struct {
	Proxyendpoint string
	Type          string // currently "ws", "net", "n-ws" (websock north), "s-ws" (websock south), may try to support http in the future
}

type General struct {
	ProxyParam       string
	ProxyRoute       string
	Timeout          string
	ProxyBufferSizes int
	AllowedCACerts   []string
}

func (g *General) Expand() {
	for i, cert := range g.AllowedCACerts {
		g.AllowedCACerts[i] = os.ExpandEnv(cert)
	}
}

type Proxies struct {
	Proxies map[string]*ProxyContent
}

func (p *Proxies) Expand() {
}

func (c *Content) Expand() {
	c.Basedir = os.ExpandEnv(c.Basedir)
	c.Homefile = os.ExpandEnv(c.Homefile)
}

func NewService(cfgpath string) *Service {
	// Read the config file
	configs := map[string]util.Expandable{
		"content": &Content{},
		"proxies": &Proxies{},
		"general": &General{},
	}
	util.ReadConfig(cfgpath, configs)
	timeout, err := time.ParseDuration(configs["general"].(*General).Timeout)
	util.CheckError(err)

	svc := &Service{
		content:    configs["content"].(*Content),
		proxies:    configs["proxies"].(*Proxies),
		proxyparam: configs["general"].(*General).ProxyParam, // e.g. name
		proxyroute: configs["general"].(*General).ProxyRoute, // e.g. /proxy
		timeout:    timeout,
		buffersize: configs["general"].(*General).ProxyBufferSizes,
	}
	http.HandleFunc("/", svc.HandleHtml)
	http.HandleFunc("/home", svc.HandleHome)
	http.HandleFunc(svc.proxyroute, svc.HandleProxy)

	return svc
}

func (p *Service) hijack(w http.ResponseWriter) (net.Conn, error) {
	h, ok := w.(http.Hijacker)
	if !ok {
		return nil, errors.New("Hijacking not supported")
	}
	conn, brw, err := h.Hijack()
	if err != nil {
		return nil, err
	}
	if brw.Reader.Buffered() > 0 {
		if err := conn.Close(); err != nil {
			log.Printf("websocket: failed to close network connection: %v", err)
		}
		return nil, errors.New("Client sent data before handshake is complete")
	}
	return conn, nil
}

// / Raw tcp proxy - north and south
func (p *Service) HandleNetProxy(w http.ResponseWriter, req *http.Request, proxycfg *ProxyContent) {
	relay := relay2.NewClient(proxycfg.Proxyendpoint, p.timeout)
	err := relay.Connect()
	if err != nil {
		log.Println("Unable to ocnnect ", err)
		http.Error(w, "Server error", 500)
		return
	}
	//defer relay.Close()
	conn, err := p.hijack(w)
	//// Only accept secure connections - make sure this is a tls connection
	south := relay2.NewClientFromConn(conn.(*tls.Conn), p.timeout)
	processor := proxy.NewEngine(relay, south, &proxy.Config{Buffersize: p.buffersize})
	go processor.ProcessNorthbound()
	go processor.ProcessSouthbound()
}

// / Raw websocket proxy - north and south
func (p *Service) HandleWsProxy(w http.ResponseWriter, req *http.Request, proxycfg *ProxyContent) {
	north := relay2.NewWebSockRelay(proxycfg.Proxyendpoint, p.timeout)
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
	processor := proxy.NewEngine(north, south, &proxy.Config{Buffersize: p.buffersize})
	go processor.ProcessNorthbound()
	go processor.ProcessSouthbound()
}

// Websocket to the north - raw tcp to the south
func (p *Service) HandleNetWSProxy(w http.ResponseWriter, req *http.Request, proxycfg *ProxyContent) {
	north := relay2.NewWebSockRelay(proxycfg.Proxyendpoint, p.timeout)
	err := north.Connect()
	if err != nil {
		log.Println("Unable to connect ", err)
		http.Error(w, "Server error", 500)
		return

	}
	//defer relay.Close()
	conn, err := p.hijack(w)
	util.CheckError(err)
	// Only accept secure connections - make sure this is a tls connection
	south := relay2.NewClientFromConn(conn.(*tls.Conn), p.timeout)
	processor := proxy.NewEngine(north, south, &proxy.Config{Buffersize: p.buffersize})
	go processor.ProcessNorthbound()
	go processor.ProcessSouthbound()
}

// Raw tcp to the north - websocket to the south
func (p *Service) HandleWSNetProxy(w http.ResponseWriter, req *http.Request, proxycfg *ProxyContent) {
	conn, err := upgrader.Upgrade(w, req, nil)
	util.CheckError(err)
	north := relay2.NewClient(proxycfg.Proxyendpoint, p.timeout)
	err = north.Connect()
	util.CheckError(err)
	south := relay2.NewWebSockRelayFromConn(conn, p.timeout)
	processor := proxy.NewEngine(north, south, &proxy.Config{Buffersize: p.buffersize})
	go processor.ProcessNorthbound()
	go processor.ProcessSouthbound()
}

// Create a http handler function for all proxy keys
func (p *Service) HandleProxy(res http.ResponseWriter, req *http.Request) {
	defer util.OnPanic(res)
	///read the proxy param and see if it matches any of the keys
	proxykey := req.URL.Query().Get(p.proxyparam)

	proxy, ok := p.proxies.Proxies[proxykey]
	if !ok {
		http.Error(res, "Not found", 400)
		return
	}
	switch proxy.Type {
	case "net":
		p.HandleNetProxy(res, req, proxy)
	case "ws":
		p.HandleWsProxy(res, req, proxy)
	case "n-ws":
		p.HandleNetWSProxy(res, req, proxy)
	case "s-ws":
		p.HandleWSNetProxy(res, req, proxy)
	default:
		log.Println("Invalid proxy type", proxy.Type)
		http.Error(res, "Server error", 500)
	}
}

func (p *Service) HandleHtml(res http.ResponseWriter, req *http.Request) {
	defer util.OnPanic(res)
	resppath := req.URL.Path[len("/"):]
	if resppath == "" {
		p.HandleHome(res, req)
		return
	}

	if strings.Contains(resppath, "..") {
		http.Error(res, "Invalid path", 400)
		return
	}
	file := filepath.Join(p.content.Basedir, resppath)
	stat, err := os.Stat(file)
	if err != nil {
		http.Error(res, "File not found", 404)
		return
	}
	f, err := os.Open(file)
	util.CheckError(err)
	defer f.Close()

	http.ServeContent(res, req, resppath, stat.ModTime(), f)
	http.ServeFile(res, req, p.content.Homefile)
}

// / Create a http handler function for the /home endpoint
func (p *Service) HandleHome(res http.ResponseWriter, req *http.Request) {
	defer util.OnPanic(res)
	stats, err := os.Stat(p.content.Homefile)
	util.CheckError(err)
	f, err := os.Open(p.content.Homefile)
	util.CheckError(err)
	defer f.Close()

	http.ServeContent(res, req, "home", stats.ModTime(), f)
}

func (p *Service) HandleTunnel(conn net.Conn, proxycontent *ProxyContent) {
	// / Create a new client from the connection
	south := relay2.NewClientFromConn(conn.(*tls.Conn), p.timeout)

	north := relay2.NewClient(proxycontent.Proxyendpoint, p.timeout)
	err := north.Connect()
	util.CheckError(err)
	processor := proxy.NewEngine(north, south, &proxy.Config{Buffersize: p.buffersize})
	go processor.ProcessNorthbound()
	go processor.ProcessSouthbound()

}

func ProxyListenAndServe(servercfg *configs.TlsConfig, svc *Service) {

	/// Start a tls listener
	/// Load the server certs
	cer, err := tls.LoadX509KeyPair(servercfg.Cert, servercfg.Key)
	util.CheckError(err)
	tlsconfig := &tls.Config{
		Certificates: []tls.Certificate{cer},
	}

	listener, err := tls.Listen("tcp", ":"+servercfg.Port, tlsconfig)
	util.CheckError(err)
	for {
		conn, err := listener.Accept()
		util.CheckError(err)
		svc.HandleTunnel(conn, svc.proxies.Proxies["tunnel"]) /// this should return after setting up the tunnel
	}

}

func ListenAndServeHttps(servercfg *configs.TlsConfig) {
	// Start the server
	err := http.ListenAndServeTLS(":"+servercfg.Port, servercfg.Cert, servercfg.Key, nil)

	util.CheckError(err)
}

func ListenAndServeTls(cfgpath string) {
	svc := NewService(cfgpath)
	servercfg := &configs.TlsConfig{}
	tlsconfig := map[string]util.Expandable{
		"tls": servercfg,
	}
	util.ReadConfig(cfgpath, tlsconfig)

	if tlsconfig["tls"].(*configs.TlsConfig).IsProxy {
		ProxyListenAndServe(servercfg, svc)
	} else {
		ListenAndServeHttps(servercfg)
	}
}
