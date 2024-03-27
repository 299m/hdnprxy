# hdnprxy

Currently this is WIP. Feel free to try it - but it's not been tested or anything.

## Building
Download the project and run `go install` in the root directory.

## Docker
An example Dockerfile can be found in the ./build directory.

## Demo hdnprxy
A demo hdnprxy is available at https://299m.io.

## How to connect
The hdnprxy is designed to form a tunnel from a local hdnprxy to a remote hdnprxy. To connect to this demo remote hdnprxy,
you need to setup the local hdnprxy and configur it to tunnel to the hdnprxy remote server (forming a TLS/TCP tunnel
between the local hdnprxy and the remote hdnprxy). See instructions below.

For this demo, outgoing connections are limited to a few sites (Al Jazeera and Google search) and port 443 only. See below
for how to set up your own server.

Once you have a local hdnpry running, configure you're system settings to proxy internet traffic through the local hdnprxy.
On Ubuntu desktop go to Settings -> Network -> Network Proxy and select Manual in the Network Proxy message box. Set the
HTTPS proxy to 127.0.0.1 and port 20443.


## Setup a local tunnel to the hdnprxy
You can either download the package from 299m.io or build it yourself

### Download the package
(currenctly only Linux Ubuntu packages are available).
Download the zip file from here.
Move the downloaded zip file to your home directory (or copy it from Downloads to your home directory).
Unzip the local.zip and run the script

cd ~/
unzip local.zip
cd local
./run-local.sh

### Build from source
You must have Go installed (see https://go.dev/doc/install).

Clone the repo to your development folder
```
git clone https://github.com/299m/hdnprxy.git
```
Build the hdnprxy
```
cd hdnprxy
go install
```
Run the hdnprxy as a local proxy
```
export LOCAL_CERT="./certs/combined-certs.pem"
export LOCAL_KEY="./certs/key.pem"

cd ./build/local
hdnprxy --config config
```

### Configure your system to use the local hdnprxy
Go to Settings -> Network -> Network Proxy and select Manual in the Network Proxy message box. Set the HTTPS proxy to 127.0.0.1 and port 20443.
Note: it should _not_ need any special permissions. It just needs access to a non restricted port and the external network.


## Setting up your own remote hdnprxy
This section just gives an overview of setup options and is intended for users with technical experience

The hdnprxy is just a way to form a tunnel from a local hdnprxy to a remote hdnprxy. To provide internet access, a full proxt is also needed. You can use any proxy you like, but for our purposes we have provided one which can be downloaded from Github https://github.com/299m/httpprxy.git

### Download and build the two projects

git clone https://github.com/299m/httpprxy.git
git clone https://github.com/299m/hdnprxy.git
cd httpprxy
go install
cd ../hdnprxy
go install



### Configuration
Most configuration can be left as the default - we only cover here, those that may be useful

#### content.json
Either create your own web site or simply leave these instructions in place

```
"Homefile": "vue-cfg/http/dist/index.html", - this should be the index.html file.
"Basedir":  "vue-cfg/http/dist" - this shoul be the html base directory from which all of this sites files are served
```

#### engine.json
You can enable/disable various debug logs

#### general.json
```
"ProxyParam": "exo", - the parameter name of a HTTP POST parameter that must be sent to initiate a proxy session (the value of this parameter must match the proxy name in proxies.json)
"ProxyRoute": "/aa912", - the URL to request a proxy connection
"AllowedCACerts": ["./certs/ca-cert.pem"] - these should be dynamically added to the pool of valid CA certs used for the next connection. You can normally leave this empty.
```

#### proxies.json
This is where you define the proxies and their targets
```
"Proxies": {            - define a set of proxies
    "ab925af": {        - the key to activate this proxy (must be passed as a POST param named in general.json->ProxyParam, in this case the name is exo )
        "Proxyendpoint": "https://$PROXY_ADDRESS:$PROXY_PORT", - the end point, such as a full porxy server
        "Type": "net"   - the type of this proxy, currently only 'net' is supported
    }
}
```


#### tls.json
Set the certificate chain to present to the client hdnprxy on connection
```
"Cert": "$REMOTE_CERT",  - the certificate bundle this server will present
"Key":  "$REMOTE_KEY",   - the key for the servers own certificate
"Port": "443",           - it's recommended to leave as 443
"IsHttps": true          - leave this.
```



## Setting up your own httpprxy
If you want to use the hdnprxy for general internet access, then you need to point the remote hdnprxy at an HTTP proxy (such as httpprxy).

#### tls.json
Set the certificate chain to present to the client hdnprxy on connection


"Cert": "$PROXY_CERT",  - the certificate bundle this server will present
"Key":  "$PROXY_KEY",   - the key for the servers own certificate
"Port": "443",           - it's recommended to leave as 443
"IsHttps": true          - leave this.

#### filter.json
Set the certificate chain to present to the client hdnprxy on connection

"Whitelist": [],       - if you want to restrict access to certain sites only, the pattern is a regular expression, e.g. .*[\.]google.com
"Blacklist": [],       - if you wish to block certain sites, but allow access to everything else - useful for blocking tracking sites
"WhitelistPorts": [80, 443] - restricted ports. It's recommended to  leave this as is.

## Problems
This is a WIP, but if you do have issues or want to report a problem, drop an email to tech-team@299m.io, and we'll do what we can to help.

