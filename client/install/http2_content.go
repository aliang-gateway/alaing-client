package install

import "fmt"

func CreateProxyConnection() string {
	return `function createProxyConnection(targetUrl,proxyUrl){return new Promise((resolve,reject)=>{const targetURL=new URL(targetUrl);const proxyURL=new URL(proxyUrl);const options={host:proxyURL.hostname,port:proxyURL.port,method:'CONNECT',path:targetURL.hostname+':'+targetURL.port,};const req=http.request(options);req.end();req.on('connect',(res,socket)=>{if(res.statusCode===200){console.log('nursor连接成功');const tlsSocket=tls.connect({socket:socket,servername:targetURL.hostname,ca:[ca],rejectUnauthorized:false,minVersion:'TLSv1.2',maxVersion:'TLSv1.3',ALPNProtocols:['h2','http/1.1']});tlsSocket.setTimeout(1000*60*5);tlsSocket.on('timeout',()=>{tlsSocket.destroy()});tlsSocket.on('secureConnect',()=>{console.log('Client TLS version:',tlsSocket.getProtocol())});resolve(tlsSocket)}else{socket.destroy();reject(new Error("nursor连接失败:" + res.statusMessage))}});req.on('error',reject)})}`
}

func CreateDirectConnection(G, e, A, i, o, s string) string {
	return fmt.Sprintf(`function connectDirectly(){`) +
		fmt.Sprintf("const conn=%s.connect(%s, %s);", G, e, A) +
		fmt.Sprintf(`conn.on("connect", %s);`, i) +
		fmt.Sprintf(`conn.on("error", %s);%s = conn;}`, o, s)
}

func CreateConnectByProxy2(G, e, A, s, i, o string) string {
	return fmt.Sprintf(`async function connectViaProxy2(url,host){try{const tlsSocket=await createProxyConnection(url,'http://127.0.0.1:56432');tlsSocket.setTimeout(5000*60*5);tlsSocket.on('timeout',()=>{tlsSocket.destroy()});const options={createConnection:()=>tlsSocket,servername:host,ca:[ca],rejectUnauthorized:false,ALPNProtocols:['h2','http/1.1'],minVersion:'TLSv1.2',maxVersion:'TLSv1.3',};const conn=%s.connect(url,options);%s=conn;conn.on("connect",%s);conn.on("error",%s);tlsSocket.on('error',(err)=>{%s(err)})}catch(error){console.error('请求失败:',error);throw error;}}`, G, s, i, o, o)
}

func CreateConnectByProxy(G, e, A, s, i, o string) string {
	return fmt.Sprintf(`function connectViaProxy() {const socket = net.connect(proxyPort, proxyHost, () => {socket.write(` + "`" +
		`CONNECT __TargetHost__:443 HTTP/1.1\r\nHost: __TargetHost__\r\n\r\n` + "`" + `);});` +
		`let response='';socket.on('data',(chunk)=>{response+=chunk.toString();if (response.includes('200 Connection Established')) {` +
		`socket.pause();socket.removeAllListeners('data');` +
		`const tlsSocket = tls.connect(
		{socket: socket,servername: targetHost,rejectUnauthorized: false, ALPNProtocols: ['h2', 'http/1.1'], ca: [ca,]},
		 () => {
                        console.log('TLS secureConnect, protocol:', tlsSocket.getProtocol());
                    });` +
		`tlsSocket.setTimeout(5000*60*5);tlsSocket.on('timeout',()=>{tlsSocket.destroy();});` +
		fmt.Sprintf(`delete %s.createConnection;`, A) +
		fmt.Sprintf(`const conn=%s.connect(%s,{`, G, e) +
		fmt.Sprintf(`createConnection:()=>tlsSocket,ALPNProtocols:['h2'],...%s});`, A) +
		fmt.Sprintf(`%s=conn;conn.on("connect", %s);conn.on("error", %s);`, s, i, o) +
		fmt.Sprintf(`tlsSocket.on('error',(err)=>{%s(err);});}else{%s(new Error('Proxy response:'+response));}});`, o, o) +
		fmt.Sprintf(`socket.on('error',(err)=>{%s(err);});  }`, o))
}

