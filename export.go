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

func main() {
	panic("test")
}
