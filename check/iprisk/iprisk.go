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

// Check 通过节点出口做双源风控检测(ippure 为主,ipok 兜底):
//  1. 主源 ippure 走节点 → fraudScore 风控值 + 出口 IP + 国家 + 是否住宅(免费无key)
//  2. 仅当 ippure 失败时,用 ipok 走节点 caller 模式兜底(聚合六源风控)
//
// 注意: ipok 不作为常规路径——软路由实测每个节点都等 ipok 直连(慢/限流)
// 会把流水线拖垮,导致风控查询集体超时。ipok 只当 ippure 挂掉时才调用。
//
// 入参 mediaClient 的超时较短(media-check-timeout,默认5s),慢节点上一次
// ippure 往返经常不够,因此这里基于同一传输层另建一个 15s 超时的客户端,
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
		if ip1.IsResidential {
			res.IPAttr = "住宅"
			res.IPSource = "原生"
		} else {
			res.IPAttr = "机房"
		}

		if ip1.FraudScore > 0 {
			// 正常分值直接采用
			res.RiskScore = ip1.FraudScore
			res.RiskPct = fmt.Sprintf("%d%%", ip1.FraudScore)
			res.Emoji = emojiFor(ip1.FraudScore)
		} else {
			// 0 分不可信: ippure 对 IPv6 出口没有数据(实测全部返回 0),
			// 机房 IP 的 0 分同样可疑。用 ipok caller 模式(走节点查出口,
			// 不受第三方 IP 配额限制)交叉验证,取 ipok 的分值。
			if ip2, err := queryIpokCaller(riskClient); err == nil && ip2 != nil {
				res.RiskScore = ip2.Risk
				res.RiskPct = fmt.Sprintf("%d%%", ip2.Risk)
				res.Emoji = emojiFor(ip2.Risk)
				if hasAny(ip2.Signals, "hosting") {
					res.IPAttr = "机房"
				} else {
					res.IPAttr = "住宅"
				}
				if hasAny(ip2.Signals, "proxy", "vpn", "tor") {
					res.IPSource = "代理"
				} else {
					res.IPSource = "原生"
				}
			} else {
				// 交叉验证失败: 不采信 0 分,标记未知
				res.RiskScore = -1
				res.RiskPct = "?"
				res.Emoji = "❓"
				res.Error = fmt.Sprintf("ippure 0分不可信且 ipok 交叉验证失败: %v", err)
			}
		}
	}

	// ---- 源2: ipok 兜底(仅当 ippure 失败时) ----
	if res.RiskScore < 0 && err1 != nil {
		ip2, _ := queryIpokCaller(riskClient) // 走节点 caller 模式
		if ip2 != nil {
			res.IP = ip2.Geo.IP
			res.Country = ip2.Geo.CountryCode
			res.RiskScore = ip2.Risk
			res.RiskPct = fmt.Sprintf("%d%%", ip2.Risk)
			res.Emoji = emojiFor(ip2.Risk)
			// 属性: 命中 hosting 信号 → 机房,否则住宅
			if hasAny(ip2.Signals, "hosting") {
				res.IPAttr = "机房"
			} else {
				res.IPAttr = "住宅"
			}
			// 来源: 命中 proxy/vpn/tor 信号 → 代理,否则原生
			if hasAny(ip2.Signals, "proxy", "vpn", "tor") {
				res.IPSource = "代理"
			} else {
				res.IPSource = "原生"
			}
		}
	}

	if res.RiskScore < 0 {
		res.Error = fmt.Sprintf("ippure: %v; ipok: 失败", err1)
	}
	return res
}
