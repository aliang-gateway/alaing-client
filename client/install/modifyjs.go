package install

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"nursor.org/nursorgate/common/logger"
)

var EXTENSION_DIR string

func AddHttp2ProxyForJsFile(fromJs, toJs string) error {
	var targetFuncRegex = regexp.MustCompile(`function\s+(\w+)\s*\(\s*(\w+)\s*,\s*(\w+)\s*\)\s*\{[^}]*?let\s*(\w*)\,(\w*)\;const[^}]*(\w)\=new\s+Promise\s*\(\s*\(\s*\(\s*\w+\s*,\s*\w+\s*\)\s*=>\s*\{\w+[^}]+\b\}\)\)\,(\w+)\=(\w+)\.connect\((\w+)\,(\w+)\);function (\w)\(\).*\)\,\w\(\)\}function \w\(\w\)\{null\=\=\w\|\|\w\((\w)\(.*off\(\Sconnect\S\,\w\).\w.off\(\Serror\S\,(\w).*\,onExitState\(\)\{(\w)\(\)\}\}\}`)
	jsfile, err := os.Open(fromJs)
	if err != nil {
		logger.Error(err)
		return err
	}
	jsContent, err := io.ReadAll(jsfile)
	if err != nil {
		logger.Error(err)
		return err
	}
	matches := targetFuncRegex.FindAllStringSubmatch(string(jsContent), -1)
	if len(matches) == 0 {
		logger.Error("NO HTTP2 PROXY SYMBOL FOUND")
		return fmt.Errorf("NO HTTP2 PROXY SYMBOL FOUND")
	}
	_W, _e, _A, _t, _r, _n, _s, _G, _i, R, _o, _a := matches[0][1], matches[0][2], matches[0][3], matches[0][4],
		matches[0][5], matches[0][6], matches[0][7], matches[0][8], matches[0][9+2], matches[0][12], matches[0][13],
		matches[0][14]

	rFunc, _ := GenerateFinalFunction(_W, _e, _A, _t, _r, _i, _n, _G, _s, _o, _a, R)
	rFunc = strings.ReplaceAll(rFunc, "\n", "")
	rFunc = strings.ReplaceAll(rFunc, "\r", "")

	finalMainjs := targetFuncRegex.ReplaceAllString(string(jsContent), rFunc)
	finalMainjs = strings.ReplaceAll(finalMainjs, "__TargetHost__", "${targetHost}")

	newJsfile, err := os.Create(toJs)
	if err != nil {
		logger.Error(err)
		return err
	}
	newJsfile.Write([]byte(finalMainjs))
	newJsfile.Close()
	jsfile.Close()
	return nil
}

func AddProxyForTransport(fromJs, toJS string) error {
	jsfile, err := os.Open(fromJs)
	if err != nil {
		logger.Error(err)
		return err
	}
	jsContent, err := io.ReadAll(jsfile)
	if err != nil {
		logger.Error(err)
		return err
	}
	oldFunc, newFunc, err := ExtractVariables(string(jsContent))
	if err == nil {
		resultFile := strings.ReplaceAll(string(jsContent), oldFunc, newFunc)
		err := os.Remove(toJS)
		if err != nil {
			logger.Error(err)
		}
		newJsfile, err := os.Create(toJS)
		if err != nil {
			logger.Error(err)
			return err
		}
		_, err = newJsfile.Write([]byte(resultFile))
		if err != nil {
			logger.Error(err)
			return err
		}
		newJsfile.Close()
		jsfile.Close()
		return nil
	}
	return nil
}

func SetExtensionPath(extensionPath string) {
	// 设置扩展路径
	EXTENSION_DIR = extensionPath
}

func BackCoreJSFile(nursorDir, extensionDir string) error {
	// 备份原始的 core.js 文件
	retriveJsFile := filepath.Join(extensionDir, "cursor-retrieval/dist/main.js")
	localJsFile := filepath.Join(extensionDir, "cursor-always-local/dist/main.js")
	extensionJsFile := filepath.Join(extensionDir, "cursor-shadow-workspace/dist/extension.js")
	deeplinkJsFile := filepath.Join(extensionDir, "cursor-deeplink/dist/main.js")
	workbenchJsPath := filepath.ToSlash(filepath.Join(filepath.Dir(extensionDir), "out", "vs", "workbench", "workbench.desktop.main.js"))

	// 创建备份目录
	backupDir := filepath.Join(nursorDir, "core")
	err := os.MkdirAll(backupDir, 0755)
	if err != nil {
		logger.Error(err)
		return err
	}
	// 备份文件
	if _, err := os.Stat(filepath.Join(backupDir, "main.js")); os.IsNotExist(err) {
		retriveJsContent, err := os.Open(retriveJsFile)
		if err != nil {
			logger.Error(err)
			return err
		}
		retriveJsContentByte, err := io.ReadAll(retriveJsContent)
		if err != nil {
			logger.Error(err)
			return err
		}
		err = os.WriteFile(filepath.Join(backupDir, "main.js"), retriveJsContentByte, 0644)
		if err != nil {
			logger.Error(err)
			return err
		}
	}
	if _, err := os.Stat(filepath.Join(backupDir, "local.js")); os.IsNotExist(err) {
		localJsContent, err := os.Open(localJsFile)
		if err != nil {
			logger.Error(err)
			return err
		}
		localJsContentByte, err := io.ReadAll(localJsContent)
		if err != nil {
			logger.Error(err)
			return err
		}
		err = os.WriteFile(filepath.Join(backupDir, "local.js"), localJsContentByte, 0644)
		if err != nil {
			logger.Error(err)
			return err
		}
	}
	if _, err := os.Stat(filepath.Join(backupDir, "extension.js")); os.IsNotExist(err) {
		extensionJsContent, err := os.Open(extensionJsFile)
		if err != nil {
			logger.Error(err)
			return err
		}
		extensionJsContentByte, err := io.ReadAll(extensionJsContent)
		if err != nil {
			logger.Error(err)
			return err
		}
		err = os.WriteFile(filepath.Join(backupDir, "extension.js"), extensionJsContentByte, 0644)
		if err != nil {
			logger.Error(err)
			return err
		}
	}
	if _, err := os.Stat(filepath.Join(backupDir, "core.js")); os.IsNotExist(err) {
		workbenchJsContent, err := os.Open(workbenchJsPath)
		if err != nil {
			logger.Error(err)
			return err
		}
		workbenchJsContentByte, err := io.ReadAll(workbenchJsContent)
		if err != nil {
			logger.Error(err)
			return err
		}
		err = os.WriteFile(filepath.Join(backupDir, "core.js"), workbenchJsContentByte, 0644)
		if err != nil {
			logger.Error(err)
			return err
		}
	}
	if _, err := os.Stat(filepath.Join(backupDir, "deeplink.js")); os.IsNotExist(err) {
		deeplinkJsContent, err := os.Open(deeplinkJsFile)
		if err != nil {
			logger.Error(err)
			return err
		}
		deeplinkJsContentByte, err := io.ReadAll(deeplinkJsContent)
		if err != nil {
			logger.Error(err)
			return err
		}
		err = os.WriteFile(filepath.Join(backupDir, "deeplink.js"), deeplinkJsContentByte, 0644)
		if err != nil {
			logger.Error(err)
			return err
		}
	}
	return nil
}

