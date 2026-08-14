package unlock

import (
	"bufio"
	"context"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/metacubex/mihomo/constant"
)

// httpConnectProxy 是一个极其精简的本地 HTTP CONNECT 代理。
type httpConnectProxy struct {
	ln       net.Listener
	dialNode func(ctx context.Context, network, addr string) (net.Conn, error)
}

func newBridge(proxy constant.Proxy) (*httpConnectProxy, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	b := &httpConnectProxy{
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

func (b *httpConnectProxy) addr() string { return b.ln.Addr().String() }

func (b *httpConnectProxy) close() {
	if b.ln != nil {
		b.ln.Close()
	}
}

func (b *httpConnectProxy) serve() {
	for {
		conn, err := b.ln.Accept()
		if err != nil {
			return
		}
		go b.handle(conn)
	}
}

func (b *httpConnectProxy) handle(conn net.Conn) {
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	br := bufio.NewReader(conn)
	reqLine, err := br.ReadString('\n')
	if err != nil {
		return
	}
	fields := strings.Fields(reqLine)
	if len(fields) < 2 || !strings.EqualFold(fields[0], "CONNECT") {
		return
	}
	target := fields[1]

	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return
		}
		if line == "\r\n" || line == "\n" {
			break
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	upstream, err := b.dialNode(ctx, "tcp", target)
	cancel()
	if err != nil {
		return
	}

	conn.SetWriteDeadline(time.Now().Add(15 * time.Second))
	if _, err := conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
		upstream.Close()
		return
	}
	conn.SetWriteDeadline(time.Time{})

	done := make(chan struct{}, 2)
	go func() {
		io.Copy(upstream, br)
		upstream.Close()
		done <- struct{}{}
	}()
	go func() {
		io.Copy(conn, upstream)
		conn.Close()
		done <- struct{}{}
	}()
	<-done
	<-done
}
