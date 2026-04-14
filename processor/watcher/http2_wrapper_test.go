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

func extractHeaderBlockFragments(t *testing.T, frames []byte) []byte {
	t.Helper()

	var block bytes.Buffer
	reader := bytes.NewReader(frames)
	framer := http2.NewFramer(nil, reader)

	for {
		frame, err := framer.ReadFrame()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("ReadFrame() error = %v", err)
		}

		switch f := frame.(type) {
		case *http2.HeadersFrame:
			block.Write(f.HeaderBlockFragment())
		case *http2.ContinuationFrame:
			block.Write(f.HeaderBlockFragment())
		default:
			t.Fatalf("unexpected frame type %T while extracting header block", frame)
		}
	}

	return block.Bytes()
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
	if got := w.requestEncoderToServer.MaxDynamicTableSize(); got != 8192 {
		t.Fatalf("encoder dynamic table size = %d, want %d", got, 8192)
	}
}

func TestWatcherWrapConnWrite_PassthroughsServerFrames(t *testing.T) {
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
	readBuf := make(chan []byte, 1)
	go func() {
		buf := make([]byte, len(frame))
		_, err := io.ReadFull(clientConn, buf)
		readBuf <- buf
		readDone <- err
	}()

	if _, err := w.Write(frame); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := <-readDone; err != nil {
		t.Fatalf("reader error = %v", err)
	}
	if got := <-readBuf; !bytes.Equal(got, frame) {
		t.Fatalf("Write() passthrough mismatch")
	}
}

func TestRebuildReqHeadersWithInjectedField_SequentialStreamsRemainConnectionDecodable(t *testing.T) {
	w := NewWatcherWrapConn(nil)
	w.ParseSettingsFrame([]byte{
		0x00, 0x01, // HEADER_TABLE_SIZE
		0x00, 0x00, 0x10, 0x00, // 4096
	}, http2SettingsSourceServer)

	firstFrames, firstFields, err := w.rebuildReqHeadersWithInjectedField(
		[]hpack.HeaderField{
			{Name: ":method", Value: "GET"},
			{Name: ":scheme", Value: "https"},
			{Name: ":authority", Value: "example.com"},
			{Name: ":path", Value: "/one"},
			{Name: "user-agent", Value: "agent-a"},
		},
		1,
		false,
		http2.PriorityParam{},
		"",
		"",
	)
	if err != nil {
		t.Fatalf("first rebuildReqHeadersWithInjectedField() error = %v", err)
	}
	secondFrames, secondFields, err := w.rebuildReqHeadersWithInjectedField(
		[]hpack.HeaderField{
			{Name: ":method", Value: "GET"},
			{Name: ":scheme", Value: "https"},
			{Name: ":authority", Value: "example.com"},
			{Name: ":path", Value: "/two"},
			{Name: "user-agent", Value: "agent-a"},
		},
		3,
		true,
		http2.PriorityParam{},
		"",
		"",
	)
	if err != nil {
		t.Fatalf("second rebuildReqHeadersWithInjectedField() error = %v", err)
	}

	serverDecoder := hpack.NewDecoder(4096, nil)

	firstDecoded, err := w.decodeHeaderBlock(extractHeaderBlockFragments(t, firstFrames), true)
	if err != nil {
		t.Fatalf("local decode first block error = %v", err)
	}
	if got := firstDecoded[":path"]; got != "/one" {
		t.Fatalf("decoded first path = %q, want %q", got, "/one")
	}

	serverHeaders1 := map[string]string{}
	serverDecoder.SetEmitFunc(func(f hpack.HeaderField) { serverHeaders1[f.Name] = f.Value })
	if _, err := serverDecoder.Write(extractHeaderBlockFragments(t, firstFrames)); err != nil {
		t.Fatalf("server decoder first block error = %v", err)
	}
	if got := serverHeaders1[":path"]; got != "/one" {
		t.Fatalf("server decoded first path = %q, want %q", got, "/one")
	}

	serverHeaders2 := map[string]string{}
	serverDecoder.SetEmitFunc(func(f hpack.HeaderField) { serverHeaders2[f.Name] = f.Value })
	if _, err := serverDecoder.Write(extractHeaderBlockFragments(t, secondFrames)); err != nil {
		t.Fatalf("server decoder second block error = %v", err)
	}
	if got := serverHeaders2[":path"]; got != "/two" {
		t.Fatalf("server decoded second path = %q, want %q", got, "/two")
	}

	if len(firstFields) == 0 || len(secondFields) == 0 {
		t.Fatal("expected rewritten header field snapshots to be returned")
	}
}

func TestRebuildReqHeadersWithInjectedField_UsesLatestServerMaxFrameSize(t *testing.T) {
	w := NewWatcherWrapConn(nil)
	w.ParseSettingsFrame([]byte{
		0x00, 0x05, // MAX_FRAME_SIZE
		0x00, 0x00, 0x00, 0x10, // 16
	}, http2SettingsSourceServer)
	w.ParseSettingsFrame([]byte{
		0x00, 0x05, // MAX_FRAME_SIZE
		0x00, 0x00, 0x00, 0x20, // 32
	}, http2SettingsSourceServer)

	var largeValue bytes.Buffer
	for i := 0; i < 80; i++ {
		largeValue.WriteByte('a')
	}

	frames, _, err := w.rebuildReqHeadersWithInjectedField(
		[]hpack.HeaderField{
			{Name: ":method", Value: "GET"},
			{Name: ":scheme", Value: "https"},
			{Name: ":authority", Value: "example.com"},
			{Name: ":path", Value: "/frame-size"},
			{Name: "x-large", Value: largeValue.String()},
		},
		1,
		false,
		http2.PriorityParam{},
		"",
		"",
	)
	if err != nil {
		t.Fatalf("rebuildReqHeadersWithInjectedField() error = %v", err)
	}

	reader := bytes.NewReader(frames)
	framer := http2.NewFramer(nil, reader)
	for {
		frame, err := framer.ReadFrame()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("ReadFrame() error = %v", err)
		}
		switch f := frame.(type) {
		case *http2.HeadersFrame:
			if got := len(f.HeaderBlockFragment()); got > 32 {
				t.Fatalf("headers fragment length = %d, want <= 32", got)
			}
		case *http2.ContinuationFrame:
			if got := len(f.HeaderBlockFragment()); got > 32 {
				t.Fatalf("continuation fragment length = %d, want <= 32", got)
			}
		}
	}
}
