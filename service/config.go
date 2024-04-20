package service

import (
	"os"
	"strings"
)

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
		c.Downloaddir = "downloads"
	}
}

// Key - if key is 123 and we see /123/ then we will proxy the request to the proxyendpoint
type ProxyContent struct {
	Proxyendpoint string
	Type          string // currently "ws", "net", "raw", "n-ws" (websock north), "s-ws" (websock south), may try to support http in the future
	Timeout       string
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
	g.Timeout = os.ExpandEnv(g.Timeout)

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
		proxy.Timeout = os.ExpandEnv(proxy.Timeout)
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
