package service

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/299m/util/util"
	"github.com/gorilla/websocket"
	"hdnprxy/configs"
	"hdnprxy/proxy"
	"hdnprxy/rules"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	CONNNET          = "net"
	CONNRAWTCP       = "raw"
	CONNWEBSOCK      = "ws"
	CONNNETTOWEBSOCK = "n-ws"
	CONNWEBSOCKNET   = "s-ws"
	CONNUDP          = "udp"
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

func NewService(cfgpath string) *Service {
	// Read the config file
	configs := map[string]util.Expandable{
		"content":       &Content{},
		"proxies":       &Proxies{},
		"general":       &General{},
		"engine":        &proxy.Config{},
		"connect-rules": &rules.ConnectConfig{},
	}
	fmt.Println("Starting server with cfg path ", cfgpath)
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
	case CONNNET, CONNRAWTCP:
		p.HandleRemoteTunnel(res, req, proxy)
	case CONNWEBSOCK:
		p.HandleWsProxy(res, req, proxy)
	case CONNNETTOWEBSOCK:
		p.HandleNetWSProxy(res, req, proxy)
	case CONNWEBSOCKNET:
		p.HandleWSNetProxy(res, req, proxy)
	case CONNUDP:
		p.HandleRemoteUdp(res, req, proxy)
	default:
		log.Println("Invalid proxy type", proxy.Type)
		http.Error(res, "Server error", 500)
	}
}

func checkFilePath(resppath string) bool {
	if !filepath.IsLocal(resppath) ||
		strings.Contains(resppath, "..") || strings.Contains(resppath, "~") || strings.Contains(resppath, "*") {
		log.Println("Invalid path", resppath)
		return false
	}
	return true
}

func (p *Service) HandleHtml(res http.ResponseWriter, req *http.Request) {
	defer util.OnPanic(res)
	resppath := req.URL.Path[len("/"):]
	if resppath == "" {
		p.HandleHome(res, req)
		return
	}

	if !checkFilePath(resppath) {
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

	fmt.Println("looking for down load dir", p.downloadsdir, " in ", resppath)
	if strings.HasPrefix(resppath, p.downloadsdir) {
		fmt.Println("Downloading file", file)
		res.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(file))
		buf := make([]byte, stat.Size())
		n, err := f.Read(buf)
		util.CheckError(err)
		if n != int(stat.Size()) {
			log.Panicln("File not read completely", n, stat.Size())
		}
		res.Write(buf)
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
		svc.HandleLocalTunnel(conn, svc.proxies.Proxies["tunnel"], tunnel) /// this should return after setting up the tunnel
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
	svc.HandleLocalTunnel(conn, svc.proxies.Proxies["tunnel"], tunnel) /// this should return after setting up the tunnel
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

// // Local side of the tunnel
func ListenAndServeUDP(servercfg *configs.TlsConfig, s *Service, tunnel *Tunnel) {
	fmt.Println("Listening for UDP packets")
	udpaddr, err := net.ResolveUDPAddr("udp", ":"+servercfg.Port)
	util.CheckError(err)
	conn, err := net.ListenUDP("udp", udpaddr)
	util.CheckError(err)

	for { /// Not sure if this will break on a ctrl-C - we'll find out
		//// For UDP create a single TCP connection to the remote end of the tunnel
		/// this should block
		s.HandleLocalUdp(conn, s.proxies.Proxies["tunnel"], tunnel)
		/// Need to handle reconnection - if ctrl-C hasn't been pressed
	}
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
	} else if tlsconfig["tls"].(*configs.TlsConfig).IsUdpProxy {
		ListenAndServeUDP(servercfg, svc, tunnel)
	} else {
		log.Panicln("Invalid tls config, one of IsProxy or IsHttps must be set")
	}
}
