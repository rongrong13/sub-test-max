package unlock

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"strconv"

	"github.com/metacubex/mihomo/constant"
)

// socks5Bridge 是一个极简的本地 SOCKS5 代理(无认证)。
//
// 目的: 让 MediaUnlockTest 的 tls_client 通过 socks5:// 走 mihomo 节点出口。
// 用 SOCKS5 而不是 HTTP CONNECT 的关键原因:
//   - tls_client 的 HTTP CONNECT 模式会先在容器内自己解析目标域名(走容器 DNS),
//     在软路由/OpenClash 直连环境下,容器 DNS 解析免费节点域名经常失败
//     (dns resolve failed),导致媒体检测全部失败。
//   - SOCKS5 会把目标域名原样发给 SOCKS 服务器,由我们转交给 mihomo 节点
//     的 DialContext 解析并出网,绕开容器 DNS,与 subs-check 原项目行为一致。
type socks5Bridge struct {
	ln       net.Listener
	dialNode func(ctx context.Context, network, addr string) (net.Conn, error)
}

func newBridge(proxy constant.Proxy) (*socks5Bridge, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	b := &socks5Bridge{
		ln: ln,
		dialNode: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, portStr, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			u16Port, err := strconv.ParseUint(portStr, 10, 16)
			if err != nil {
				return nil, err
			}
			return proxy.DialContext(ctx, &constant.Metadata{
				Host:    host,
				DstPort: uint16(u16Port),
			})
		},
	}
	go b.serve()
	return b, nil
}

func (b *socks5Bridge) addr() string { return b.ln.Addr().String() }

func (b *socks5Bridge) close() {
	if b.ln != nil {
		b.ln.Close()
	}
}

func (b *socks5Bridge) serve() {
	for {
		conn, err := b.ln.Accept()
		if err != nil {
			return
		}
		go b.handle(conn)
	}
}

func (b *socks5Bridge) handle(conn net.Conn) {
	defer conn.Close()

	// 1. 握手: 客户端发 [05 01 00], 我们回 [05 00] 表示无认证
	head := make([]byte, 3)
	if _, err := io.ReadFull(conn, head); err != nil {
		return
	}
	if head[0] != 0x05 {
		return
	}
	if _, err := conn.Write([]byte{0x05, 0x00}); err != nil {
		return
	}

	// 2. 读取 CONNECT 请求: 05 01 00 ATYP 地址 端口
	req := make([]byte, 4)
	if _, err := io.ReadFull(conn, req); err != nil {
		return
	}
	if req[0] != 0x05 || req[1] != 0x01 {
		return
	}

	var host string
	switch req[3] {
	case 0x01: // IPv4
		ip := make([]byte, 4)
		if _, err := io.ReadFull(conn, ip); err != nil {
			return
		}
		host = net.IP(ip).String()
	case 0x03: // 域名
		l := make([]byte, 1)
		if _, err := io.ReadFull(conn, l); err != nil {
			return
		}
		d := make([]byte, int(l[0]))
		if _, err := io.ReadFull(conn, d); err != nil {
			return
		}
		host = string(d)
	case 0x04: // IPv6
		ip := make([]byte, 16)
		if _, err := io.ReadFull(conn, ip); err != nil {
			return
		}
		host = net.IP(ip).String()
	default:
		return
	}
	portBytes := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBytes); err != nil {
		return
	}
	port := binary.BigEndian.Uint16(portBytes)
	target := net.JoinHostPort(host, strconv.Itoa(int(port)))

	// 3. 用 mihomo 节点拨号
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	upstream, err := b.dialNode(ctx, "tcp", target)
	if err != nil {
		return
	}
	defer upstream.Close()

	// 4. 回 CONNECT 成功: 05 00 00 01 0.0.0.0 0
	if _, err := conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0}); err != nil {
		return
	}

	// 5. 双向透传
	done := make(chan struct{}, 2)
	go func() { io.Copy(upstream, conn); upstream.Close(); done <- struct{}{} }()
	go func() { io.Copy(conn, upstream); conn.Close(); done <- struct{}{} }()
	<-done
	<-done
}
