package test

import (
	"testing"

	"nursor.org/nursorgate/client/install"
)

func TestReplaceExtensionTransport(t *testing.T) {
	install.AddProxyForTransport("/Applications/Cursor.app/Contents/Resources/app/extensions/cursor-deeplink/dist/main.js", "/Applications/Cursor.app/Contents/Resources/app/extensions/cursor-deeplink/dist/main.js")
}

func TestReplaceExtensionHttp2(t *testing.T) {
	install.AddHttp2ProxyForJsFile("/Applications/Cursor.app/Contents/Resources/app/extensions/cursor-deeplink/dist/main.js", "/Applications/Cursor.app/Contents/Resources/app/extensions/cursor-deeplink/dist/main.js")
}
