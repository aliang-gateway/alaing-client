package tls

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"

	"aliang.one/nursorgate/common/logger"
)

func parseSNIFromBuffer(buf []byte) (string, []byte, error) {

	// 确保是 TLS ClientHello
	if len(buf) < 5 || buf[0] != 0x16 || buf[1] != 0x03 {
		return "", nil, fmt.Errorf("not a valid TLS ClientHello")
	}

	// 解析 TLS 记录层头部
	handshakeLength := int(binary.BigEndian.Uint16(buf[3:5])) + 5
	if handshakeLength > len(buf) {
		return "", nil, fmt.Errorf("incomplete TLS handshake")
	}

	// 跳过 TLS 记录层 (5 bytes) 和 Handshake 类型 (1 byte)
	pos := 5 + 1
	if pos >= len(buf) {
		return "", nil, fmt.Errorf("invalid handshake length")
	}

	// 跳过 Handshake 长度 (3 bytes)
	pos += 3
	if pos >= len(buf) {
		return "", nil, fmt.Errorf("invalid handshake structure")
	}

	// 跳过 TLS 版本 (2 bytes)
	pos += 2
	if pos >= len(buf) {
		return "", nil, fmt.Errorf("invalid handshake version")
	}

	// 跳过 Random 字段 (32 bytes)
	pos += 32
	if pos >= len(buf) {
		return "", nil, fmt.Errorf("invalid random field")
	}

	// 跳过 Session ID 长度 (1 byte) + Session ID
	sessionIDLength := int(buf[pos])
	pos += 1 + sessionIDLength
	if pos >= len(buf) {
		return "", nil, fmt.Errorf("invalid session ID")
	}

	// 跳过 Cipher Suites 长度 (2 bytes) + Cipher Suites
	cipherSuitesLength := int(binary.BigEndian.Uint16(buf[pos:]))
	pos += 2 + cipherSuitesLength
	if pos >= len(buf) {
		return "", nil, fmt.Errorf("invalid cipher suites")
	}

	// 跳过 Compression Length (1 byte) + Compression Methods
	compressionMethodsLength := int(buf[pos])
	pos += 1 + compressionMethodsLength
	if pos >= len(buf) {
		return "", nil, fmt.Errorf("invalid compression methods")
	}

	// 解析扩展字段
	if pos+2 > len(buf) {
		return "", nil, fmt.Errorf("invalid extension length")
	}
	extensionsLength := int(binary.BigEndian.Uint16(buf[pos:]))
	pos += 2
	if pos+extensionsLength > len(buf) {
		return "", nil, fmt.Errorf("invalid extensions")
	}

	// 遍历扩展字段，寻找 SNI (0x0000)
	for pos+4 <= len(buf) {
		extType := binary.BigEndian.Uint16(buf[pos:])
		extLen := binary.BigEndian.Uint16(buf[pos+2:])
		pos += 4
		if pos+int(extLen) > len(buf) {
			return "", nil, fmt.Errorf("invalid extension data")
		}

		// 发现 SNI 扩展
		if extType == 0x0000 {
			pos += 2 // 跳过 SNI 列表长度
			if pos+1 > len(buf) {
				return "", nil, fmt.Errorf("invalid SNI length")
			}

			nameType := buf[pos]
			pos++
			if nameType != 0x00 { // 仅支持 HostName 类型
				return "", nil, fmt.Errorf("unsupported SNI type")
			}

			if pos+2 > len(buf) {
				return "", nil, fmt.Errorf("invalid SNI host length")
			}

			nameLen := int(binary.BigEndian.Uint16(buf[pos:]))
			pos += 2
			if pos+nameLen > len(buf) {
				return "", nil, fmt.Errorf("invalid SNI hostname")
			}

			serverName := string(buf[pos : pos+nameLen])
			return serverName, buf, nil
		}

		pos += int(extLen) // 跳过当前扩展
	}

	return "", nil, fmt.Errorf("SNI not found")
}

func ExtractSNI(conn net.Conn) (string, []byte, error) {
	var totalBuf []byte
	buf := make([]byte, 4096)
	err := conn.SetReadDeadline(time.Now().Add(60 * 2 * time.Second))
	if err != nil {
		return "", nil, err
	} // 防止阻塞

	for {
		n, err := conn.Read(buf)
		if n > 0 {
			totalBuf = append(totalBuf, buf[:n]...)
		}

		if len(totalBuf) >= 5 {
			if totalBuf[0] != 0x16 {
				return "", totalBuf, fmt.Errorf("not a TLS handshake: first byte is 0x%x", totalBuf[0])
			}
			recordLen := int(binary.BigEndian.Uint16(totalBuf[3:5]))
			expectedLen := recordLen + 5
			if len(totalBuf) >= expectedLen {
				break // got full TLS record
			}
		}

		if err != nil {
			if err == io.EOF {
				if len(totalBuf) < 5 {
					return "", nil, fmt.Errorf("EOF before complete TLS ClientHello len: %d", len(totalBuf))
				}
				break
			}
			logger.Warn("failure in reading ClientHello", err)
			return "", nil, err
		}
	}

	s, _, err := parseSNIFromBuffer(totalBuf)
	if err != nil {
		logger.Warn("failure in reading sni", err)
		return "", nil, err
	}
	return s, totalBuf, nil
}
