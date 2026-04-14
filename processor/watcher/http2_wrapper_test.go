package tls

import (
	"bytes"
	"io"
	"net"
	"testing"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

func buildHeaderBlockWithDynamicTableSize(t *testing.T, tableSize uint32, field hpack.HeaderField) []byte {
	t.Helper()

	var buf bytes.Buffer
	enc := hpack.NewEncoder(&buf)
	enc.SetMaxDynamicTableSizeLimit(tableSize)
	enc.SetMaxDynamicTableSize(tableSize)
	if err := enc.WriteField(field); err != nil {
		t.Fatalf("WriteField() error = %v", err)
	}
	return buf.Bytes()
}

func buildSettingsFrame(t *testing.T, setting http2.Setting) []byte {
	t.Helper()

	var buf bytes.Buffer
	framer := http2.NewFramer(&buf, nil)
	if err := framer.WriteSettings(setting); err != nil {
		t.Fatalf("WriteSettings() error = %v", err)
	}
	return buf.Bytes()
}

func TestParseSettingsFrame_ClientUpdatesResponseDecoder(t *testing.T) {
	w := NewWatcherWrapConn(nil)
	w.ParseSettingsFrame([]byte{
		0x00, 0x01, // HEADER_TABLE_SIZE
		0x00, 0x00, 0x20, 0x00, // 8192
	}, http2SettingsSourceClient)

	block := buildHeaderBlockWithDynamicTableSize(t, 8192, hpack.HeaderField{Name: ":status", Value: "200"})
	headers, err := w.decodeHeaderBlock(block, false)
	if err != nil {
		t.Fatalf("decodeHeaderBlock(response) error = %v", err)
	}
	if got := headers[":status"]; got != "200" {
		t.Fatalf("decoded response pseudo-header = %q, want %q", got, "200")
	}
}

func TestParseSettingsFrame_ServerUpdatesRequestDecoderAndEncoder(t *testing.T) {
	w := NewWatcherWrapConn(nil)
	w.ParseSettingsFrame([]byte{
		0x00, 0x01, // HEADER_TABLE_SIZE
		0x00, 0x00, 0x20, 0x00, // 8192
		0x00, 0x05, // MAX_FRAME_SIZE
		0x00, 0x00, 0x10, 0x00, // 4096
	}, http2SettingsSourceServer)

	block := buildHeaderBlockWithDynamicTableSize(t, 8192, hpack.HeaderField{Name: ":method", Value: "GET"})
	headers, err := w.decodeHeaderBlock(block, true)
	if err != nil {
		t.Fatalf("decodeHeaderBlock(request) error = %v", err)
	}
	if got := headers[":method"]; got != "GET" {
		t.Fatalf("decoded request pseudo-header = %q, want %q", got, "GET")
	}

	if got, ok := w.GetServerHTTP2Setting(SETTINGS_MAX_FRAME_SIZE); !ok || got != 4096 {
		t.Fatalf("GetServerHTTP2Setting(MAX_FRAME_SIZE) = (%d, %t), want (4096, true)", got, ok)
	}
	if got := w.hpackEncoderToServer.MaxDynamicTableSize(); got != 8192 {
		t.Fatalf("encoder dynamic table size = %d, want %d", got, 8192)
	}
}

func TestProcessHttp2ResponseFrame_UpdatesResponseLifecycle(t *testing.T) {
	w := NewWatcherWrapConn(nil)
	stream := w.getOrCreateStream(1)

	frame := []byte{
		0x00, 0x00, 0x02, // length
		0x00,                   // DATA
		0x01,                   // END_STREAM
		0x00, 0x00, 0x00, 0x01, // stream 1
		'o', 'k',
	}

	if err := w.processHttp2ResponseFrame(frame); err != nil {
		t.Fatalf("processHttp2ResponseFrame() error = %v", err)
	}
	if !stream.RespEndStream {
		t.Fatal("expected RespEndStream to be true")
	}
	if stream.ReqEndStream {
		t.Fatal("did not expect ReqEndStream to be changed by response DATA")
	}
}

func TestWatcherWrapConnWrite_ProcessesServerSettings(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	w := NewWatcherWrapConn(serverConn)
	w.prefetched = true

	frame := buildSettingsFrame(t, http2.Setting{
		ID:  http2.SettingHeaderTableSize,
		Val: 8192,
	})

	readDone := make(chan error, 1)
	go func() {
		_, err := io.ReadFull(clientConn, make([]byte, len(frame)))
		readDone <- err
	}()

	if _, err := w.Write(frame); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := <-readDone; err != nil {
		t.Fatalf("reader error = %v", err)
	}

	if got, ok := w.GetServerHTTP2Setting(SETTINGS_HEADER_TABLE_SIZE); !ok || got != 8192 {
		t.Fatalf("GetServerHTTP2Setting(HEADER_TABLE_SIZE) = (%d, %t), want (8192, true)", got, ok)
	}
}
