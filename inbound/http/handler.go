package http

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"nursor.org/nursorgate/common/logger"
	M "nursor.org/nursorgate/inbound/tun/metadata"
	"nursor.org/nursorgate/processor/tcp"
)

func HandleHttpConnection(conn net.Conn, req *http.Request) {
	defer conn.Close()

	log.Printf("Received non-CONNECT request: %s %s", req.Method, req.URL.String())

	// Extract metadata from HTTP request for transparent proxy handling
	metadata, err := extractMetadataFromHTTP(req, conn)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to extract metadata from HTTP request: %v", err))
		respWriter := NewCustomResponseWriter(conn)
		respWriter.WriteHeader(http.StatusBadGateway)
		respWriter.Write([]byte(fmt.Sprintf("Failed to process request: %v", err)))
		respWriter.Flush()
		return
	}

	logger.Debug(fmt.Sprintf("HTTP transparent proxy: hostname=%s, port=%d, srcIP=%s, dstIP=%s",
		metadata.HostName, metadata.DstPort, metadata.SrcIP, metadata.DstIP))

	// Import tcp handler for unified TCP processing
	tcpHandler := tcp.GetHandler()

	// Create context for the handler
	ctx := context.Background()

	// Delegate to processor/tcp for routing and relay
	// The handler will:
	// 1. Detect protocol (TLS on 443, HTTP on 80, direct for others)
	// 2. Route based on domain rules (cursor proxy, door proxy, or direct)
	// 3. Handle bidirectional relay with statistics
	logger.Debug(fmt.Sprintf("Routing HTTP request through TCP handler for %s:%d", metadata.HostName, metadata.DstPort))
	if err := tcpHandler.Handle(ctx, conn, metadata); err != nil {
		logger.Error(fmt.Sprintf("TCP handler failed for %s: %v", metadata.HostName, err))
		return
	}

	logger.Debug(fmt.Sprintf("HTTP connection closed: %s:%d", metadata.HostName, metadata.DstPort))
}

// extractMetadataFromHTTP extracts connection metadata from HTTP request
func extractMetadataFromHTTP(req *http.Request, conn net.Conn) (*M.Metadata, error) {
	metadata := &M.Metadata{
		Network: M.TCP,
	}

	// Extract host and port from HTTP request
	host := req.Header.Get("Host")
	if host == "" {
		host = req.URL.Host
	}

	if host == "" {
		return nil, fmt.Errorf("cannot determine host from HTTP request")
	}

	// Parse host:port
	var hostOnly string
	var port uint16 = 80 // Default HTTP port

	if strings.Contains(host, ":") {
		parts := strings.Split(host, ":")
		hostOnly = parts[0]
		portNum, err := strconv.ParseUint(parts[1], 10, 16)
		if err == nil {
			port = uint16(portNum)
		}
	} else {
		hostOnly = host
	}

	// Set hostname with HTTP binding source
	if hostOnly != "" {
		metadata.SetHostName(hostOnly, M.BindingSourceHTTP, 10*time.Minute)
	}
	metadata.DstPort = port

	// Try to parse host as IP
	if ip := net.ParseIP(hostOnly); ip != nil {
		if addr, err := convertNetIPToNetipAddr(ip); err == nil {
			metadata.DstIP = addr
		}
	} else {
		// Try to resolve hostname
		ips, err := net.LookupIP(hostOnly)
		if err != nil {
			logger.Debug(fmt.Sprintf("DNS resolution failed for %s: %v (will retry at tunnel time)", hostOnly, err))
		} else if len(ips) > 0 {
			if addr, err := convertNetIPToNetipAddr(ips[0]); err == nil {
				metadata.DstIP = addr
				logger.Debug(fmt.Sprintf("Resolved %s to %s", hostOnly, metadata.DstIP.String()))
			}
		}
	}

	// Extract source
	if remoteAddr, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
		if addr, err := convertNetIPToNetipAddr(remoteAddr.IP); err == nil {
			metadata.SrcIP = addr
		}
		metadata.SrcPort = uint16(remoteAddr.Port)
	}

	// Extract local address
	if localAddr, ok := conn.LocalAddr().(*net.TCPAddr); ok {
		if addr, err := convertNetIPToNetipAddr(localAddr.IP); err == nil {
			metadata.MidIP = addr
		}
		metadata.MidPort = uint16(localAddr.Port)
	}

	return metadata, nil
}

// convertNetIPToNetipAddr converts net.IP to netip.Addr
func convertNetIPToNetipAddr(ip net.IP) (netip.Addr, error) {
	if ip == nil {
		return netip.Addr{}, fmt.Errorf("nil IP")
	}
	return netip.ParseAddr(ip.String())
}
