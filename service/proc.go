package service

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/299m/util/util"
	"github.com/gorilla/websocket"
	"hdnprxy/configs"
	"hdnprxy/proxy"
	relay2 "hdnprxy/relay"
	"hdnprxy/rules"
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

	allowedcacerts []string /// If we are using our own ca cert. This can be useful if we don't trust the ceritifcate store of our system
	proxycfg       *proxy.Config
	debuglogs      bool

	downloadsdir string

	rulesproc *rules.Processor
}

type Content struct {
	Homefile    string
	Basedir     string
	Downloaddir string
}

func (c *Content) Expand() {
	c.Basedir = os.ExpandEnv(c.Basedir)
	c.Homefile = os.ExpandEnv(c.Homefile)
	c.Downloaddir = os.ExpandEnv(c.Downloaddir)
	if c.Downloaddir == "" {
		c.Downloaddir = filepath.Join(c.Basedir, "downloads")
	}
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
	Debuglogs        bool

	IsLocal bool //// Set this if this is the local side of a tunnel

}

func (g *General) Expand() {
	g.ProxyParam = os.ExpandEnv(g.ProxyParam)
	g.ProxyRoute = os.ExpandEnv(g.ProxyRoute)

	//// Do any other expansion above this
	if len(g.AllowedCACerts) == 1 && strings.Contains(g.AllowedCACerts[0], ",") {
		g.AllowedCACerts = strings.Split(g.AllowedCACerts[0], ",")
		return
	}
	for i, cert := range g.AllowedCACerts {
		g.AllowedCACerts[i] = os.ExpandEnv(cert)
	}
}

type Proxies struct {
	Proxies map[string]*ProxyContent
}

func (p *Proxies) Expand() {
	//shallow copy the map first - otherwise we're iterating and changing it at the same time
	proxies := make(map[string]*ProxyContent)
	for key, proxy := range p.Proxies {
		truekey := os.ExpandEnv(key)
		proxy.Proxyendpoint = os.ExpandEnv(proxy.Proxyendpoint)
		proxies[truekey] = proxy
	}
	p.Proxies = proxies
}

type Tunnel struct {
	Paramname string //// These are the triggers to start the tunnel on the config side
	Paramval  string
}

func (t *Tunnel) Expand() {
	t.Paramval = os.ExpandEnv(t.Paramval)
	t.Paramname = os.ExpandEnv(t.Paramname)
}

func NewService(cfgpath string) *Service {
	// Read the config file
	configs := map[string]util.Expandable{
		"content":       &Content{},
		"proxies":       &Proxies{},
		"general":       &General{},
		"engine":        &proxy.Config{},
		"connect-rules": &rules.ConnectConfig{},
	}
	util.ReadConfig(cfgpath, configs)
	timeout, err := time.ParseDuration(configs["general"].(*General).Timeout)
	util.CheckError(err)

	svc := &Service{
		content:        configs["content"].(*Content),
		proxies:        configs["proxies"].(*Proxies),
		proxyparam:     configs["general"].(*General).ProxyParam, // e.g. name
		proxyroute:     configs["general"].(*General).ProxyRoute, // e.g. /proxy
		timeout:        timeout,
		buffersize:     configs["general"].(*General).ProxyBufferSizes,
		allowedcacerts: configs["general"].(*General).AllowedCACerts,
		proxycfg:       configs["engine"].(*proxy.Config),
		debuglogs:      configs["general"].(*General).Debuglogs,
		downloadsdir:   configs["content"].(*Content).Downloaddir,
		rulesproc:      rules.NewProcessor(configs["connect-rules"].(*rules.ConnectConfig)),
	}
	if !configs["general"].(*General).IsLocal {
		http.HandleFunc("/", svc.HandleHtml)
		http.HandleFunc("/home", svc.HandleHome)
		fmt.Println("Proxy route", svc.proxyroute)
		http.HandleFunc(svc.proxyroute, svc.HandleProxy)
	}
	svc.DebugLog("Proxies ", svc.proxies.Proxies)

	return svc
}

func (p *Service) DebugLog(msg ...any) {
	if p.debuglogs {
		fmt.Print(time.Now(), ">")
		fmt.Println(msg...)
	}
}

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

