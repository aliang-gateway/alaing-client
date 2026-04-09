package tcp

import (
	"net/netip"
	"strings"

	M "aliang.one/nursorgate/inbound/tun/metadata"
)

const aliangLocalHTTPProxyPort uint16 = 56432

func shouldForceAliangRoute(metadata *M.Metadata) bool {
	if metadata == nil || metadata.DstPort != aliangLocalHTTPProxyPort {
		return false
	}

	if metadata.DstIP.IsValid() && metadata.DstIP.IsLoopback() {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(metadata.HostName))
	if host == "" {
		return false
	}
	if host == "localhost" {
		return true
	}

	if ip, err := netip.ParseAddr(host); err == nil && ip.IsLoopback() {
		return true
	}

	return false
}
