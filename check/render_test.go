package check

import (
	"testing"

	"github.com/rongrong13/sub-test-max/check/iprisk"
	"github.com/rongrong13/sub-test-max/check/unlock"
	"github.com/rongrong13/sub-test-max/config"
	proxyutils "github.com/rongrong13/sub-test-max/proxy"
)

// withConfig 临时替换 config.GlobalConfig 的内容,测试结束后还原。
func withConfig(t *testing.T, cfg config.Config, fn func()) {
	t.Helper()
	old := *config.GlobalConfig
	*config.GlobalConfig = cfg
	defer func() { *config.GlobalConfig = old }()
	fn()
}

// okResult 构造一个"解锁成功"的 mediatest 结果。
func okResult(name, region string) unlock.Result {
	return unlock.Result{Name: name, Status: 1, StatusText: "ok", Region: region, Ok: true}
}

func TestRenderName_RenameOff_NoTags(t *testing.T) {
	withConfig(t, config.Config{
		RenameNode: false,
		Platforms:  []string{"openai", "netflix"},
	}, func() {
		r := Result{
			Proxy: map[string]any{"name": "🇭🇰香港01"},
		}
		got := RenderName(r, false)
		want := "🇭🇰香港01"
		if got != want {
			t.Errorf("RenderName() = %q, want %q", got, want)
		}
	})
}

func TestRenderName_RenameOff_PreservesOriginalWithPipes(t *testing.T) {
	// 机场原名里就带 | 的情况,不能破坏它
	withConfig(t, config.Config{
		RenameNode: false,
		Platforms:  []string{},
	}, func() {
		r := Result{
			Proxy: map[string]any{"name": "🇺🇸美国01-0.1倍 | 电信联通移动推荐"},
		}
		got := RenderName(r, false)
		want := "🇺🇸美国01-0.1倍 | 电信联通移动推荐"
		if got != want {
			t.Errorf("RenderName() = %q, want %q", got, want)
		}
	})
}

func TestRenderName_RenameOff_WithMediaTags(t *testing.T) {
	withConfig(t, config.Config{
		RenameNode: false,
		Platforms:  []string{"openai", "netflix", "disney"},
	}, func() {
		r := Result{
			Proxy: map[string]any{"name": "🇭🇰香港01"},
			Media: []unlock.Result{
				okResult("OpenAI ChatGPT", "hk"),
				okResult("Netflix", "hk"),
				okResult("Disney+", "hk"),
			},
		}
		got := RenderName(r, false)
		want := "🇭🇰香港01·GPT✓(hk)·NF✓(hk)·D+✓(hk)"
		if got != want {
			t.Errorf("RenderName() = %q, want %q", got, want)
		}
	})
}

func TestRenderName_OnlyOkServicesShown(t *testing.T) {
	// 未解锁(status_text != ok)的服务不显示在节点名里
	withConfig(t, config.Config{
		RenameNode: false,
		Platforms:  []string{"openai", "netflix", "gemini"},
	}, func() {
		r := Result{
			Proxy: map[string]any{"name": "n"},
			Media: []unlock.Result{
				{Name: "OpenAI ChatGPT", Status: 3, StatusText: "no"},
				okResult("Netflix", "us"),
				{Name: "Google Gemini", Status: -1, StatusText: "network_error"},
			},
		}
		got := RenderName(r, false)
		want := "n·NF✓(us)"
		if got != want {
			t.Errorf("RenderName() = %q, want %q", got, want)
		}
	})
}

func TestRenderName_PlatformsOrderMatters(t *testing.T) {
	// 流媒体摘要顺序遵循 r.Media(即 resolveMediaProviders 按 config.Platforms 展开的顺序)
	withConfig(t, config.Config{
		RenameNode: false,
		Platforms:  []string{"netflix", "openai"},
	}, func() {
		r := Result{
			Proxy: map[string]any{"name": "n"},
			Media: []unlock.Result{
				okResult("Netflix", "hk"),
				okResult("OpenAI ChatGPT", "hk"),
			},
		}
		got := RenderName(r, false)
		want := "n·NF✓(hk)·GPT✓(hk)"
		if got != want {
			t.Errorf("RenderName() = %q, want %q", got, want)
		}
	})
}

func TestRenderName_IncludeSpeedTrue(t *testing.T) {
	withConfig(t, config.Config{
		RenameNode:   false,
		SpeedTestUrl: "https://example.com/file",
		Platforms:    []string{},
	}, func() {
		r := Result{
			Proxy: map[string]any{"name": "n"},
			Speed: 5120, // 5.0 MB/s
		}
		got := RenderName(r, true)
		want := "n·5.0MB/s"
		if got != want {
			t.Errorf("RenderName() = %q, want %q", got, want)
		}
	})
}

func TestRenderName_IncludeSpeedFalse_NoSpeedTag(t *testing.T) {
	withConfig(t, config.Config{
		RenameNode:   false,
		SpeedTestUrl: "https://example.com/file",
		Platforms:    []string{},
	}, func() {
		r := Result{
			Proxy: map[string]any{"name": "n"},
			Speed: 5120,
		}
		got := RenderName(r, false) // filter 阶段调用时传 false
		want := "n"
		if got != want {
			t.Errorf("RenderName() = %q, want %q", got, want)
		}
	})
}

func TestRenderName_SpeedTagFormat_KB(t *testing.T) {
	withConfig(t, config.Config{
		RenameNode:   false,
		SpeedTestUrl: "https://example.com/file",
		Platforms:    []string{},
	}, func() {
		r := Result{
			Proxy: map[string]any{"name": "n"},
			Speed: 512, // < 1024,展示 KB/s
		}
		got := RenderName(r, true)
		want := "n·512KB/s"
		if got != want {
			t.Errorf("RenderName() = %q, want %q", got, want)
		}
	})
}

