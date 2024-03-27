
<template>
<div class="card" >
    <Fieldset legend="Demo hdnprxy">
        <p class="m-2">
            The main use for the hdnprxy is to form a tunnel between a local and a remote hdnprxy server. All the code for it is open source and available on Github at https://github.com/299m/hdnprxy. It is released under a BSD style license.
        </p>
        <p class="m-2">
            This server is a demo setup of a remote hdnprxy. 
        </p>
    </Fieldset>
</div>

<div class="card">
<Fieldset legend="How to connect">
    <p class="m-2">
        The hdnprxy is designed to form a tunnel from a local hdnprxy to a remote hdnprxy. To connect to this demo remote hdnprxy, you need to setup the local hdnprxy and configur it to tunnel to the hdnprxy remote server (forming a TLS/TCP tunnel between the local hdnprxy and the remote hdnprxy). See instructions below.
        <Divider />
        For this demo, outgoing connections are limited to a few sites (Al Jazeera and Google search) and port 443 only. See below for how to set up your own server.
        
    </p>
    <p class="m-2">
        Once you have a local hdnpry running, configure you're system settings to proxy internet traffic through the local hdnprxy. On Ubuntu desktop go to Settings -> Network -> Network Proxy and select Manual in the Network Proxy message box. Set the HTTPS proxy to 127.0.0.1 and port 20443.
    </p>
</Fieldset>
</div>

<div class="card">
<Fieldset legend="Setup a local tunnel to the hdnprxy">
    <p class="m-2">You can either download the package from here or build it yourself</p>
    <TabView>
        <TabPanel header="Download package">
            <p class="m-0">
                (currenctly only Linux Ubuntu packages are available).
                <Divider />
                Download the zip file from <a href="/downloads/local.zip">here</a>.
                <Divider />
                Move the downloaded zip file to your home directory (or copy it from Downloads to your home directory).
                <Divider />
                Unzip the local.zip and run the script
                <pre class="language-markup" tabindex="-1">
                    <code class="language-markup">
cd ~/
unzip local.zip
cd local
./run-local.sh
                    </code>
                </pre>
                <Divider />

                Go to Settings -> Network -> Network Proxy and select Manual in the Network Proxy message box. Set the HTTPS proxy to 127.0.0.1 and port 20443.
                <Divider/> 
                Note: it should _not_ need any special permissions. It just needs access to a non restricted port and the external network.
            </p>
        </TabPanel>
        <TabPanel header="Build it from source">
            <p class="m-0">
                You must have Go installed (see <a href="https://go.dev/doc/install">install Go</a>).
                <Divider />
                Clone the repo to your development folder
                <pre class="language-markup" tabindex="-1">
                    <code class="language-markup">
git clone https://github.com/299m/hdnprxy.git
                    </code>
                </pre>
                <Divider />
                Build the hdnprxy
                <pre class="language-markup" tabindex="-1">
                    <code class="language-markup">
cd hdnprxy
go install
                    </code>
                </pre>
                <Divider />
                Run the hdnprxy as a local proxy
                <pre class="language-markup" tabindex="-1">
                    <code class="language-markup">
export LOCAL_CERT="./certs/combined-certs.pem"
export LOCAL_KEY="./certs/key.pem"

cd ./build/local
hdnprxy --config config
                    </code>
                </pre>
                <Divider />
                Go to Settings -> Network -> Network Proxy and select Manual in the Network Proxy message box. Set the HTTPS proxy to 127.0.0.1 and port 20443.
                <Divider/> 
                Note: it should _not_ need any special permissions. It just needs access to a non restricted port and the external network.

            </p>
        </TabPanel>
    </TabView>
</Fieldset>
</div>

<div class="card">
    <Fieldset legend="Setting up your own remote hdnprxy">
        <p class="m-2">This section just gives an overview of setup options and is intended for users with technical experience</p>
        <p class="m-2">The hdnprxy is just a way to form a tunnel from a local hdnprxy to a remote hdnprxy. To provide internet access, a full proxt is also needed. You can use any proxy you like, but for our purposes we have provided one which can be downloaded from Github https://github.com/299m/httpprxy.git</p>
        <Divider />
        <p class="m-2">
        Download and build the two projects
        <pre class="language-markup" tabindex="-1">
            <code class="language-markup">
git clone https://github.com/299m/httpprxy.git
git clone https://github.com/299m/hdnprxy.git
cd httpprxy
go install
cd ../hdnprxy
go install
            </code>
        </pre>
        </p>
        <h3 class="m-1">
            Configuration</h3>
            <p class="m-2"> Most configuration can be left as the default - we only cover here, those that may be useful
            </p>

        <h3 class="m-1">content.json</h3>        
        <p class="m-2">Either create your own web site or simply leave these instructions in place</p>
        <p></p>
        <pre class="language-markup" tabindex="-1">
            <code class="language-markup">