// / Raw tcp proxy - north and south
func (p *Service) HandleNetProxy(w http.ResponseWriter, req *http.Request, proxycfg *ProxyContent) {
	defer util.OnPanic(w)
	fmt.Println("Handling net proxy")

	north := relay2.NewClient(proxycfg.Proxyendpoint, p.timeout)
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
	south := relay2.NewClientFromConn(conn.(*tls.Conn), p.timeout)
	if p.proxycfg.Lognorth { /// slightly messy - but lets see whats beign sent
		north.EnableDebugLogs(true, "svc-net-north")
	}
	p.DebugLog("Sending pending data to north", string(pendingdata))
	north.SendMsg(pendingdata)

	processor := proxy.NewEngine(north, south, p.proxycfg, p.rulesproc)
	go processor.ProcessNorthbound()
	go processor.ProcessSouthbound()
}

// / Raw websocket proxy - north and south
func (p *Service) HandleWsProxy(w http.ResponseWriter, req *http.Request, proxycfg *ProxyContent) {
	defer util.OnPanic(w)
	fmt.Println("Handling ws proxy")
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
	processor := proxy.NewEngine(north, south, p.proxycfg, p.rulesproc)
	go processor.ProcessNorthbound()
	go processor.ProcessSouthbound()
}

// Websocket to the north - raw tcp to the south
func (p *Service) HandleNetWSProxy(w http.ResponseWriter, req *http.Request, proxycfg *ProxyContent) {
	defer util.OnPanic(w)
	fmt.Println("Handling net ws proxy")
	north := relay2.NewWebSockRelay(proxycfg.Proxyendpoint, p.timeout)
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
	south := relay2.NewClientFromConn(conn.(*tls.Conn), p.timeout)
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
	north := relay2.NewClient(proxycfg.Proxyendpoint, p.timeout)
	north.AllowCert(p.allowedcacerts)
	err = north.Connect()
	util.CheckError(err)
	south := relay2.NewWebSockRelayFromConn(conn, p.timeout)
	processor := proxy.NewEngine(north, south, p.proxycfg, p.rulesproc)
	go processor.ProcessNorthbound()
	go processor.ProcessSouthbound()
}

