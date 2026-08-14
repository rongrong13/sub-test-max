package unlock

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
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

// TestBridgeConnectRelay 验证本地 CONNECT 代理能把隧道流量正确转发到上游节点。
// 这里用 echo server 模拟节点输出的目标服务:客户端经桥发出 CONNECT 后,
// 写入的数据应被 echo 原样回传。
func TestBridgeConnectRelay(t *testing.T) {
	echoAddr, closeEcho := startEchoServer(t)
	defer closeEcho()

	b := &httpConnectProxy{
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

	// 发送 CONNECT 到 echo server(目标地址无需真实可拨,因为我们用 dialNode 固定转发)
	if _, err := fmt.Fprintf(client, "CONNECT example.com:443 HTTP/1.1\r\nHost: example.com:443\r\n\r\n"); err != nil {
		t.Fatal(err)
	}

	br := bufio.NewReader(client)
	status, err := br.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(status, "200") {
		t.Fatalf("expected 200 Connection Established, got %q", status)
	}
	// 消费剩余响应头
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if line == "\r\n" || line == "\n" {
			break
		}
	}

	// 通过隧道发送数据,应原样回传
	payload := "hello-via-bridge"
	if _, err := client.Write([]byte(payload)); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, len(payload))
	if _, err := io.ReadFull(br, buf); err != nil {
		t.Fatal(err)
	}
	if string(buf) != payload {
		t.Fatalf("relay mismatch: got %q, want %q", string(buf), payload)
	}
}
