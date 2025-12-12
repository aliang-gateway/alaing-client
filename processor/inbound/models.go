package inbound

// InboundResponse API response from inbound endpoint
type InboundResponse struct {
	Code int            `json:"code"`
	Msg  string         `json:"msg"`
	Data []InboundInfo  `json:"data"`
}

// InboundInfo represents inbound proxy information
type InboundInfo struct {
	InboundType string      `json:"inbound_type"`  // "vless" or "shadowsocks"
	Tag         string      `json:"tag"`           // e.g., "vless-35001"
	Config      interface{} `json:"config"`        // specific config object
}

// VLESSInboundConfig VLESS protocol configuration
type VLESSInboundConfig struct {
	TLSEnabled     bool     `json:"tls_enabled"`
	RealityEnabled bool     `json:"reality_enabled"`
	PublicKey      string   `json:"public_key"`
	ShortIDs       []string `json:"short_ids"`
	TLSServerName  string   `json:"tls_server_name"`
	ServerHost     string   `json:"server_host"`
	ServerPort     int      `json:"server_port"`
	VlessUUID      string   `json:"vless_uuid"`
	VlessFlow      string   `json:"vless_flow"`
}

// SSInboundConfig Shadowsocks protocol configuration
type SSInboundConfig struct {
	TLSEnabled     bool     `json:"tls_enabled"`
	RealityEnabled bool     `json:"reality_enabled"`
	SSPassword     string   `json:"ss_password"`
	Method         string   `json:"method"`
	ProxyUser      string   `json:"proxy_username"`
	PublicKey      string   `json:"public_key"`
	ShortIDs       []string `json:"short_ids"`
	ServerHost     string   `json:"server_host"`
	ServerPort     int      `json:"server_port"`
}
