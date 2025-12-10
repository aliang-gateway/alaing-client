package metadata

import (
	"net"
	"net/netip"
	"time"
)

// BindingSource represents the source of domain name binding
type BindingSource string

const (
	BindingSourceSNI     BindingSource = "sni"
	BindingSourceHTTP    BindingSource = "http_host"
	BindingSourceDNS     BindingSource = "dns"
	BindingSourceCONNECT BindingSource = "connect"
)

// DNSInfo contains DNS-related binding information
type DNSInfo struct {
	// BindingSource indicates where the domain name comes from
	BindingSource BindingSource `json:"bindingSource"`
	// BindingTime is when the domain was bound to the IP
	BindingTime time.Time `json:"bindingTime"`
	// CacheTTL suggests how long to cache this binding
	CacheTTL time.Duration `json:"cacheTTL"`
	// ShouldCache indicates if this binding should be cached
	ShouldCache bool `json:"shouldCache"`
}

// Metadata contains metadata of transport protocol sessions.
type Metadata struct {
	Network  Network    `json:"network"`
	SrcIP    netip.Addr `json:"sourceIP"`
	MidIP    netip.Addr `json:"dialerIP"`
	DstIP    netip.Addr `json:"destinationIP"`
	SrcPort  uint16     `json:"sourcePort"`
	MidPort  uint16     `json:"dialerPort"`
	DstPort  uint16     `json:"destinationPort"`
	HostName string     `json:"hostName"`
	DNSInfo  *DNSInfo   `json:"dnsInfo"`
	Route    string     `json:"route"` // Final routing decision for this connection
}

func (m *Metadata) DestinationAddrPort() netip.AddrPort {
	return netip.AddrPortFrom(m.DstIP, m.DstPort)
}

func (m *Metadata) DestinationAddress() string {
	return m.DestinationAddrPort().String()
}

func (m *Metadata) SourceAddrPort() netip.AddrPort {
	return netip.AddrPortFrom(m.SrcIP, m.SrcPort)
}

func (m *Metadata) SourceAddress() string {
	return m.SourceAddrPort().String()
}

func (m *Metadata) Addr() net.Addr {
	return &Addr{metadata: m}
}

func (m *Metadata) TCPAddr() *net.TCPAddr {
	if m.Network != TCP || !m.DstIP.IsValid() {
		return nil
	}
	return net.TCPAddrFromAddrPort(m.DestinationAddrPort())
}

func (m *Metadata) UDPAddr() *net.UDPAddr {
	if m.Network != UDP || !m.DstIP.IsValid() {
		return nil
	}
	return net.UDPAddrFromAddrPort(m.DestinationAddrPort())
}

// Addr implements the net.Addr interface.
type Addr struct {
	metadata *Metadata
}

func (a *Addr) Metadata() *Metadata {
	return a.metadata
}

func (a *Addr) Network() string {
	return a.metadata.Network.String()
}

func (a *Addr) String() string {
	return a.metadata.DestinationAddress()
}
