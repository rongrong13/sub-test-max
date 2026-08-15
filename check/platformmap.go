package check

import "strings"

// platformProviderMap 把 subs-check 原有的 platform 名映射到 MediaUnlockTest 的服务名。
// 这样旧的配置文件(platforms: [netflix, disney, openai, ...])无需改动即可继续使用,
// 同时可直接写 mediatest 的服务名(如 "Amazon Prime Video")来启用更多服务。
// 映射映射到具体的单服务测试; "iprisk" 是无检测标记,仅触发 IP 风险检测。
var platformProviderMap = map[string]string{
	"openai":  "OpenAI ChatGPT",
	"youtube": "Youtube Premium",
	"netflix": "Netflix",
	"disney":  "Disney+",
	"gemini":  "Google Gemini",
	"claude":  "Anthropic Claude",
	"spotify":   "Spotify Registration",
	"tiktok":    "TikTok",
	"reddit":    "Reddit",
	"x":         "X (formerly Twitter)",
	"instagram": "Instagram",
	"facebook":  "Facebook",
}

// resolvePlatformKey 返回给定 platform 配置项对应的解析结果:
//   - 若是平台别名(如 "netflix"),返回映射出的 mediatest 服务名
//   - 若是 iprisk,返回特殊标记
//   - 其它字符串视为直接引用 mediatest 服务名,原样返回
func resolvePlatformKey(p string) (provider string, isIPRisk bool, isKnown bool) {
	if p == "iprisk" {
		return "", true, true
	}
	if v, ok := platformProviderMap[strings.ToLower(p)]; ok {
		return v, false, true
	}
	return p, false, false
}

// resolveMediaProviders 收集需要执行流媒体解锁检测的服务列表(按 platforms 顺序),
// 并返回是否需要 IP 风险检测。iprisk 只触发 IP 风险检测,不计入服务列表。
func resolveMediaProviders(platforms []string) (providers []string, doIPRisk bool) {
	seen := map[string]bool{}
	for _, p := range platforms {
		provider, isIPRisk, _ := resolvePlatformKey(p)
		if isIPRisk {
			doIPRisk = true
			continue
		}
		if provider != "" && !seen[provider] {
			seen[provider] = true
			providers = append(providers, provider)
		}
	}
	return
}