// Create a http handler function for all proxy keys
func (p *Service) HandleProxy(res http.ResponseWriter, req *http.Request) {
	defer util.OnPanic(res)
	p.DebugLog("Http handle proxy")
	///read the proxy param and see if it matches any of the keys
	dec := json.NewDecoder(req.Body)
	data := make(map[string]string)
	err := dec.Decode(&data)
	util.CheckError(err)
	proxykey, ok := data[p.proxyparam]
	if !ok || proxykey == "" {
		log.Println("No proxy param in the request, or the value is empty")
		http.Error(res, "Not found", 404)
		return
	}

	proxy, ok := p.proxies.Proxies[proxykey]
	if !ok {
		log.Println("Proxy not found", proxykey)
		http.Error(res, "Not found", 404)
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

	if !filepath.IsLocal(resppath) ||
		strings.Contains(resppath, "..") || strings.Contains(resppath, "~") || strings.Contains(resppath, "*") {
		log.Println("Invalid path", resppath)
		http.Error(res, "Invalid path", 400)
		return
	}
	file := filepath.Join(p.content.Basedir, resppath)
	stat, err := os.Stat(file)
	if err != nil {
		log.Println("File not found", file, err)
		http.Error(res, "File not found", 404)
		return
	}
	f, err := os.Open(file)
	util.CheckError(err)
	defer f.Close()

	if strings.Contains(file, p.downloadsdir) {
		p.DebugLog("Downloading file", file)
		http.ServeFile(res, req, file)
		return
	}
	http.ServeContent(res, req, resppath, stat.ModTime(), f)
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

func (p *Service) HandleTunnel(conn net.Conn, proxycontent *ProxyContent, tunnel *Tunnel) {
	defer util.OnPanicFunc()
	// / Create a new client from the connection
	fmt.Println("Handling tunnel")
	/// the first response should not have any body - it's simply a status response

	south := relay2.NewClientFromConn(conn, p.timeout)

	north := relay2.NewTunnelClient(proxycontent.Proxyendpoint, p.timeout, tunnel.Paramname, tunnel.Paramval)
	north.AllowCert(p.allowedcacerts)
	err := north.Connect()
	util.CheckError(err)
	processor := proxy.NewEngine(north, south, p.proxycfg, p.rulesproc)
	go processor.ProcessNorthbound()
	go processor.ProcessSouthbound()
	fmt.Println("Tunnel setup complete")
}

func ProxyListenAndServe(servercfg *configs.TlsConfig, svc *Service, tunnel *Tunnel) {

	/// Start a tls listener
	/// Load the server certs
	cer, err := tls.LoadX509KeyPair(servercfg.Cert, servercfg.Key)
	util.CheckError(err)
	tlsconfig := &tls.Config{
		Certificates: []tls.Certificate{cer},
	}

	fmt.Println("Starting tunnel server on port", servercfg.Port)
	listener, err := tls.Listen("tcp", ":"+servercfg.Port, tlsconfig)
	util.CheckError(err)
	for {
		conn, err := listener.Accept()
		util.CheckError(err)
		svc.HandleTunnel(conn, svc.proxies.Proxies["tunnel"], tunnel) /// this should return after setting up the tunnel
	}

}

func sendResponse(conn net.Conn, status string, statuscode int) {
	resp := http.Response{
		Status:        status,
		StatusCode:    statuscode,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        http.Header{},
		ContentLength: 0,
		Close:         false,
		Uncompressed:  false,
	}
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	err := resp.Write(conn)

	util.CheckError(err)
}

/* OBSOLETE - pass to the proxy

func HandleConnect(conn net.Conn) {
	fmt.Println("Handling connect")
	///Read as much data as is available, check it matches a CONNECT header and then respond with success, else with a 400
	reader := bufio.NewReader(conn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		sendResponse(conn, "Bad request", http.StatusBadRequest)
		return
	}
	///Check we have a host and port
	host, port, err := net.SplitHostPort(req.Host)
	util.CheckError(err)
	if len(host) == 0 || len(port) == 0 {
		fmt.Println("Invalid host or port", host, port)
		sendResponse(conn, "Bad request", http.StatusBadRequest)
		return

	}
	fmt.Println("Host", host, "Port", port)
	sendResponse(conn, "Success", http.StatusOK)
	util.CheckError(err)

}
*/

func handleIncomingNetConn(conn net.Conn, err error, upgradetotls bool, servercfg *configs.TlsConfig, svc *Service, tunnel *Tunnel) {
	defer util.OnPanicFunc()
	util.CheckError(err)
	//HandleConnect(conn)
	if upgradetotls {
		/// Start a tls listener
		/// Load the server certs
		cer, err := tls.LoadX509KeyPair(servercfg.Cert, servercfg.Key)
		util.CheckError(err)
		tlsconfig := &tls.Config{
			Certificates: []tls.Certificate{cer},
		}
		conn = tls.Server(conn, tlsconfig)
	}
	svc.HandleTunnel(conn, svc.proxies.Proxies["tunnel"], tunnel) /// this should return after setting up the tunnel
}

// /Run this within your local network - the HTTP Connect is plain text over the network
func ProxyListenAndServeTcpTls(servercfg *configs.TlsConfig, svc *Service, tunnel *Tunnel, upgradetotls bool) {

	fmt.Println("Starting tunnel server on port", servercfg.Port, "with tls", upgradetotls)
	listener, err := net.Listen("tcp", ":"+servercfg.Port)
	util.CheckError(err)
	for {
		conn, err := listener.Accept()
		go handleIncomingNetConn(conn, err, upgradetotls, servercfg, svc, tunnel)
	}

}

func ListenAndServeHttps(servercfg *configs.TlsConfig) {
	// Start the server
	fmt.Println("Starting server on port", servercfg.Port, "with cert", servercfg.Cert, "and key", servercfg.Key)
	err := http.ListenAndServeTLS(":"+servercfg.Port, servercfg.Cert, servercfg.Key, nil)

	util.CheckError(err)
}

func ListenAndServeTls(cfgpath string) {
	svc := NewService(cfgpath)
	servercfg := &configs.TlsConfig{}
	tunnel := &Tunnel{}
	tlsconfig := map[string]util.Expandable{
		"tls":    servercfg,
		"tunnel": tunnel,
	}
	util.ReadConfig(cfgpath, tlsconfig)

	if tlsconfig["tls"].(*configs.TlsConfig).IsProxy {
		ProxyListenAndServe(servercfg, svc, tunnel)
	} else if tlsconfig["tls"].(*configs.TlsConfig).IsHttps {
		ListenAndServeHttps(servercfg)
	} else if tlsconfig["tls"].(*configs.TlsConfig).IsTlsProxy {
		ProxyListenAndServeTcpTls(servercfg, svc, tunnel, true)
	} else if tlsconfig["tls"].(*configs.TlsConfig).IsTcpProxy {
		ProxyListenAndServeTcpTls(servercfg, svc, tunnel, false)
	} else {
		log.Panicln("Invalid tls config, one of IsProxy or IsHttps must be set")
	}
}
