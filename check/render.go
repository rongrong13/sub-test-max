package check

import (
	"fmt"
	"strings"

	"github.com/rongrong13/sub-test-max/check/unlock"
	"github.com/rongrong13/sub-test-max/config"
	proxyutils "github.com/rongrong13/sub-test-max/proxy"
)

// providerAbbr 服务名 → 节点名标注用的短缩写(参考原版 subs-check 命名风格)。
var providerAbbr = map[string]string{
	"Netflix":              "NF",
	"Netflix CDN":          "NF-CDN",
	"Disney+":              "D+",
	"Youtube Premium":      "YT",
	"Youtube CDN":          "YT-CDN",
	"OpenAI ChatGPT":       "GPT",
	"Anthropic Claude":     "CL",
	"Google Gemini":        "GM",
	"Microsoft Copilot":    "CP",
	"Spotify Registration": "SP",
	"Amazon Prime Video":   "PV",
	"Hulu":                 "HL",
	"Max":                  "MX",
	"TikTok":               "TT",
	"Steam":                "ST",
	"Reddit":               "RD",
	"Apple":                "AP",
	"Bing":                 "BG",
	"Dazn":                 "DZ",
	"X (formerly Twitter)": "X",
	"Instagram":            "IG",
	"Facebook":             "FB",
	"Youtube":              "YT",
}

// abbrOf 返回服务名缩写;未映射的服务取单词首字母(最多3个)。
func abbrOf(provider string) string {
	if v, ok := providerAbbr[provider]; ok {
		return v
	}
	var sb strings.Builder
	for _, word := range strings.Fields(provider) {
		if len(word) > 0 {
			sb.WriteString(strings.ToUpper(word[:1]))
		}
		if sb.Len() >= 3 {
			break
		}
	}
	if sb.Len() == 0 {
		if len(provider) > 3 {
			return strings.ToUpper(provider[:3])
		}
		return strings.ToUpper(provider)
	}
	return sb.String()
}

// unlockResultLike 便于测试注入的最小接口。
type unlockResultLike interface {
	GetName() string
	GetStatusText() string
	GetRegion() string
}

type wrappedUnlockResult struct{ name, statusText, region string }

func (w wrappedUnlockResult) GetName() string       { return w.name }
func (w wrappedUnlockResult) GetStatusText() string { return w.statusText }
func (w wrappedUnlockResult) GetRegion() string     { return w.region }

// formatStreamSummary 把解锁检测结果压成原版风格的短标注(只保留解锁成功的服务):
//
//	"YT-VN|GPT⁺-SG|GM-VN" — 平台缩写,GPT 完全解锁用 ⁺ 标记,区域大写后缀。
//
// 解锁失败(✗)的服务不显示。
func formatStreamSummary(results []unlockResultLike) string {
	parts := make([]string, 0, len(results))
	for _, item := range results {
		if item.GetStatusText() != "ok" {
			continue
		}
		abbr := abbrOf(item.GetName())
		// 与原版一致: GPT 完全解锁显示 GPT⁺
		if abbr == "GPT" {
			abbr += "⁺"
		}
		if region := strings.ToUpper(item.GetRegion()); region != "" {
			abbr += "-" + region
		}
		parts = append(parts, abbr)
	}
	return strings.Join(parts, "|")
}

// adapter 将 []unlock.Result 适配为 []unlockResultLike(纯函数,便于 render 复用)。
func adaptResults(res []unlock.Result) []unlockResultLike {
	out := make([]unlockResultLike, 0, len(res))
	for _, r := range res {
		out = append(out, wrappedUnlockResult{name: r.Name, statusText: r.StatusText, region: r.Region})
	}
	return out
}

// RenderName 根据 Result 的结构化字段构造展示名。
//
// 这是整个项目唯一的"节点名生成"出口,纯函数:
//   - 无 I/O,无 goroutine
//   - 不读写 proxy map 的 name 字段,不修改 Result
//   - 仅依赖传入的 Result 和 config.GlobalConfig
//
// 命名格式与原版 subs-check 一致(参考 subs-check2 历史输出):
//
//		{base}|{风险%}|{媒体解锁}|{订阅备注}
//		例: "🇭🇰HK_43|20%|YT-VN|GPT⁺-SG|GM-VN|edgetunnel"
//
//	  - base: 国旗+国家码_序号(rename-node 开启时)或机场原名
//	  - 风险%: ippure/ipok 风控值,如 20%(失败时省略)
//	  - 媒体解锁: 平台缩写+区域,如 YT-VN、GPT⁺-SG、GM-VN(仅解锁成功的服务)
//	  - 订阅备注: 订阅链接 #备注(如 #edgetunnel),追加在最后
//
// 注意: 不包含 IP 属性标注(与原版一致,保持名字紧凑)。
// 速度标签仅在测速开启(配置了 speed-test-url)且 includeSpeed=true
// (最终输出 all.yaml 时)才加入;过滤阶段调用传 false,名字不含速度。
func RenderName(r Result, includeSpeed bool) string {
	// 1. base 名字
	var base string
	if config.GlobalConfig.RenameNode {
		base = config.GlobalConfig.NodePrefix + proxyutils.Rename(r.Country)
	} else if r.Proxy != nil {
		if n, ok := r.Proxy["name"].(string); ok {
			base = strings.TrimSpace(n)
		}
	}

	// 2. 速度标签(仅测速开启且 includeSpeed 时)
	var speedTag string
	if includeSpeed && config.GlobalConfig.SpeedTestUrl != "" && r.Speed > 0 {
		speedTag = formatSpeedTag(r.Speed)
	}

	// 3. 风险度百分比(仅当有 IP 风险结果且非失败)
	var riskPct string
	if r.IPRisk != nil && r.IPRisk.RiskPct != "" && r.IPRisk.RiskPct != "?" {
		riskPct = r.IPRisk.RiskPct
	}

	// 4. 流媒体解锁摘要(只显示解锁成功的服务)
	streamSummary := formatStreamSummary(adaptResults(r.Media))

	// 5. sub_tag(订阅 #备注)追加到最后
	var subTag string
	if r.Proxy != nil {
		if t, ok := r.Proxy["sub_tag"].(string); ok && t != "" {
			subTag = t
		}
	}

	// 用 | 连接主体部分(空项跳过)
	parts := make([]string, 0, 4)
	for _, p := range []string{base, speedTag, riskPct, streamSummary} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	var out string
	if len(parts) > 0 {
		out = strings.Join(parts, "|")
	}

	if subTag != "" {
		if out != "" {
			out += "|" + subTag
		} else {
			out = subTag
		}
	}
	return out
}

// formatSpeedTag 把测速结果(KB/s)格式化为展示字符串。
//
//	<1024 → "NKB/s"
//	>=1024 → "X.XMB/s"
func formatSpeedTag(speed int) string {
	if speed < 1024 {
		return fmt.Sprintf("%dKB/s", speed)
	}
	return fmt.Sprintf("%.1fMB/s", float64(speed)/1024)
}
