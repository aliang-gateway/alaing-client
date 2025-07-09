package test

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"nursor.org/nursorgate/common/logger"
	"testing"
	"time"

	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/waiter"
	//tungvisor "nursor.org/nursorgate/client/server/utils/tun_gvisor"
	//"nursor.org/nursorgate/client/utils/tun_gvisor/devices/tun"
)

func TestGvisor(t *testing.T) {
	//nicId := 123
	//createTUN, err := tun.CreateTUN("utun99", 1500)
	//if err != nil {
	//	t.Fatal(err)
	//}
	//s, err := tun.ConfigStack(createTUN, nicId)
	//if err != nil {
	//	t.Fatal(err)
	//}
	//go func() {
	//	for {
	//		select {
	//		case r := <-tun.TCPChan:
	//			println("new comming a tcp connection------")
	//			server.HandleTProxyHttpsConnection(r)
	//		}
	//	}
	//}()
	//err = tungvisor.ConfigureTunInterface("utun99")
	//if err != nil {
	//	t.Fatal(err)
	//}
	////err = utils.ConfigureTunRoute()
	////if err != nil {
	////	t.Fatal(err)
	////}
	//var wq waiter.Queue
	//ep, e := s.NewEndpoint(tcp.ProtocolNumber, ipv4.ProtocolNumber, &wq)
	//if e != nil {
	//	log.Fatal(e)
	//}
	//writeToEP(ep, nicId, s, &wq)
}
func writer(ch chan struct{}, ep tcpip.Endpoint) {
	defer func() {
		ep.Shutdown(tcpip.ShutdownWrite)
		close(ch)
	}()

	var b bytes.Buffer
	go func() {
		for {
			time.Sleep(time.Second * 10)
			b.Write([]byte("hello from goland"))
		}

	}()
	if err := func() error {
		for {
			//if _, err := b.ReadFrom(os.Stdin); err != nil {
			//	return fmt.Errorf("b.ReadFrom failed: %w", err)
			//}

			for b.Len() != 0 {
				if _, err := ep.Write(&b, tcpip.WriteOptions{Atomic: true}); err != nil {
					return fmt.Errorf("ep.Write failed: %s", err)
				} else {
					log.Println("success in write in data")
				}
			}
		}
	}(); err != nil {
		fmt.Printf("write fun break out for %s", err.Error())
	}
}

func writeToEP(ep tcpip.Endpoint, tunNicId int, tunStack *stack.Stack, wq *waiter.Queue) {
	// Issue connect request and wait for it to complete.

	addr := tcpip.AddrFromSlice(net.ParseIP("172.16.238.2").To4())
	remote := tcpip.FullAddress{
		Addr: addr,
		NIC:  tcpip.NICID(tunNicId),
		Port: 31465,
	}

	// 注册写事件
	waitEntry, notifyCh := waiter.NewChannelEntry(waiter.WritableEvents)
	wq.EventRegister(&waitEntry)
	defer wq.EventUnregister(&waitEntry) // 确保事件被取消注册

	// sourceAddr := tcpip.AddrFromSlice(net.ParseIP("172.16.113.250").To4())
	sourceAddr := tcpip.AddrFromSlice(net.ParseIP("10.0.0.1").To4())
	r, ter := tunStack.FindRoute(
		tcpip.NICID(tunNicId),
		sourceAddr,
		remote.Addr,
		ipv4.ProtocolNumber,
		true,
	)
	if ter != nil {
		logger.Error(ter)
	} else {
		log.Printf("Route found:")
		log.Printf("  Source: %v", r.LocalAddress().String())
		log.Printf("  Destination: %v", r.RemoteAddress().String())
		log.Printf("  NextHop: %v", r.NextHop().String())
		log.Printf("  NIC: %v", r.NICID())
	}

	log.Printf("Attempting to connect to %v:%d", remote.Addr, remote.Port)
	log.Printf("Connecting to IP: %s", remote.Addr.String())

	terr := ep.Connect(remote)
	if _, ok := terr.(*tcpip.ErrConnectStarted); ok {
		fmt.Println("Connect is pending...")
		select {
		case <-notifyCh:
			if err := ep.LastError(); err != nil {
				fmt.Printf("connection failed: %v", err)
			} else {
				fmt.Println("connection success")
				data := []byte("111")
				var r bytes.Reader
				r.Reset(data)
				if _, err := ep.Write(&r, tcpip.WriteOptions{}); err != nil {
					fmt.Printf("Failed to send data: %v\n", err)
				} else {
					fmt.Println("Data sent: 111")
				}
			}
		case <-time.After(time.Second * 3000):
			fmt.Printf("connection timeout after %v", time.Second*10)
			ep.Close()
		}
		terr = ep.LastError()
		if terr != nil {
			fmt.Println(terr.String())
		}
	}

	wq.EventUnregister(&waitEntry)

	if terr != nil {
		log.Fatal("Unable to connect: ", terr)
	}
	fmt.Println("Connected")

	// 读取数据
	waitEntry, notifyCh = waiter.NewChannelEntry(waiter.ReadableEvents)
	wq.EventRegister(&waitEntry)
	defer wq.EventUnregister(&waitEntry)

	var buf [1024]byte
	buffer := bytes.NewBuffer(buf[:0])

	for {
		buffer.Reset() // 重置buffer
		_, err := ep.Read(buffer, tcpip.ReadOptions{})
		if err != nil {
			if _, ok := err.(*tcpip.ErrClosedForReceive); ok {
				break
			}
			if _, ok := err.(*tcpip.ErrWouldBlock); ok {
				<-notifyCh
				continue
			}
			fmt.Printf("read failed: %v", err)
		}
		println(buffer.String())
	}
	ep.Close()
}
