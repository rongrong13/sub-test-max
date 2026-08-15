package unlock

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"testing"
)

// startEchoServer 起一个把收到的数据原样回写的 TCP 服务,作为测试上游。
func startEchoServer(t *testing.T) (addr string, close func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				io.Copy(c, c)
			}(conn)
		}
	}()
	return ln.Addr().String(), func() { ln.Close() }
}

// TestBridgeSocks5Relay 验证本地 SOCKS5 桥能把隧道流量正确转发到上游节点。
// 客户端以域名形式发起 CONNECT(dialNode 固定转发到 echo server),
// 验证 SOCKS5 域名解析路径与双向透传。
func TestBridgeSocks5Relay(t *testing.T) {
	echoAddr, closeEcho := startEchoServer(t)
	defer closeEcho()

	b := &socks5Bridge{
		dialNode: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("tcp", echoAddr)
		},
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	b.ln = ln
	go b.serve()
	defer b.close()

	client, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	// 1. 握手: 05 01 00 -> 05 00
	if _, err := client.Write([]byte{0x05, 0x01, 0x00}); err != nil {
		t.Fatal(err)
	}
	reply := make([]byte, 2)
	if _, err := io.ReadFull(client, reply); err != nil {
		t.Fatal(err)
	}
	if reply[0] != 0x05 || reply[1] != 0x00 {
		t.Fatalf("handshake failed: %v", reply)
	}

	// 2. CONNECT 域名 example.com:443(dialNode 固定转发到 echo server)
	host := "example.com"
	port := 443
	req := []byte{0x05, 0x01, 0x00, 0x03, byte(len(host))}
	req = append(req, []byte(host)...)
	pb := make([]byte, 2)
	binary.BigEndian.PutUint16(pb, uint16(port))
	req = append(req, pb...)
	if _, err := client.Write(req); err != nil {
		t.Fatal(err)
	}

	// 3. 期待成功应答: 05 00 00 01 0.0.0.0:0
	succ := make([]byte, 10)
	if _, err := io.ReadFull(client, succ); err != nil {
		t.Fatal(err)
	}
	if succ[0] != 0x05 || succ[1] != 0x00 {
		t.Fatalf("connect failed: %v", succ)
	}

	// 4. 通过隧道发送数据,应原样回传
	payload := "hello-via-socks5-bridge"
	if _, err := client.Write([]byte(payload)); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, len(payload))
	if _, err := io.ReadFull(client, buf); err != nil {
		t.Fatal(err)
	}
	if string(buf) != payload {
		t.Fatalf("relay mismatch: got %q, want %q", string(buf), payload)
	}
}
