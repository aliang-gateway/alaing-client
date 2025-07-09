package test

import (
	"testing"

	"nursor.org/nursorgate/client/install"
)

func TestReplace(t *testing.T) {
	install.ReplaceSentryJs("C:\\Users\\Administrator\\AppData\\Local\\Programs\\cursor\\resources\\app\\out\\vs\\workbench\\workbench.desktop.main.js")
}

func TestReplaceBackendURL(t *testing.T) {
	install.ReplaceBackendURL("C:\\Users\\Administrator\\AppData\\Local\\Programs\\cursor\\resources\\app\\out\\vs\\workbench\\workbench.desktop.main.js")
}

func TestFindLogin(t *testing.T) {
	install.FindLoginJsFunc("C:\\Users\\Administrator\\AppData\\Local\\Programs\\cursor\\resources\\app\\out\\vs\\workbench\\workbench.desktop.main.js")
}

func TestReplaceLoginAncher(t *testing.T) {
	install.ReplaceLoginAncher("/Applications/Cursor.app/Contents/Resources/app/out/vs/workbench/workbench.desktop.main.js")
}

func TestBackupJS(t *testing.T) {
	install.BackCoreJSFile("C:\\Users\\Administrator\\.nursor", "C:\\Users\\Administrator\\AppData\\Local\\Programs\\cursor\\resources\\app\\extensions")
}
