package iprisk

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Result 是一次 IP 风控检测的结构化结果。
// RiskScore: 0-100 风控值(越低越干净), -1 表示未知。
type Result struct {
	IP        string
	Country   string
	RiskScore int    // 0-100, -1 = 未知
	RiskPct   string // "37%" 或 "?"
	Emoji     string
	IPAttr    string // 机房 / 住宅 / 未知
	IPSource  string // 代理 / 原生 / 未知
	Error     string
}

// FullString 返回与 IP-Stream-Checker 一致的汇总标注 【emoji 属性|来源】。
func (r *Result) FullString() string {
	return fmt.Sprintf("【%s %s|%s】", r.Emoji, r.IPAttr, r.IPSource)
}

// emojiFor 按风控值返回 emoji(风控值越低越干净,≤30 即绿色可用)。
func emojiFor(score int) string {
	switch {
	case score <= 10:
		return "⚪"
	case score <= 30:
		return "🟢"
	case score <= 50:
		return "🟡"
	case score <= 70:
		return "🟠"
	case score <= 90:
		return "🔴"
	default:
		return "⚫"
	}
}

var browserUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36"

// ==================== 源1: IPPure ====================
// https://my.ippure.com/v1/info
// 免费、无 key、无反爬。请求走节点时,返回的就是节点出口 IP 的风控值
// (fraudScore 0-100,综合 IP 类型 / VPN/Proxy / 滥用历史)。

type ippureResp struct {
	IP            string `json:"ip"`
	CountryCode   string `json:"countryCode"`
	FraudScore    int    `json:"fraudScore"`
	IsResidential bool   `json:"isResidential"`
}

// queryIppure 走节点查询 ippure(调用方 IP 即节点出口 IP)。
func queryIppure(c *http.Client) (*ippureResp, error) {
	req, err := http.NewRequest("GET", "https://my.ippure.com/v1/info", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", browserUA)
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ippure HTTP %d", resp.StatusCode)
	}
	var d ippureResp
	if err := json.Unmarshal(body, &d); err != nil {
		return nil, err
	}
	if d.IP == "" {
		return nil, fmt.Errorf("ippure 返回空响应")
	}
	return &d, nil
}

// ==================== 源2: IPOK ====================
// https://ipok.io/api/ip
// 聚合 ip-api / proxycheck / AbuseIPDB / ipapi.is / StopForumSpam / IPOK-DB
// 六源加权的风控值(0-100),并给出 signals(proxy/vpn/tor/hosting)用于标注来源。
// 两种调用方式:
//   - 直连指定 IP: /api/ip?ip=<出口IP>(源1拿到出口 IP 后用,不经节点更快更稳)
//   - 走节点 caller 模式: /api/ip?lang=zh-CN(源1失败时,请求走节点,服务端识别出口 IP)
//
// 注意: ipok 需要浏览器 UA,否则返回 403。

type ipokResp struct {
	Geo struct {
		IP          string `json:"ip"`
		CountryCode string `json:"countryCode"`
	} `json:"geo"`
	Risk    int      `json:"risk"`
	Signals []string `json:"signals"`
}

func parseIpok(body []byte, status int) (*ipokResp, error) {
	if status != http.StatusOK {
		return nil, fmt.Errorf("ipok HTTP %d", status)
	}
	var d ipokResp
	if err := json.Unmarshal(body, &d); err != nil {
		return nil, err
	}
	if d.Geo.IP == "" && d.Risk == 0 && len(d.Signals) == 0 {
		return nil, fmt.Errorf("ipok 返回空响应")
	}
	return &d, nil
}

func doIpokReq(req *http.Request, client *http.Client) (*ipokResp, error) {
	req.Header.Set("User-Agent", browserUA)
	req.Header.Set("Referer", "https://ipok.io/")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parseIpok(body, resp.StatusCode)
}

