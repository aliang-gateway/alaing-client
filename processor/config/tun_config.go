package config

var CursorAiGatewayHost string

func SetCursorAiGatewayHost(host string) {
	CursorAiGatewayHost = host
}

func GetCursorAiGatewayHost() string {
	return CursorAiGatewayHost
}