func TestRenderName_SpeedZero_NoSpeedTag(t *testing.T) {
	// Speed=0 表示未测速(ForceClose 场景),即使 includeSpeed=true 也不加标签
	withConfig(t, config.Config{
		RenameNode:   false,
		SpeedTestUrl: "https://example.com/file",
		Platforms:    []string{},
	}, func() {
		r := Result{
			Proxy: map[string]any{"name": "n"},
			Speed: 0,
		}
		got := RenderName(r, true)
		want := "n"
		if got != want {
			t.Errorf("RenderName() = %q, want %q", got, want)
		}
	})
}

func TestRenderName_SpeedBeforeMediaTags(t *testing.T) {
	// 锁定标签顺序: base · speed · risk% · media-tags · IP标注 · sub_tag
	withConfig(t, config.Config{
		RenameNode:   false,
		SpeedTestUrl: "https://example.com/file",
		Platforms:    []string{"openai", "netflix"},
	}, func() {
		r := Result{
			Proxy: map[string]any{"name": "n", "sub_tag": "tag"},
			Speed: 5120, // 5.0MB/s
			Media: []unlock.Result{
				okResult("OpenAI ChatGPT", "hk"),
				okResult("Netflix", "hk"),
			},
		}
		got := RenderName(r, true)
		want := "n·5.0MB/s·GPT✓(hk)·NF✓(hk)·tag"
		if got != want {
			t.Errorf("RenderName() = %q, want %q", got, want)
		}
	})
}

func TestRenderName_SubTagAppendedLast(t *testing.T) {
	withConfig(t, config.Config{
		RenameNode: false,
		Platforms:  []string{"disney"},
	}, func() {
		r := Result{
			Proxy: map[string]any{"name": "n", "sub_tag": "my-sub"},
			Media: []unlock.Result{
				okResult("Disney+", ""),
			},
		}
		got := RenderName(r, false)
		want := "n·D+✓·my-sub"
		if got != want {
			t.Errorf("RenderName() = %q, want %q", got, want)
		}
	})
}

func TestRenderName_IPRiskTag(t *testing.T) {
	withConfig(t, config.Config{
		RenameNode: false,
		Platforms:  []string{"iprisk"},
	}, func() {
		r := Result{
			Proxy: map[string]any{"name": "n"},
			IPRisk: &iprisk.Result{
				RiskScore: 5, RiskPct: "5%", Emoji: "🟢",
				IPAttr: "住宅", IPSource: "原生",
			},
		}
		got := RenderName(r, false)
		want := "n·5%【🟢 住宅|原生】"
		if got != want {
			t.Errorf("RenderName() = %q, want %q", got, want)
		}
	})
}

func TestRenderName_RiskPlusStream(t *testing.T) {
	// 组装 IP 风险 + 流媒体解锁摘要 + IP 标注,完整复现 IP-Stream-Checker 命名
	withConfig(t, config.Config{
		RenameNode: false,
		Platforms:  []string{"gemini", "netflix", "openai", "iprisk"},
	}, func() {
		r := Result{
			Proxy: map[string]any{"name": "🇭🇰 HK-01"},
			IPRisk: &iprisk.Result{
				RiskScore: 61, RiskPct: "61%", Emoji: "🟠",
				IPAttr: "机房", IPSource: "代理",
			},
			Media: []unlock.Result{
				okResult("Google Gemini", "sg"),
				okResult("Netflix", "us"),
				okResult("OpenAI ChatGPT", ""),
			},
		}
		got := RenderName(r, false)
		want := "🇭🇰 HK-01·61%·GM✓(sg)·NF✓(us)·GPT✓【🟠 机房|代理】"
		if got != want {
			t.Errorf("RenderName() = %q, want %q", got, want)
		}
	})
}

func TestRenderName_RenameOnWithCountry(t *testing.T) {
	proxyutils.ResetRenameCounter()
	withConfig(t, config.Config{
		RenameNode: true,
		NodePrefix: "PREFIX-",
		Platforms:  []string{},
	}, func() {
		r := Result{
			Proxy:   map[string]any{"name": "original"},
			Country: "HK",
		}
		got := RenderName(r, false)
		if got == "original" {
			t.Errorf("RenderName() should not use original name when RenameNode=true, got %q", got)
		}
		if len(got) < len("PREFIX-") || got[:len("PREFIX-")] != "PREFIX-" {
			t.Errorf("RenderName() should start with prefix, got %q", got)
		}
		if !stringContains(got, "HK") {
			t.Errorf("RenderName() should contain country code HK, got %q", got)
		}
	})
}

func TestRenderName_RenameOnButEmptyCountry_UsesOtherFallback(t *testing.T) {
	// 重命名开启但 Country 为空(Phase 2 查询失败),应走 ❓Other 兜底
	proxyutils.ResetRenameCounter()
	withConfig(t, config.Config{
		RenameNode: true,
		NodePrefix: "PREFIX-",
		Platforms:  []string{},
	}, func() {
		r := Result{
			Proxy:   map[string]any{"name": "🇹🇼原名|745KB/s|YT-TW"},
			Country: "",
		}
		got := RenderName(r, false)
		if got == "🇹🇼原名|745KB/s|YT-TW" {
			t.Errorf("RenderName() should not preserve polluted original name when RenameNode=true, got %q", got)
		}
		if len(got) < len("PREFIX-") || got[:len("PREFIX-")] != "PREFIX-" {
			t.Errorf("RenderName() should start with prefix, got %q", got)
		}
		if !stringContains(got, "Other") {
			t.Errorf("RenderName() should fall back to Other when Country is empty, got %q", got)
		}
	})
}

// 辅助函数
func stringContains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
