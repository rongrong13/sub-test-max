package check

import (
	"regexp"
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
	withConfig(t, config.Config{RenameNode: false, Platforms: []string{"openai", "netflix"}}, func() {
		r := Result{Proxy: map[string]any{"name": "🇭🇰香港01"}}
		if got, want := RenderName(r, false), "🇭🇰香港01"; got != want {
			t.Errorf("RenderName() = %q, want %q", got, want)
		}
	})
}

func TestRenderName_RenameOff_PreservesOriginalWithPipes(t *testing.T) {
	withConfig(t, config.Config{RenameNode: false, Platforms: []string{}}, func() {
		r := Result{Proxy: map[string]any{"name": "🇺🇸美国01-0.1倍 | 电信联通移动推荐"}}
		if got, want := RenderName(r, false), "🇺🇸美国01-0.1倍 | 电信联通移动推荐"; got != want {
			t.Errorf("RenderName() = %q, want %q", got, want)
		}
	})
}

func TestRenderName_OriginalStyleMediaTags(t *testing.T) {
	// 原版风格: base|GPT⁺-HK|NF-HK|D+-HK
	withConfig(t, config.Config{RenameNode: false, Platforms: []string{"openai", "netflix", "disney"}}, func() {
		r := Result{
			Proxy: map[string]any{"name": "🇭🇰香港01"},
			Media: []unlock.Result{
				okResult("OpenAI ChatGPT", "hk"),
				okResult("Netflix", "hk"),
				okResult("Disney+", "hk"),
			},
		}
		if got, want := RenderName(r, false), "🇭🇰香港01|GPT⁺-HK|NF-HK|D+-HK"; got != want {
			t.Errorf("RenderName() = %q, want %q", got, want)
		}
	})
}

func TestRenderName_OnlyOkServicesShown(t *testing.T) {
	withConfig(t, config.Config{RenameNode: false, Platforms: []string{"openai", "netflix", "gemini"}}, func() {
		r := Result{
			Proxy: map[string]any{"name": "n"},
			Media: []unlock.Result{
				{Name: "OpenAI ChatGPT", Status: 3, StatusText: "no"},
				okResult("Netflix", "us"),
				{Name: "Google Gemini", Status: -1, StatusText: "network_error"},
			},
		}
		if got, want := RenderName(r, false), "n|NF-US"; got != want {
			t.Errorf("RenderName() = %q, want %q", got, want)
		}
	})
}

func TestRenderName_PlatformsOrderMatters(t *testing.T) {
	withConfig(t, config.Config{RenameNode: false, Platforms: []string{"netflix", "openai"}}, func() {
		r := Result{
			Proxy: map[string]any{"name": "n"},
			Media: []unlock.Result{
				okResult("Netflix", "hk"),
				okResult("OpenAI ChatGPT", "hk"),
			},
		}
		if got, want := RenderName(r, false), "n|NF-HK|GPT⁺-HK"; got != want {
			t.Errorf("RenderName() = %q, want %q", got, want)
		}
	})
}

func TestRenderName_SpeedTag_WithSpeedEnabled(t *testing.T) {
	// 测速开启且 includeSpeed=true → base|速度|...
	withConfig(t, config.Config{RenameNode: false, SpeedTestUrl: "https://example.com/file", Platforms: []string{}}, func() {
		r := Result{Proxy: map[string]any{"name": "n"}, Speed: 5120}
		if got, want := RenderName(r, true), "n|5.0MB/s"; got != want {
			t.Errorf("RenderName() = %q, want %q", got, want)
		}
		// 过滤阶段 includeSpeed=false → 无速度
		if got, want := RenderName(r, false), "n"; got != want {
			t.Errorf("RenderName() = %q, want %q", got, want)
		}
	})
}

func TestRenderName_NoSpeedTag_WhenSpeedDisabled(t *testing.T) {
	// 测速未开启(URL为空) → 即使 includeSpeed=true 也不加速度
	withConfig(t, config.Config{RenameNode: false, SpeedTestUrl: "", Platforms: []string{}}, func() {
		r := Result{Proxy: map[string]any{"name": "n"}, Speed: 5120}
		if got, want := RenderName(r, true), "n"; got != want {
			t.Errorf("RenderName() = %q, want %q", got, want)
		}
	})
}

func TestRenderName_RiskPct(t *testing.T) {
	withConfig(t, config.Config{RenameNode: false, Platforms: []string{}}, func() {
		r := Result{Proxy: map[string]any{"name": "n"}, IPRisk: &iprisk.Result{RiskPct: "20%", Emoji: "🟢"}}
		if got, want := RenderName(r, false), "n|20%"; got != want {
			t.Errorf("RenderName() = %q, want %q", got, want)
		}
	})
}

func TestRenderName_RiskUnknownSkipped(t *testing.T) {
	withConfig(t, config.Config{RenameNode: false, Platforms: []string{}}, func() {
		r := Result{Proxy: map[string]any{"name": "n"}, IPRisk: &iprisk.Result{RiskPct: "?", Emoji: "❓"}}
		if got, want := RenderName(r, false), "n"; got != want {
			t.Errorf("RenderName() = %q, want %q", got, want)
		}
	})
}