func ModifyJSFile(nursorDir, extensionDir string) error {
	// 修改 core.js 文件
	retriveJsFile := filepath.Join(extensionDir, "cursor-retrieval/dist/main.js")
	localJsFile := filepath.Join(extensionDir, "cursor-always-local/dist/main.js")
	extensionJsFile := filepath.Join(extensionDir, "cursor-shadow-workspace/dist/extension.js")
	deeplinkJsFile := filepath.Join(extensionDir, "cursor-deeplink/dist/main.js")
	// 0.5.0版本多了deep-link
	workbenchJsPath := filepath.ToSlash(filepath.Join(filepath.Dir(extensionDir), "out", "vs", "workbench", "workbench.desktop.main.js"))

	if err := AddHttp2ProxyForJsFile(filepath.Join(nursorDir, "core", "main.js"), retriveJsFile); err != nil {
		logger.Error("failure to add http2 core proxy for main js", err)
		// return err
	}
	if err := AddHttp2ProxyForJsFile(filepath.Join(nursorDir, "core", "local.js"), localJsFile); err != nil {
		logger.Error("failure to add http2 core proxy for local.js", err)
		// return err
	}
	if err := AddHttp2ProxyForJsFile(filepath.Join(nursorDir, "core", "extension.js"), extensionJsFile); err != nil {
		// return err
		logger.Error("failure to add http2 core proxy for local.js", err)
	}
	if err := AddHttp2ProxyForJsFile(filepath.Join(nursorDir, "core", "deeplink.js"), deeplinkJsFile); err != nil {
		// return err
		logger.Error("failure to add http2 core proxy for deeplink.js", err)
	}

	// 添加proxyfortransport
	if err := AddProxyForTransport(retriveJsFile, retriveJsFile); err != nil {
		// return err
		logger.Error("failure to add proxy for transport", err)
	}
	if err := AddProxyForTransport(localJsFile, localJsFile); err != nil {
		// return err
		logger.Error("failure to add proxy for transport for lcoaljs", err)
	}
	if err := AddProxyForTransport(extensionJsFile, extensionJsFile); err != nil {
		// return err
		logger.Error("failure to add proxy for transport for extensionjs", err)
	}
	if err := AddProxyForTransport(deeplinkJsFile, deeplinkJsFile); err != nil {
		// return err
		logger.Error("failure to add roxy for transport for deeplinkjs", err)
	}
	if err := ReplaceSentryJs(workbenchJsPath); err != nil {
		// return err
		logger.Error("failure to replace sentry js", err)
	}
	return nil

}

func IsJsModified(extensionDir string) bool {
	// const nursorMark=0;
	retriveJsFile := filepath.Join(extensionDir, "cursor-retrieval/dist/main.js")
	localJsFile := filepath.Join(extensionDir, "cursor-always-local/dist/main.js")
	extensionJsFile := filepath.Join(extensionDir, "cursor-shadow-workspace/dist/extension.js")
	deeplinkJsFile := filepath.Join(extensionDir, "cursor-deeplink/dist/main.js")
	workbenchJsPath := filepath.ToSlash(filepath.Join(filepath.Dir(extensionDir), "out", "vs", "workbench", "workbench.desktop.main.js"))
	if !IsMarkExist(retriveJsFile) {
		return false
	}
	if !IsMarkExist(localJsFile) {
		return false
	}
	if !IsMarkExist(extensionJsFile) {
		return false
	}
	if !IsMarkExist(deeplinkJsFile) {
		return false
	}
	if !IsMarkExist(workbenchJsPath) {
		return false
	}
	return true
}

func IsMarkExist(jsFile string) bool {
	jsContent, err := os.ReadFile(jsFile)
	if err != nil {
		logger.Error(err)
		return false
	}
	markRegex := regexp.MustCompile(`const nursorMark=0;`)
	return markRegex.MatchString(string(jsContent))
}
