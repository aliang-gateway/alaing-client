package install

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"nursor.org/nursorgate/common/logger"
)

func ReplaceSentryJs(fromJs string) error {
	if IsMarkExist(fromJs) {
		return nil
	}
	SentryEndpointPattern := regexp.MustCompile(`(https://\w*\@metrics.cursor.sh/\d*?)\",`)
	mysentryEndporint := "https://73030158a161cc39c1fad0d39450e4db@sentry.nursor.org/8"
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
	matches := SentryEndpointPattern.FindAllStringSubmatch(string(jsContent), -1)
	if len(matches) > 0 && len(matches[0]) > 1 {
		newContent := strings.ReplaceAll(string(jsContent), matches[0][1], mysentryEndporint)
		newJsfile, err := os.Create(fromJs)
		if err != nil {
			logger.Error(err)
			return err
		}
		newJsfile.Write([]byte(newContent))
		newJsfile.Close()
		jsfile.Close()
	}

	return nil
}

func ReplaceBackendURL(fromJs string) error {
	backendURLPattern := regexp.MustCompile(`getBackendUrl\(\)\W*\{\W*(return this\.\w*\.applicationUserPersistentStorage\.cursorCreds\.backendUrl)`)
	jsContent := `                const vscode = require('vscode');
                const config = vscode.workspace.getConfiguration('nursorPremiumChannel');
                const isProxyEnabled = config.get('isPoweredByNursor', false);
                if (isProxyEnabled){
                return "cursor.sh.nursor.org"

                }else{
                return __BackendURL__
                }`
	jsfile, err := os.Open(fromJs)
	if err != nil {
		logger.Error(err)
		return err
	}
	jsContentByte, err := io.ReadAll(jsfile)
	if err != nil {
		logger.Error(err)
		return err
	}
	matches := backendURLPattern.FindAllStringSubmatch(string(jsContentByte), -1)
	if len(matches) > 0 && len(matches[0]) > 1 {
		jsContent = strings.ReplaceAll(jsContent, "__BackendURL__", matches[0][1])
		newContent := strings.ReplaceAll(string(jsContentByte), matches[0][1], jsContent)
		newJsfile, err := os.Create(fromJs)
		if err != nil {
			logger.Error(err)
			return err
		}
		newJsfile.Write([]byte(newContent))
		newJsfile.Close()
		jsfile.Close()
	}
	return nil
}

// login的方法
func FindLoginJsFunc(fromJs string) (string, error) {
	loginFuncPattern := regexp.MustCompile(`\w\.authId\W*this\.(\w*)\(\w*\.authId\)[\w|\W]*\w\.refreshToken\W*&&\W*\(this.(\w*)\(\w.accessToken\,\W*\w\.refreshToken\)\,\W*await\Wthis.refreshMembershipType\(\)\,\W*this\.(\w*)\(\)\,`)
	jsContent := `this.nursor_login = async (mockResponse) =>{if (mockResponse && typeof mockResponse === "object") {const { authId, accessToken, refreshToken } = mockResponse;if (authId) {this.%s(authId);}if (accessToken && refreshToken) {this.%s(accessToken, refreshToken);await this.refreshMembershipType();this.%s();}console.log("Login completed with mock data:", mockResponse);} else {console.warn("Invalid mock response provided:", mockResponse);}}, window.nursor=this,
	 window.nursorwsf= function(){try{const ws=new WebSocket('ws://127.0.0.1:56433/ws');ws.onmessage=(e)=>{try{window.nursor?.nursor_login(JSON.parse(e.data))}catch(e){}};ws.onclose=()=>setTimeout(connect,1000)}catch(e){setTimeout(connect,1000)}},setTimeout(window.nursorwsf,1000*5),
	`
	jsfile, err := os.Open(fromJs)
	if err != nil {
		logger.Error()
		return "", err
	}
	jsContentBye, err := io.ReadAll(jsfile)
	if err != nil {
		return "", err
	}
	matches := loginFuncPattern.FindAllStringSubmatch(string(jsContentBye), -1)
	if len(matches) > 0 && len(matches[0]) > 0 {
		result := fmt.Sprintf(jsContent, matches[0][1], matches[0][2], matches[0][3])
		fmt.Print(result)
		result = strings.ReplaceAll(result, "\n", "")
		return result, nil
	}
	return "", errors.New("not found")
}

func ReplaceLoginAncher(fromJs string) error {
	ancherPattern := regexp.MustCompile(`(this\.onDidChangeSnippetLearningEligibility\=this\..{1,5}\.event\,)this`)

	jsfile, err := os.Open(fromJs)
	if err != nil {
		logger.Error()
		return err
	}
	jsContentBye, err := io.ReadAll(jsfile)
	if err != nil {
		return err
	}
	matches := ancherPattern.FindAllStringSubmatch(string(jsContentBye), -1)
	if len(matches) == 0 {
		logger.Error("error in replace login ancher, tranditional login unsupported")
		return nil
	}
	targetStr, err := FindLoginJsFunc(fromJs)
	if err != nil {
		logger.Error("not found login js func var name")
		return nil
	}
	finalTargetStr := strings.ReplaceAll(targetStr, "\n", "")
	finalTargetStr = strings.ReplaceAll(finalTargetStr, "\t", "")
	finalTargetStrAndWindow := matches[0][1] + "window.nursor=this," + finalTargetStr
	result := strings.ReplaceAll(string(jsContentBye), matches[0][1], finalTargetStrAndWindow)
	newJsfile, err := os.Create(fromJs)
	if err != nil {
		logger.Error(err)
		return err
	}
	newJsfile.Write([]byte(result))
	newJsfile.Close()
	jsfile.Close()

	return nil
}

func GetSqliteSymble(fromJs string) (string, error) {
	sqliteSymblePatter := regexp.MustCompile(`this\.\w{1,5}.get\(\"cursorAuth/refreshToken\"\,`)
	jsfile, err := os.Open(fromJs)
	if err != nil {
		logger.Error()
		return "", err
	}
	jsContentBye, err := io.ReadAll(jsfile)
	if err != nil {
		return "", err
	}
	matches := sqliteSymblePatter.FindAllStringSubmatch(string(jsContentBye), -1)
	if len(matches) > 0 && len(matches[0]) > 0 {
		return matches[0][0], nil
	}
	return "", errors.New("not found sqlite symbel")
}