func TestRenderName_RenameNode_CountrySeq(t *testing.T) {
	proxyutils.ResetRenameCounter()
	withConfig(t, config.Config{RenameNode: true, Platforms: []string{}}, func() {
		r := Result{Proxy: map[string]any{"name": "orig"}, Country: "HK"}
		if got, want := RenderName(r, false), "🇭🇰HK_1"; got != want {
			t.Errorf("RenderName() = %q, want %q", got, want)
		}
	})
}

func TestRenderName_SubTag(t *testing.T) {
	// 订阅 #备注 追加在最后
	withConfig(t, config.Config{RenameNode: false, Platforms: []string{"openai"}}, func() {
		r := Result{
			Proxy: map[string]any{"name": "n", "sub_tag": "edgetunnel"},
			Media: []unlock.Result{okResult("OpenAI ChatGPT", "sg")},
		}
		if got, want := RenderName(r, false), "n|GPT⁺-SG|edgetunnel"; got != want {
			t.Errorf("RenderName() = %q, want %q", got, want)
		}
	})
}

func TestRenderName_NewPlatformAbbr(t *testing.T) {
	// 新增平台缩写: Instagram→IG, Facebook→FB, Reddit→RD, X→X
	withConfig(t, config.Config{RenameNode: false, Platforms: []string{"reddit", "instagram", "facebook", "x"}}, func() {
		r := Result{
			Proxy: map[string]any{"name": "n"},
			Media: []unlock.Result{
				okResult("Reddit", ""),
				okResult("Instagram", "us"),
				okResult("Facebook", "us"),
				okResult("X (formerly Twitter)", "us"),
			},
		}
		if got, want := RenderName(r, false), "n|RD|IG-US|FB-US|X-US"; got != want {
			t.Errorf("RenderName() = %q, want %q", got, want)
		}
	})
}

func TestRenderName_NoIPAttrTag(t *testing.T) {
	// 原版风格不带【emoji 属性|来源】标注
	withConfig(t, config.Config{RenameNode: false, Platforms: []string{"gemini"}}, func() {
		r := Result{
			Proxy:  map[string]any{"name": "n"},
			IPRisk: &iprisk.Result{RiskPct: "35%", Emoji: "🟡", IPAttr: "机房", IPSource: "未知"},
			Media:  []unlock.Result{okResult("Google Gemini", "vn")},
		}
		if got, want := RenderName(r, false), "n|35%|GM-VN"; got != want {
			t.Errorf("RenderName() = %q, want %q", got, want)
		}
	})
}

func TestRenderName_FullExample(t *testing.T) {
	// 完整示例: 🇭🇰HK_43|1.2MB/s|20%|YT-VN|GPT⁺-SG|GM-VN|edgetunnel
	proxyutils.ResetRenameCounter()
	withConfig(t, config.Config{
		RenameNode:   true,
		SpeedTestUrl: "https://example.com/file",
		Platforms:    []string{"youtube", "openai", "gemini"},
	}, func() {
		r := Result{
			Proxy:   map[string]any{"name": "orig", "sub_tag": "edgetunnel"},
			Country: "HK",
			Speed:   1228,
			IPRisk:  &iprisk.Result{RiskPct: "20%", Emoji: "🟢"},
			Media: []unlock.Result{
				okResult("Youtube Premium", "vn"),
				okResult("OpenAI ChatGPT", "sg"),
				okResult("Google Gemini", "vn"),
			},
		}
		want := "🇭🇰HK_1|1.2MB/s|20%|YT-VN|GPT⁺-SG|GM-VN|edgetunnel"
		if got := RenderName(r, true); got != want {
			t.Errorf("RenderName() = %q, want %q", got, want)
		}
	})
}

func TestRenderName_FullExample_FilterStage(t *testing.T) {
	// 过滤阶段(无速度,序号按调用次数递增,跳号不影响去重)
	proxyutils.ResetRenameCounter()
	withConfig(t, config.Config{
		RenameNode:   true,
		SpeedTestUrl: "https://example.com/file",
		Platforms:    []string{"youtube", "openai", "gemini"},
	}, func() {
		r := Result{
			Proxy:   map[string]any{"name": "orig", "sub_tag": "edgetunnel"},
			Country: "HK",
			Speed:   1228,
			IPRisk:  &iprisk.Result{RiskPct: "20%", Emoji: "🟢"},
			Media: []unlock.Result{
				okResult("Youtube Premium", "vn"),
				okResult("OpenAI ChatGPT", "sg"),
				okResult("Google Gemini", "vn"),
			},
		}
		want := "🇭🇰HK_1|20%|YT-VN|GPT⁺-SG|GM-VN|edgetunnel"
		if got := RenderName(r, false); got != want {
			t.Errorf("RenderName() = %q, want %q", got, want)
		}
	})
}

func TestFilterRegexMatchesNewFormat(t *testing.T) {
	// 新格式下 filter 正则仍可用
	name := "🇭🇰HK_43|20%|YT-VN|GPT⁺-SG|GM-VN|edgetunnel"
	if !regexp.MustCompile(`|[0-2]?[0-9]%`).MatchString(name) {
		t.Errorf("risk filter should match %q", name)
	}
	if !regexp.MustCompile(`GPT⁺|GPT`).MatchString(name) {
		t.Errorf("GPT filter should match %q", name)
	}
	if !regexp.MustCompile(`YT-`).MatchString(name) {
		t.Errorf("YT filter should match %q", name)
	}
}
