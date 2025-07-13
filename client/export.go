package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"encoding/json"

	"nursor.org/nursorgate/client/inbound/cert"
	"nursor.org/nursorgate/client/outbound"
	"nursor.org/nursorgate/client/server"
	"nursor.org/nursorgate/client/server/tun"
	"nursor.org/nursorgate/client/utils"
	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/common/model"
)

//export startClient
func startClient() {
	// 初始化允许代理域名
	model.NewAllowProxyDomain()
	server.StartMitmHttp()
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
	err := cert.ExportRootCaCertToFile(C.GoString(certPath))
	if err != nil {
		logger.Error(err.Error())
		return
	}
}

//export setSqlitePath
func setSqlitePath(sqlitePath *C.char) {
	utils.NewKVStore().SetDBPath(C.GoString(sqlitePath))
}

//export getDataFromSqlite
func getDataFromSqlite(key *C.char) *C.char {
	data, err := utils.NewKVStore().Read(C.GoString(key))
	if err != nil {
		return nil
	}
	return C.CString(data)
}

//export setDataToSqlite
func setDataToSqlite(key *C.char, value *C.char) {
	utils.NewKVStore().Set(C.GoString(key), C.GoString(value))
}

//export deleteDataFromSqlite
func deleteDataFromSqlite(key *C.char) {
	utils.NewKVStore().Delete(C.GoString(key))
}

//export closeSqlite
func closeSqlite() {
	utils.NewKVStore().Close()
}

//export getToCursorDomain
func getToCursorDomain() *C.char {
	jsonStr, _ := json.Marshal(model.NewAllowProxyDomain())
	return C.CString(string(jsonStr))
}

//export runGate
func runGate() *C.char {
	model.NewAllowProxyDomain()
	utils.SetServerHost("192.140.163.38:12235")
	go tun.Start()
	res := <-tun.RunStatusChan
	logger.Info(res)
	resStr, _ := json.Marshal(res)
	return C.CString(string(resStr))
}

//export stopGate
func stopGate() {
	tun.Stop()
}