func GenerateFinalFunction(W, e, A, t, r, i, n, G, s, o, a, R string) (string, error) {
	// 函数模板（包含代理逻辑）
	finalFunc := fmt.Sprintf("function %s(%s, %s){let %s,%s;", W, e, A, t, r) +
		fmt.Sprintf("const %s=new Promise((resolve,reject)=>{%s=resolve;%s=reject;});", n, t, r) +
		`const nursorMark=0;` +
		`const vscode=require('vscode');` +
		`const tls=require('tls');const net=require('net');const http = require('http');const config=vscode.workspace.getConfiguration('nursorPremiumChannel');` +
		`const isProxyEnabled=config.get('isPoweredByNursor',false);const proxyUrl=config.get('channelAddress',"http://127.0.0.1:56432");` +
		`const proxyHost=new URL(proxyUrl).hostname;const proxyPort=parseInt(new URL(proxyUrl).port||56432);const targetHost=new URL(e).hostname;` +
		`const ca=` + "`" + `-----BEGIN CERTIFICATE-----
MIIFezCCA2OgAwIBAgIUQ9BIhbm9Ic82l6nMer43HRywNOkwDQYJKoZIhvcNAQEL
BQAwXTELMAkGA1UEBhMCQ04xETAPBgNVBAgMCFNoYW5naGFpMREwDwYDVQQHDAhT
aGFuZ2hhaTEPMA0GA1UECgwGTnVyc29yMRcwFQYDVQQDDA5OdXJzb3IgUm9vdCBD
QTAeFw0yNTAzMjIwNjAwMTNaFw0zNTAzMjAwNjAwMTNaMGoxCzAJBgNVBAYTAkNO
MREwDwYDVQQIDAhTaGFuZ2hhaTERMA8GA1UEBwwIU2hhbmdoYWkxDzANBgNVBAoM
Bk51cnNvcjEkMCIGA1UEAwwbTnVyc29yIE1JVE0gSW50ZXJtZWRpYXRlIENBMIIC
IjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAllDeS6PhxAHFTtKr3t9UvXoP
jj7eLA44n5F+S4lcO/LNLio0I4hZE/tYAK7I/FyNBrBzKYFIeoyNIp8W9SXJGDhC
ykReqswWkz3ur3562vkVPTj38nNdpSP5bStTWBJ822MkNJt2VhxkmX96fMH185EB
vGn6RaqKOJMCz5DXxbuBQbypYPMrm7T5lqOigiiqAeyf4xQk0se8qX8b+KtkZrBK
mFz9PtHYS010dQUhdNxtGmlockAQ4CoGdMJePd9t7vI92+Ml+fA1xIfPSb4GIiwE
I0Nfli5wXB19xNa2eG+Jhafy7CDhTeIo/xEpYbzrAnp/PnmpkLzpnC0rv61xhL4P
mJwazefnfYa+E+bLFRzR+22u6g5YuuikEAoB26SlAgwtcI3NSTlnrAVAoGopS/EK
0o4EiAZIkcgA6r/15xkudYM2O+WtnYRc6uqaAZ2+BZuNVMXmLGcamXo9WlVTC33X
Wqb2NjRLjqC3r5OnxhysbexEStVzSWHzhLGoeqWUKmYovymz0KXHvOsDZwqyQAJc
04qeUlg2NkODIHZXnsRitDa3MsXrn/vTtGUtCB4CijWbGezvcykYcr8MQY+6vBi4
4sIPiZM6+6JH7l/p1iMRDaI1qNnpxVMYNZrA3spwJPSasIaRtXL6rAzWhgg0DWM3
2eTrRoq69pD1kYAaTMMCAwEAAaMmMCQwEgYDVR0TAQH/BAgwBgEB/wIBADAOBgNV
HQ8BAf8EBAMCAYYwDQYJKoZIhvcNAQELBQADggIBADXiph86bjk8G0YPFAoRTrpz
y4uY36N4bUjK9LINtanjj9Hn0vHQetAY9Q/Ej0QeDE+CI4sqMs9RlHs47indmeRp
J8djq82dqFLG65ld94rKsh/dvpLWDwKEr1703iUIWcTt5jg8QioIM9MjiJY3XMoI
bpofKARomYyO3fAzvJelBn0fw9CzOQ0BE2KofvWOPiYiF8sNcomFoTjmp2ZbgT7u
+r4ShORiqkbs/jEuLfWKO70vs1kHlt89si9K8mKX4aNr2MnCjpwsOmxam+Z4Op35
k3o98HaLuM+MolfCV5EYEubt9MaNOq3rT9jkypy7oTirYxUPu9aQDPOp/H0kncIu
faTBaQsJLCaU5xALO/U7nQZHu19gjZlrX0qfGxk5vzb8yaDnustsfO16YRF3Yvuc
eMYrKpP3dr0h8XqmvbOO3uCprTHFfhsQrEd5CRM8gI4CMIj5nxL0Li7vhvN3bKnk
jMQelb2ap66YStUU4nyVr6+z99S4WXmf37UbvnzSIJN7BZjqIgEsXM4Ht3Cmvh8C
0si4GKry62tIy9+EQs860MyUozv7Cv1xPYPhQQpN/vlSqgjyGFiI7T0hl9o9HCRz
5IL75GVKUADepeutcn/HzvVpU+uHk11e/K7AzhmM0/f+Umpp4H+wdxdbr/X0AAHl
GjVxVTvgmxRJgAjHW4gd
-----END CERTIFICATE-----` + "`" + `;` +
		CreateProxyConnection() +
		CreateDirectConnection(G, e, A, i, o, s) +
		CreateConnectByProxy(G, e, A, s, i, o) +
		CreateConnectByProxy2(G, e, A, s, i, o) +
		fmt.Sprintf(`function %s(){if(%s!=null)%s(%s);%s();}`, i, t, t, s, a) +
		fmt.Sprintf(`function %s(e){if(%s!=null) %s(%s(e));%s();}`, o, r, r, R, a) +
		fmt.Sprintf(`function %s(){%s?.off("connect",%s);%s?.off("error", %s);}`, a, s, i, s, o) +
		fmt.Sprintf(`let %s;if(isProxyEnabled){connectViaProxy2(e,targetHost);}else{connectDirectly();}`, s) +
		fmt.Sprintf(`return {t:"connecting",conn:%s,abort(e){if (%s && !%s.destroyed) {`, n, s, s) +
		fmt.Sprintf(`%s.destroy(void 0, %s?.constants?.NGHTTP2_CANCEL);}`, s, G) +
		fmt.Sprintf(`if(%s!=null)%s(e);},onExitState(){%s();}};}`, r, r, a)

	return finalFunc, nil
}
