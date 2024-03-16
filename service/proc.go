package service

import (
	"hdnprxy/util"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Service struct {
	content *Content
}

type Content struct {
	Homefile string
	Basedir  string
}

// Key - if key is 123 and we see /123/ then we will proxy the request to the proxyendpoint
type ProxyContent struct {
	Proxyendpoint string
}

type Proxies struct {
	Proxies map[string]ProxyContent
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
	}
	util.ReadConfig(cfgpath, configs)

	svc := &Service{
		content: configs["content"].(*Content),
	}
	http.HandleFunc("/", svc.HandleHtml)
	http.HandleFunc("/home", svc.HandleHome)

	return svc
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
