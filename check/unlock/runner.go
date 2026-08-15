package unlock

import (
	"log/slog"
	"sync"

	"github.com/metacubex/mihomo/constant"
	"github.com/rongrong13/sub-test-max/internal/mediatest/core"
	"github.com/rongrong13/sub-test-max/internal/mediatest/providers"
)

// providerRegistry 把 mediatest 所有区域测试表按服务名平铺成一个 map,
// 供按名选择单个服务使用(与 IP-Stream-Checker 里 media_unlock.py 的
// -providers 行为一致)。同名服务(如 Bahamut Anime 出现在多个区域)只保留第一个。
var providerRegistry = buildRegistry()

func buildRegistry() map[string]func(client core.HttpClient) core.Result {
	reg := map[string]func(client core.HttpClient) core.Result{}
	for _, list := range [][]providers.TestItem{
		providers.GlobeTests,
		providers.HongKongTests,
		providers.TaiwanTests,
		providers.JapanTests,
		providers.KoreaTests,
		providers.NorthAmericaTests,
		providers.SouthAmericaTests,
		providers.EuropeTests,
		providers.AfricaTests,
		providers.SouthEastAsiaTests,
		providers.OceaniaTests,
		providers.AITests,
	} {
		for _, t := range list {
			if _, ok := reg[t.Name]; !ok && t.Func != nil {
				reg[t.Name] = t.Func
			}
		}
	}
	return reg
}

// ProviderNames 返回内嵌 MediaUnlockTest 支持的全部服务名(测试列出名字)。
func ProviderNames() []string {
	return providerNames
}

var providerNames = providerNameList()

func providerNameList() []string {
	seen := map[string]bool{}
	names := make([]string, 0, len(providerRegistry))
	for _, list := range [][]providers.TestItem{
		providers.GlobeTests,
		providers.HongKongTests,
		providers.TaiwanTests,
		providers.JapanTests,
		providers.KoreaTests,
		providers.NorthAmericaTests,
		providers.SouthAmericaTests,
		providers.EuropeTests,
		providers.AfricaTests,
		providers.SouthEastAsiaTests,
		providers.OceaniaTests,
		providers.AITests,
	} {
		for _, t := range list {
			if t.Func != nil && !seen[t.Name] {
				seen[t.Name] = true
				names = append(names, t.Name)
			}
		}
	}
	return names
}

// Result 是单个流媒体服务的检测结果(对应 mediatest -json 单条记录)。
type Result struct {
	Name       string
	Status     int
	StatusText string
	Region     string
	Info       string
	Ok         bool
}

func toResult(name string, r core.Result) Result {
	st := "unknown"
	ok := false
	switch r.Status {
	case core.StatusOK:
		st, ok = "ok", true
	case core.StatusRestricted:
		st = "restricted"
	case core.StatusNo:
		st = "no"
	case core.StatusNetworkErr:
		st = "network_error"
	case core.StatusErr:
		st = "error"
	case core.StatusBanned:
		st = "banned"
	case core.StatusFailed:
		st = "failed"
	case core.StatusUnexpected:
		st = "unexpected"
	}
	return Result{Name: name, Status: r.Status, StatusText: st, Region: r.Region, Info: r.Info, Ok: ok}
}

var clientMu sync.Mutex

// newBridgedClient 为该节点起一个本地 SOCKS5 桥,并返回一个走该桥的
// tls_client 客户端。并发环境下 core 的 SocksProxy 是包级全局,因此创建
// 客户端时用互斥锁保护"写全局 + 创建"两步;客户端创建后即常驻自己的代理
// 配置,互不干扰。
func newBridgedClient(proxy constant.Proxy) (*socks5Bridge, core.HttpClient, error) {
	bridge, err := newBridge(proxy)
	if err != nil {
		return nil, nil, err
	}

	clientMu.Lock()
	// 用 socks5:// 而非 http://: SOCKS5 会把目标域名原样交给节点解析,
	// 绕开容器内 DNS,兼容软路由/OpenClash 直连环境。
	core.HTTPProxy = ""
	core.SocksProxy = "socks5://" + bridge.addr()
	client := core.NewHttpClient(0)
	core.SocksProxy = ""
	clientMu.Unlock()

	return bridge, client, nil
}

// Run 对指定节点测试一组服务,返回与 IP-Stream-Checker 一致的检测结果。
// providers 为空时测试全部服务(极慢)。
func Run(proxy constant.Proxy, providerNames []string, conc int, mediaTimeout int) []Result {
	if len(providerNames) == 0 {
		providerNames = ProviderNames()
	}
	if conc <= 0 {
		conc = 20
	}
	if mediaTimeout <= 0 {
		mediaTimeout = 15
	}

	// 过滤出注册表中存在的服务
	var targetFuncs []struct {
		name string
		fn   func(client core.HttpClient) core.Result
	}
	for _, n := range providerNames {
		if fn, ok := providerRegistry[n]; ok {
			targetFuncs = append(targetFuncs, struct {
				name string
				fn   func(client core.HttpClient) core.Result
			}{n, fn})
		}
	}
	if len(targetFuncs) == 0 {
		return nil
	}

	bridge, client, err := newBridgedClient(proxy)
	if err != nil {
		slog.Debug("unlock: 创建节点桥失败", "err", err)
		return nil
	}
	defer bridge.close()

	sem := make(chan struct{}, conc)
	results := make([]Result, len(targetFuncs))
	var wg sync.WaitGroup
	for i, tf := range targetFuncs {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, tf struct {
			name string
			fn   func(client core.HttpClient) core.Result
		}) {
			defer wg.Done()
			defer func() { <-sem }()

			res := tf.fn(client)
			results[idx] = toResult(tf.name, res)
		}(i, tf)
	}
	wg.Wait()
	return results
}