// queryIpokByIP 直连查询指定 IP(不经节点)。
func queryIpokByIP(ip string) (*ipokResp, error) {
	client := &http.Client{Timeout: 12 * time.Second}
	req, err := http.NewRequest("GET", "https://ipok.io/api/ip?lang=zh-CN&ip="+ip, nil)
	if err != nil {
		return nil, err
	}
	return doIpokReq(req, client)
}

// queryIpokCaller 走节点查询(调用方 IP 即节点出口 IP)。
func queryIpokCaller(c *http.Client) (*ipokResp, error) {
	req, err := http.NewRequest("GET", "https://ipok.io/api/ip?lang=zh-CN", nil)
	if err != nil {
		return nil, err
	}
	return doIpokReq(req, c)
}

func hasAny(signals []string, keys ...string) bool {
	for _, s := range signals {
		for _, k := range keys {
			if s == k {
				return true
			}
		}
	}
	return false
}

// Check 通过节点出口做双源风控检测(ippure + ipok 互为备选):
//  1. ippure 走节点 → fraudScore 风控值 + 出口 IP + 国家 + 是否住宅(快)
//  2. ipok: 源1成功则直连 ?ip=出口IP 聚合六源风控; 源1失败则走节点 caller 模式
//
// 任一源成功即有风控值;两个都失败才返回未知(❓),此时节点无风控标签,
// 不会被风险过滤误杀也不会被保留(与"测不出"语义一致)。
//
// 注意: 入参 mediaClient 的超时较短(media-check-timeout,默认5s),慢节点上
// 一次 ippure 往返经常不够,因此这里基于同一传输层另建一个 15s 超时的客户端,
// 风控查询不受媒体检测超时限制。
func Check(mediaClient *http.Client) *Result {
	res := &Result{RiskScore: -1, RiskPct: "?", Emoji: "❓", IPAttr: "未知", IPSource: "未知"}

	// 独立超时: 沿用节点传输层,但给风控查询 15s(慢节点 5s 不够)
	riskClient := &http.Client{
		Transport: mediaClient.Transport,
		Timeout:   15 * time.Second,
	}

	// ---- 源1: ippure(走节点) ----
	ip1, err1 := queryIppure(riskClient)
	if err1 == nil && ip1 != nil {
		res.IP = ip1.IP
		res.Country = ip1.CountryCode
		res.RiskScore = ip1.FraudScore
		res.RiskPct = fmt.Sprintf("%d%%", ip1.FraudScore)
		res.Emoji = emojiFor(ip1.FraudScore)
		if ip1.IsResidential {
			res.IPAttr = "住宅"
			res.IPSource = "原生"
		} else {
			res.IPAttr = "机房"
		}
	}

	// ---- 源2: ipok(互为备选 + 补充代理/原生信号) ----
	var ip2 *ipokResp
	if res.IP != "" {
		ip2, _ = queryIpokByIP(res.IP) // 直连指定 IP,不经节点
	} else {
		ip2, _ = queryIpokCaller(riskClient) // 走节点 caller 模式
	}
	if ip2 != nil {
		if res.RiskScore < 0 { // ippure 失败,用 ipok 的风控值顶上
			res.RiskScore = ip2.Risk
			res.RiskPct = fmt.Sprintf("%d%%", ip2.Risk)
			res.Emoji = emojiFor(ip2.Risk)
		}
		if res.IP == "" {
			res.IP = ip2.Geo.IP
		}
		if res.Country == "" {
			res.Country = ip2.Geo.CountryCode
		}
		// 属性: 命中 hosting 信号 → 机房,否则住宅
		if hasAny(ip2.Signals, "hosting") {
			res.IPAttr = "机房"
		} else if res.IPAttr == "未知" {
			res.IPAttr = "住宅"
		}
		// 来源: 命中 proxy/vpn/tor 信号 → 代理,否则原生
		if hasAny(ip2.Signals, "proxy", "vpn", "tor") {
			res.IPSource = "代理"
		} else if res.IPSource == "未知" {
			res.IPSource = "原生"
		}
	}

	if res.RiskScore < 0 {
		res.Error = fmt.Sprintf("ippure: %v; ipok: 失败", err1)
	}
	return res
}
