package test

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nursor.org/nursorgate/client/install"
	"nursor.org/nursorgate/common/logger"
	//tungvisor "nursor.org/nursorgate/client/server/utils/tun_gvisor"
	//"nursor.org/nursorgate/client/utils/tun_gvisor/devices/tun"
)

func TestCreateProxyJs(t *testing.T) {
	finaljs, _ := install.GenerateFinalFunction("W", "e", "A", "t", "r", "i", "n", "G", "s", "o", "a", "R")

	newJsfile, err := os.Create("conn_with_proxy.js")
	if err != nil {
		logger.Error(err)
		t.Fatal(err) // Add proper test failure handling
	}
	defer newJsfile.Close() // Better file handling
	finaljs = strings.ReplaceAll(finaljs, "\n", "")
	finaljs = strings.ReplaceAll(finaljs, "\t", "")
	finaljs = strings.ReplaceAll(finaljs, "__TargetHost__", "${targetHost}")

	_, err = newJsfile.Write([]byte(finaljs))
	if err != nil {
		t.Fatal(err)
	}
	t.Log("conn_with_proxy.js created successfully")
}

func TestCreateConnectForTransport(t *testing.T) {
	finaljs := install.CreateProxyForTransportContent()
	newJsfile, err := os.Create("proxy_for_transport.js")
	if err != nil {
		logger.Error(err)
		t.Fatal(err) // Add proper test failure handling
	}
	defer newJsfile.Close() // Better file handling
	_, err = newJsfile.Write([]byte(finaljs))
	if err != nil {
		t.Fatal(err)
	}
	t.Log("conn_with_proxy.js created successfully")
}

func TestFindTransportAndReplaceWithProxy(t *testing.T) {
	corejsPath := filepath.Join(os.Getenv("HOME"), ".nursor", "core")
	corejs := filepath.Join(corejsPath, "main.js")
	corefile, _ := os.Open(corejs)
	content, _ := io.ReadAll(corefile)

	_, finalTrJs, _ := install.ExtractVariables(string(content))
	newJsfile, err := os.Create("transport_with_proxy.js")
	if err != nil {
		logger.Error(err)
		t.Fatal(err) // Add proper test failure handling
	}
	defer newJsfile.Close() // Better file handling
	_, err = newJsfile.Write([]byte(finalTrJs))
	if err != nil {
		t.Fatal(err)
	}
	t.Log("conn_with_proxy.js created successfully")
}

func TestSaveTransportWhichHasProxy(t *testing.T) {
	fromJs := "/Users/liang/.nursor/core/main.js"
	// toJS := "/Users/liang/MyProgram/goprogram/nursor/nursorgate/client/test/withChangedProxy.js"
	jsfile, err := os.Open(fromJs)
	if err != nil {
		logger.Error(err)
		t.Fail()
	}
	jsContent, err := io.ReadAll(jsfile)
	if err != nil {
		logger.Error(err)
		t.Fail()
	}
	oldFunc, newFunc, err := install.ExtractVariables(string(jsContent))
	if err == nil {
		print(oldFunc, newFunc)

	}

}
