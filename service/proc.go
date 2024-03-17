package service

import (
	"bufio"
	"github.com/gorilla/websocket"
	"hdnprxy/proxy"
	relay2 "hdnprxy/relay"
	"hdnprxy/util"
	"log"
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
	Type          string // currently "ws" or "net", may try to support http in the future
}

type General struct {
	ProxyParam       string
	ProxyRoute       string
	Timeout          string
	ProxyBufferSizes int
}

func (g *General) Expand() {
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

func (p *Service) HandleNetProxy(w http.ResponseWriter, req *http.Request, proxycfg *ProxyContent) {
	relay := relay2.NewClient(proxycfg.Proxyendpoint, p.timeout)
	err := relay.Connect()
	if err != nil {
		log.Println("Unable to ocnnect ", err)
		http.Error(w, "Server error", 500)
		return
	}
	//defer relay.Close()
	h, ok := w.(http.Hijacker)
	if !ok {
		log.Println("Hijacking not supported")
		http.Error(w, "Server error", 500)
		return
	}
	var brw *bufio.ReadWriter
	conn, brw, err := h.Hijack()
	if err != nil {
		log.Println("Hijack failed ", err)
		http.Error(w, "Server error", 500)
		return
	}
	if brw.Reader.Buffered() > 0 {
		if err := conn.Close(); err != nil {
			log.Printf("websocket: failed to close network connection: %v", err)
		}
		log.Println("Client sent data before handshake is complete")
		http.Error(w, "Server error", 500)
	}
	processor := proxy.NewEngine(relay, conn, &proxy.Config{Buffersize: p.buffersize})
	go processor.ProcessNorthbound()
	go processor.ProcessSouthbound()
}

// Create a http handler function for all proxy keys
func (p *Service) HandleProxy(res http.ResponseWriter, req *http.Request) {
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
	}
}

func (p *Service) HandleHtml(res http.ResponseWriter, req *http.Request) {
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
	stats, err := os.Stat(p.content.Homefile)
	util.CheckError(err)
	f, err := os.Open(p.content.Homefile)
	util.CheckError(err)
	defer f.Close()

	http.ServeContent(res, req, "home", stats.ModTime(), f)
}

type TlsConfig struct {
	Cert string
	Key  string
	Port string
}

func (t *TlsConfig) Expand() {
	t.Cert = os.ExpandEnv(t.Cert)
	t.Key = os.ExpandEnv(t.Key)
}

func ListenAndServeTls(cfgpath string) {
	// Read the config file
	NewService(cfgpath)
	servercfg := &TlsConfig{}
	tlsconfig := map[string]util.Expandable{
		"tls": servercfg,
	}
	util.ReadConfig(cfgpath, tlsconfig)
	// Start the server
	err := http.ListenAndServeTLS(servercfg.Port, servercfg.Cert, servercfg.Key, nil)
	util.CheckError(err)
}
