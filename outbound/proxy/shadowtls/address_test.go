package shadowtls

import (
	"net"
	"net/netip"
	"testing"
	"time"
)

// TestEncodeAddressIPv4 tests IPv4 address encoding in SOCKS5 format
func TestEncodeAddressIPv4(t *testing.T) {
	tests := []struct {
		name      string
		hostName  string
		ip        netip.Addr
		port      uint16
		wantType  byte
		wantLen   int
		wantError bool
	}{
		{
			name:     "IPv4 address encoding",
			hostName: "",
			ip:       netip.MustParseAddr("192.168.1.1"),
			port:     80,
			wantType: AddrTypeIPv4,
			wantLen:  7, // 1 (type) + 4 (ip) + 2 (port)
		},
		{
			name:     "IPv4 with hostname (prefers hostname)",
			hostName: "example.com",
			ip:       netip.MustParseAddr("192.168.1.1"),
			port:     443,
			wantType: AddrTypeDomain,
			wantLen:  1 + 1 + 11 + 2, // 1 (type) + 1 (length) + 11 (domain) + 2 (port)
		},
		{
			name:     "IPv6 address encoding",
			hostName: "",
			ip:       netip.MustParseAddr("::1"),
			port:     8080,
			wantType: AddrTypeIPv6,
			wantLen:  19, // 1 (type) + 16 (ipv6) + 2 (port)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := encodeAddress(tt.hostName, tt.ip, tt.port)

			if (err != nil) != tt.wantError {
				t.Errorf("encodeAddress() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if err == nil {
				// Verify address type
				if buf[0] != tt.wantType {
					t.Errorf("encodeAddress() type = %d, want %d", buf[0], tt.wantType)
				}

				// Verify buffer length
				if len(buf) != tt.wantLen {
					t.Errorf("encodeAddress() len = %d, want %d", len(buf), tt.wantLen)
				}

				// Verify port encoding (last 2 bytes, big-endian)
				expectedPortHigh := byte(tt.port >> 8)
				expectedPortLow := byte(tt.port & 0xFF)
				if buf[len(buf)-2] != expectedPortHigh || buf[len(buf)-1] != expectedPortLow {
					t.Errorf("encodeAddress() port bytes = [%d, %d], want [%d, %d]",
						buf[len(buf)-2], buf[len(buf)-1], expectedPortHigh, expectedPortLow)
				}
			}
		})
	}
}

// TestEncodeAddressDomain tests domain name encoding
func TestEncodeAddressDomain(t *testing.T) {
	tests := []struct {
		name      string
		hostName  string
		port      uint16
		wantError bool
		errMsg    string
	}{
		{
			name:     "short domain",
			hostName: "test.com",
			port:     80,
		},
		{
			name:     "long domain",
			hostName: "very-long-subdomain.example.co.uk",
			port:     443,
		},
		{
			name:      "domain too long (> 255 bytes)",
			hostName:  string(make([]byte, 256)),
			port:      80,
			wantError: true,
			errMsg:    "too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := encodeAddress(tt.hostName, netip.Addr{}, tt.port)

			if (err != nil) != tt.wantError {
				t.Errorf("encodeAddress() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("encodeAddress() error = %v, should contain %v", err.Error(), tt.errMsg)
				}
			}

			if err == nil {
				// Verify domain encoding
				if buf[0] != AddrTypeDomain {
					t.Errorf("encodeAddress() type = %d, want %d", buf[0], AddrTypeDomain)
				}

				// Verify domain length
				domainLen := buf[1]
				if domainLen != byte(len(tt.hostName)) {
					t.Errorf("encodeAddress() domain length = %d, want %d", domainLen, len(tt.hostName))
				}

				// Verify domain content
				decodedDomain := string(buf[2 : 2+domainLen])
				if decodedDomain != tt.hostName {
					t.Errorf("encodeAddress() domain = %s, want %s", decodedDomain, tt.hostName)
				}
			}
		})
	}
}

// TestSendRequest tests the SendRequest function
func TestSendRequest(t *testing.T) {
	tests := []struct {
		name      string
		hostName  string
		ip        netip.Addr
		port      uint16
		useNil    bool
		wantError bool
		errMsg    string
	}{
		{
			name:     "send IPv4 address",
			hostName: "",
			ip:       netip.MustParseAddr("10.0.0.1"),
			port:     80,
		},
		{
			name:     "send domain address",
			hostName: "example.com",
			ip:       netip.MustParseAddr("192.168.1.1"),
			port:     443,
		},
		{
			name:      "nil connection",
			useNil:    true,
			hostName:  "example.com", // Provide hostname to avoid address encoding error
			ip:        netip.MustParseAddr("192.168.1.1"),
			port:      80,
			wantError: true,
			errMsg:    "connection is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var conn net.Conn
			if !tt.useNil {
				conn = &MockConn{}
			}
			// else conn remains nil (nil interface, not nil pointer)

			err := SendRequest(conn, tt.hostName, tt.ip, tt.port)

			if (err != nil) != tt.wantError {
				t.Errorf("SendRequest() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("SendRequest() error = %v, should contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

// MockAddr implements net.Addr for testing
type MockAddr struct{}

func (m *MockAddr) Network() string { return "tcp" }
func (m *MockAddr) String() string  { return "mock://localhost" }

// MockConn implements net.Conn for testing
type MockConn struct {
	written []byte
}

// Write writes data to the mock connection
func (m *MockConn) Write(b []byte) (int, error) {
	m.written = append(m.written, b...)
	return len(b), nil
}

// Read is not implemented for this mock
func (m *MockConn) Read(b []byte) (int, error) {
	return 0, nil
}

// Close is not implemented for this mock
func (m *MockConn) Close() error {
	return nil
}

// LocalAddr returns a mock address
func (m *MockConn) LocalAddr() net.Addr {
	return &MockAddr{}
}

// RemoteAddr returns a mock address
func (m *MockConn) RemoteAddr() net.Addr {
	return &MockAddr{}
}

// SetDeadline is not implemented for this mock
func (m *MockConn) SetDeadline(time.Time) error {
	return nil
}

// SetReadDeadline is not implemented for this mock
func (m *MockConn) SetReadDeadline(time.Time) error {
	return nil
}

// SetWriteDeadline is not implemented for this mock
func (m *MockConn) SetWriteDeadline(time.Time) error {
	return nil
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