"Homefile": "vue-cfg/http/dist/index.html", - this should be the index.html file.
"Basedir":  "vue-cfg/http/dist" - this shoul be the html base directory from which all of this sites files are served 
            </code>
        </pre>
        <h3 class="m-1">engine.json</h3>   
        <p class="m-2">You can enable/disable various debug logs</p>
        <pre class="language-markup" tabindex="-1">
            <code class="language-markup">  
move along now, nothing to see here. 
            </code>
        </pre>
        <h3 class="m-1">general.json</h3>   

        <pre class="language-markup" tabindex="-1">
            <code class="language-markup">    
"ProxyParam": "exo", - the parameter name of a HTTP POST parameter that must be sent to initiate a proxy session (the value of this parameter must match the proxy name in proxies.json)
"ProxyRoute": "/aa912", - the URL to request a proxy connection
"AllowedCACerts": ["./certs/ca-cert.pem"] - these should be dynamically added to the pool of valid CA certs used for the next connection. You can normally leave this empty.
            </code>
        </pre>

        <h3 class="m-1">proxies.json</h3>   
        <p class="m-2">This is where you define the proxies and their targets</p>
        
        <pre class="language-markup" tabindex="-1">
            <code class="language-markup">    
"Proxies": {            - define a set of proxies 
    "ab925af": {        - the key to activate this proxy (must be passed as a POST param named in general.json->ProxyParam, in this case the name is exo )
        "Proxyendpoint": "https://$PROXY_ADDRESS:$PROXY_PORT", - the end point, such as a full porxy server
        "Type": "net"   - the type of this proxy, currently only 'net' is supported
    }
}
            
            </code>
        </pre>

        <h3 class="m-1">tls.json</h3>   
        <p class="m-2">Set the certificate chain to present to the client hdnprxy on connection</p>
        
        <pre class="language-markup" tabindex="-1">
            <code class="language-markup">    
"Cert": "$REMOTE_CERT",  - the certificate bundle this server will present
"Key":  "$REMOTE_KEY",   - the key for the servers own certificate
"Port": "443",           - it's recommended to leave as 443
"IsHttps": true          - leave this.
                      
            </code>
        </pre>

    </Fieldset>
</div>


<div class="card">
<Fieldset legend="Setting up your own httpprxy">
    <p class="m-2">
        If you want to use the hdnprxy for general internet access, then you need to point the remote hdnprxy at an HTTP proxy (such as httpprxy)
    </p>
    <h3 class="m-1">tls.json</h3>   
    <p class="m-2">Set the certificate chain to present to the client hdnprxy on connection</p>
    
    <pre class="language-markup" tabindex="-1">
        <code class="language-markup">    
"Cert": "$PROXY_CERT",  - the certificate bundle this server will present
"Key":  "$PROXY_KEY",   - the key for the servers own certificate
"Port": "443",           - it's recommended to leave as 443
"IsHttps": true          - leave this.
                  
        </code>
    </pre>
    <h3 class="m-1">filter.json</h3>   
    <p class="m-2">Set the certificate chain to present to the client hdnprxy on connection</p>
    
    <pre class="language-markup" tabindex="-1">
        <code class="language-markup">    
"Whitelist": [],       - if you want to restrict access to certain sites only, e.g. stackoverflow.com
"Blacklist": [],       - if you wish to block certain sites, but allow access to everything else
"WhitelistPorts": [80, 443] - restricted ports. It's recommended to  leave this as is.
                  
        </code>
    </pre>

</Fieldset>
</div>

<div class="card">
    <Fieldset legend="Problems?">
    <p class="m-2">This is a WIP, but if you do have issues or want to report a problem, drop an email to tech-team@299m.io, and we'll do what we can to help.
    </p>
    <p>
    </p>
    </Fieldset>
</div>
<Divider/>
<a href="https://www.digitalocean.com/?refcode=e8c6f2e583f5&utm_campaign=Referral_Invite&utm_medium=Referral_Program&utm_source=badge"><img src="https://web-platforms.sfo2.cdn.digitaloceanspaces.com/WWW/Badge%201.svg" alt="DigitalOcean Referral Badge" /></a>

</template>


<script setup>
import { ref, watchEffect } from 'vue';

const value = ref('off');
const options = ref(['Off', 'On']);
watchEffect(() => console.log(value.value));
</script>
