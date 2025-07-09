package install

import (
	"fmt"
	"regexp"
	"strings"

	"nursor.org/nursorgate/client/outbound/cert"
)

func CreateProxyForTransportContent() string {
	targetFunc := `const createProxyConnection=async(url,useHttp2)=>{const core_ca=` + "`__MYCERT__`" + `;const net=require('net');const tls=require('tls');const proxyHost='127.0.0.1';const proxyPort=56432;const targetHost=new URL(url).hostname;const targetPort=new URL(url).port||443;const socket=net.connect(proxyPort,proxyHost);socket.write(__REPLACE__1);return new Promise((resolve,reject)=>{let response='';socket.on('data',(data)=>{response+=data.toString();if(response.includes('200 Connection')){const tlsSocket=tls.connect({socket,servername:targetHost,ALPNProtocols:useHttp2?['h2','http/1.1']:['http/1.1'],minVersion:'TLSv1.2',ca:[core_ca],rejectUnauthorized:false});tlsSocket.setTimeout(1000*60*60*10);tlsSocket.on('timeout',()=>{console.log('TLS socket timeout, closing connection');tlsSocket.destroy()});tlsSocket.on('secureConnect',()=>{console.log('TLS connected, ALPN:',tlsSocket.alpnProtocol);if(useHttp2&&tlsSocket.alpnProtocol!=='h2'){reject(new Error('HTTP/2 not negotiated'))}else{resolve(tlsSocket)}})}});socket.on('error',reject)})};`
	targetFunc = strings.ReplaceAll(targetFunc, "__REPLACE__1", "`CONNECT __TargetHost__:443 HTTP/1.1\\r\\nHost: __TargetHost__\\r\\n\\r\\n`")
	targetFunc = strings.ReplaceAll(targetFunc, "__TargetHost__", "${targetHost}")
	targetFunc = strings.ReplaceAll(targetFunc, "__MYCERT__", string(cert.CaCert))
	return targetFunc
}

func NewTransportWithProxy(Ie, e, funcBody string) string {
	return fmt.Sprintf(`function %s(%s){%s;const vscode=require('vscode');const nursorMark=0;const config=vscode.workspace.getConfiguration('nursorPremiumChannel');const isProxyEnabled=config.get('isPoweredByNursor',false);if(isProxyEnabled){ let httpVersion=e.httpVersion;let isHttp2=httpVersion.includes('2');let nodeoptions=e.nodeOptions;let url=e.baseUrl;let createConnection=()=>createProxyConnection(url,isHttp2);if(isHttp2){console.log('Using HTTP/2');let newOptions={...e.nodeoptions,createConnection,}; %s.nodeOptions=newOptions;}else{const https=require('https');const agent=new https.Agent({keepAlive:true,createConnection,});%s.nodeOptions={...e.nodeOptions,agent,};console.log('Using HTTP/1.1');}} %s}`, Ie, e, CreateProxyForTransportContent(), e, e, funcBody)
}

func ExtractVariables(rawJSFileContent string) (string, string, error) {
	// 1. 匹配 createConnectTransport: () => Ie 中的函数名
	createTransportRe := regexp.MustCompile(`createConnectTransport\s*:\s*\(\s*\)\s*=>\s*(\w+)`)
	matches := createTransportRe.FindStringSubmatch(rawJSFileContent)
	if len(matches) < 2 {
		return "", "", fmt.Errorf("未能找到 createConnectTransport 的函数名")
	}
	funcName := matches[1] // 提取 Ie

	// 2. 匹配 function Ie(e) { ... } 的定义
	funcRe := regexp.MustCompile(fmt.Sprintf(`function\s+%s\s*\((.*?)\)\s*\{([\s\S]*?)\}`, regexp.QuoteMeta(funcName)))
	funcMatches := funcRe.FindStringSubmatch(rawJSFileContent)
	if len(funcMatches) < 3 {
		return "", "", fmt.Errorf("未能找到函数 %s 的定义", funcName)
	}
	params := funcMatches[1] // 函数参数，如 "e"
	body := funcMatches[2]   // 函数体，如 "return (0, ce.o)(te(e))"

	finalFunc := NewTransportWithProxy(funcName, params, body)
	oldFunc := string(funcMatches[0])
	return oldFunc, finalFunc, nil

}
