package main

/*
#include <stdlib.h>
#include <stdbool.h>
*/
import "C"

import (
	"encoding/json"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/common/model"
	"nursor.org/nursorgate/inbound/http"
	"nursor.org/nursorgate/outbound"
	user "nursor.org/nursorgate/processor/auth"
	"nursor.org/nursorgate/processor/cert/client"
	proxyConfig "nursor.org/nursorgate/processor/config"
	"nursor.org/nursorgate/processor/http2"
	"nursor.org/nursorgate/runner"
	"nursor.org/nursorgate/runner/utils"
)

//export startClient
func startClient() {
	// 初始化允许代理域名
	model.NewAllowProxyDomain()
	http.StartMitmHttp()
}

//export setOutboundToken
func setOutboundToken(token *C.char) {
	outbound.SetOutboundToken(C.GoString(token))
}

//export setServerHost
func setServerHost(host *C.char) {
	utils.SetServerHost(C.GoString(host))
}

//export exportCaCertToFile
func exportCaCertToFile(certPath *C.char) {
	err := client.ExportRootCaCertToFile(C.GoString(certPath))
	if err != nil {
		logger.Error(err.Error())
		return
	}
}

//export getToCursorDomain
func getToCursorDomain() *C.char {
	jsonStr, _ := json.Marshal(model.NewAllowProxyDomain())
	return C.CString(string(jsonStr))
}

//export runGate
func runGate(innerToken *C.char) *C.char {
	innerTokenStr := C.GoString(innerToken)
	user.SetInnerToken(innerTokenStr)
	logger.SetUserInfo(innerTokenStr)
	model.NewAllowProxyDomain()
	utils.SetServerHost("api2.nursor.org:12235")
	go runner.Start()
	res := <-runner.RunStatusChan
	logger.Info(res)
	resStr, _ := json.Marshal(res)
	return C.CString(string(resStr))
}

//export setUserInfo
func setUserInfo(innerToken *C.char, username *C.char, password *C.char, userUUID *C.char) {
	innerTokenStr := C.GoString(innerToken)
	usernameStr := C.GoString(username)
	passwordStr := C.GoString(password)
	userUUIDStr := C.GoString(userUUID)
	user.SetUsername(usernameStr)
	user.SetPassword(passwordStr)
	user.SetInnerToken(innerTokenStr)
	user.SetUserUUID(userUUIDStr)
	logger.SetUserInfo(innerTokenStr)
}

//export setLogWatchMode
func setLogWatchMode(enableWatch *C.bool, level *C.int) {
	watchMode := *enableWatch != C.bool(false)
	http2.IsWatcherAllowed = watchMode
	logLevel := int(*level)
	logger.SetHttpLogLevel(logger.LogLevel(logLevel))
	logger.SetLogLevel(logger.LogLevel(logLevel))
}

//export setCursorGateMode
func setCursorGateMode(enableCursorGate *C.bool) {
	cursorMode := *enableCursorGate != C.bool(false)
	http2.IsCursorProxyEnabled = cursorMode
}

//export stopGate
func stopGate() {
	runner.Stop()
}

//export setVLESSProxy
func setVLESSProxy(server *C.char, uuid *C.char, sni *C.char, publicKey *C.char, isDefault *C.bool, isDoorProxy *C.bool) {
	cfg := &proxyConfig.VLESSConfig{
		Server:         C.GoString(server),
		UUID:           C.GoString(uuid),
		SNI:            C.GoString(sni),
		PublicKey:      C.GoString(publicKey),
		RealityEnabled: len(C.GoString(publicKey)) > 0,
		TLSEnabled:     len(C.GoString(sni)) > 0,
	}

	proxyCfg := &proxyConfig.ProxyConfig{
		Type:        "vless",
		VLESS:       cfg,
		IsDefault:   *isDefault != C.bool(false),
		IsDoorProxy: *isDoorProxy != C.bool(false),
	}

	if err := proxyConfig.SetProxyConfig(proxyCfg); err != nil {
		logger.Error(err.Error())
	}
}

//export setShadowsocksProxy
func setShadowsocksProxy(server *C.char, method *C.char, password *C.char, obfsMode *C.char, obfsHost *C.char, isDefault *C.bool, isDoorProxy *C.bool) {
	cfg := &proxyConfig.ShadowsocksConfig{
		Server:   C.GoString(server),
		Method:   C.GoString(method),
		Password: C.GoString(password),
		ObfsMode: C.GoString(obfsMode),
		ObfsHost: C.GoString(obfsHost),
	}

	proxyCfg := &proxyConfig.ProxyConfig{
		Type:        "shadowsocks",
		Shadowsocks: cfg,
		IsDefault:   *isDefault != C.bool(false),
		IsDoorProxy: *isDoorProxy != C.bool(false),
	}

	if err := proxyConfig.SetProxyConfig(proxyCfg); err != nil {
		logger.Error(err.Error())
	}
}

//export registerProxy
func registerProxy(name *C.char, proxyType *C.char, server *C.char, uuid *C.char, sni *C.char, publicKey *C.char) {
	nameStr := C.GoString(name)
	typeStr := C.GoString(proxyType)

	var cfg *proxyConfig.ProxyConfig

	switch typeStr {
	case "vless":
		cfg = &proxyConfig.ProxyConfig{
			Type: "vless",
			VLESS: &proxyConfig.VLESSConfig{
				Server:         C.GoString(server),
				UUID:           C.GoString(uuid),
				SNI:            C.GoString(sni),
				PublicKey:      C.GoString(publicKey),
				RealityEnabled: len(C.GoString(publicKey)) > 0,
				TLSEnabled:     len(C.GoString(sni)) > 0,
			},
		}
	default:
		logger.Error("Unsupported proxy type: " + typeStr)
		return
	}

	if err := proxyRegistry.GetRegistry().RegisterFromConfig(nameStr, cfg); err != nil {
		logger.Error(err.Error())
	}
}

//export switchProxy
func switchProxy(name *C.char) {
	nameStr := C.GoString(name)
	if err := proxyRegistry.SetDefault(nameStr); err != nil {
		logger.Error(err.Error())
	}
}

//export listProxies
func listProxies() *C.char {
	info := proxyRegistry.GetRegistry().ListWithInfo()
	jsonStr, _ := json.Marshal(info)
	return C.CString(string(jsonStr))
}

func main() {
	panic("test")
}
