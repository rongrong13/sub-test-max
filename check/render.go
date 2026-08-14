package check

import (
	"fmt"
	"strings"

	"github.com/rongrong13/sub-test-max/check/unlock"
	"github.com/rongrong13/sub-test-max/config"
	proxyutils "github.com/rongrong13/sub-test-max/proxy"
)

// providerAbbr 服务名 → 节点名标注用的短缩写(未映射的服务使用原名前几个字符)。
// 与 IP-Stream-Checker 中 stream_tester/media_unlock.py 的 PROVIDER_ABBR 一致。
var providerAbbr = map[string]string{
	"Netflix":             "NF",
	"Netflix CDN":         "NF-CDN",
	"Disney+":            "D+",
	"Youtube Premium":     "YT",
	"Youtube CDN":         "YT-CDN",
	"OpenAI ChatGPT":      "GPT",
	"Anthropic Claude":    "Claude",
	"Google Gemini":       "GM",
	"Microsoft Copilot":   "Copilot",
	"Spotify Registration": "Spotify",
	"Amazon Prime Video":  "Prime",
	"Hulu":                "Hulu",
	"HBO Max":             "Max",
	"TikTok":              "TikTok",
	"Steam":               "Steam",
	"Reddit":              "Reddit",
	"Apple":               "Apple",
	"Bing":                "Bing",
	"Dazn":                "Dazn",
}

func abbrOf(provider string) string {
	if v, ok := providerAbbr[provider]; ok {
		return v
	}
	if len(provider) > 6 {
		return provider[:6]
	}
	return provider
}

// statusSymbol 根据状态返回标注符号与地区后缀(仅 "ok" 与 "restricted" 带地区的完整标注)。
func statusSymbol(r unlockResultLike) string {
	return statusSymbolCore(r)
}

// unlockResultLike 便于测试注入的最小接口。
type unlockResultLike interface {
	GetName() string
	GetStatusText() string
	GetRegion() string
}

type wrappedUnlockResult struct{ name, statusText, region string }

func (w wrappedUnlockResult) GetName() string        { return w.name }
func (w wrappedUnlockResult) GetStatusText() string  { return w.statusText }
func (w wrappedUnlockResult) GetRegion() string      { return w.region }

func statusSymbolCore(r unlockResultLike) string {
	status := r.GetStatusText()
	region := r.GetRegion()
	switch status {
	case "ok":
		if region != "" {
			return fmt.Sprintf("✓(%s)", region)
		}
		return "✓"
	case "restricted":
		if region != "" {
			return fmt.Sprintf("⚠(%s)", region)
		}
		return "⚠"
	default:
		return "✗"
	}
}

// formatStreamSummary 把解锁检测结果压成节点名标注(只保留解锁成功的服务),与
// IP-Stream-Checker 的 format_summary 一致:
//   "GM✓(sg)·NF✓(us)·GPT✓" — 解锁失败(✗)的服务不显示。
func formatStreamSummary(results []unlockResultLike) string {
	parts := make([]string, 0, len(results))
	for _, item := range results {
		if item.GetStatusText() != "ok" {
			continue
		}
		parts = append(parts, abbrOf(item.GetName())+statusSymbol(item))
	}
	return strings.Join(parts, "·")
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
// 采用了 IP-Stream-Checker 的命名格式:
//   {base}[·速度][·风险%][·流媒体解锁摘要]{IP标注【emoji 属性|来源】}
//   例: "🇭🇰 HK-01·61%·GM✓(sg)·NF✓(us)·GPT✓【🟢 住宅|原生】"
//
// includeSpeed 为 true 时追加速度标签,只在最终输出 all.yaml 时用。
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

	// 2. 速度标签(仅 includeSpeed)
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

	// 5. IP 风险完整标注【emoji 机房|代理】
	ipTag := ""
	if r.IPRisk != nil {
		ipTag = r.IPRisk.FullString()
	}

	// 6. sub_tag 追加到最后
	var subTag string
	if r.Proxy != nil {
		if t, ok := r.Proxy["sub_tag"].(string); ok && t != "" {
			subTag = t
		}
	}

	// 用 · 连接主体部分(空项跳过)
	parts := make([]string, 0, 4)
	for _, p := range []string{base, speedTag, riskPct, streamSummary} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	var out string
	if len(parts) > 0 {
		out = strings.Join(parts, "·")
	}

	// IP 标注直接拼接在末尾(参考 IP-Stream-Checker 的做法)
	out += ipTag

	if subTag != "" {
		if out != "" {
			out += "·" + subTag
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
