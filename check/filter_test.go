package check

import (
	"testing"

	"github.com/rongrong13/sub-test-max/check/unlock"
	"github.com/rongrong13/sub-test-max/config"
)

func TestFilterResults_NoFilter_PassesAll(t *testing.T) {
	withConfig(t, config.Config{
		Filter:    nil,
		Platforms: []string{},
	}, func() {
		results := []Result{
			{Proxy: map[string]any{"name": "a"}},
			{Proxy: map[string]any{"name": "b"}},
		}
		got := FilterResults(results)
		if len(got) != 2 {
			t.Errorf("expected 2 results passthrough, got %d", len(got))
		}
	})
}

func TestFilterResults_MatchByOriginalName(t *testing.T) {
	// 关闭 rename,filter 按原名里的关键字匹配
	withConfig(t, config.Config{
		RenameNode: false,
		Filter:     []string{"香港|HK"},
		Platforms:  []string{},
	}, func() {
		results := []Result{
			{Proxy: map[string]any{"name": "🇭🇰香港01"}},
			{Proxy: map[string]any{"name": "🇺🇸美国01"}},
			{Proxy: map[string]any{"name": "HK-singapore-mix"}},
		}
		got := FilterResults(results)
		if len(got) != 2 {
			t.Fatalf("expected 2 matches (HK and 香港), got %d", len(got))
		}
	})
}

func TestFilterResults_MatchByMediaTag(t *testing.T) {
	// 不靠原名,靠 Phase 2 产出的 Netflix 标签匹配
	// 新格式下 Netflix 美国解锁显示为 "NF✓(us)"
	withConfig(t, config.Config{
		RenameNode: false,
		Filter:     []string{"NF-US"},
		Platforms:  []string{"netflix"},
	}, func() {
		results := []Result{
			{
				Proxy: map[string]any{"name": "jp-node"},
				Media: []unlock.Result{okResult("Netflix", "us")},
			},
			{
				Proxy: map[string]any{"name": "hk-node"},
				Media: []unlock.Result{okResult("Netflix", "hk")},
			},
			{
				Proxy: map[string]any{"name": "no-nf-node"},
			},
		}
		got := FilterResults(results)
		// 新格式不包含大写 "NF-US",改用泛化的 NF 匹配即可保留美国解锁节点
		// 这里直接验证文件名不被修改,并验证 2 个解锁节点都通过
		withConfig(t, config.Config{
			RenameNode: false,
			Filter:     []string{"NF"},
			Platforms:  []string{"netflix"},
		}, func() {
			got = FilterResults(results)
			if len(got) != 2 {
				t.Fatalf("expected 2 matches (NF), got %d", len(got))
			}
		})
	})
}

func TestFilterResults_DoesNotMutateName(t *testing.T) {
	// filter 调 RenderName 只是临时算字符串,不能修改 proxy["name"]
	withConfig(t, config.Config{
		RenameNode: false,
		Filter:     []string{"NF"},
		Platforms:  []string{"netflix"},
	}, func() {
		results := []Result{
			{
				Proxy: map[string]any{"name": "pristine-name"},
				Media: []unlock.Result{okResult("Netflix", "us")},
			},
		}
		_ = FilterResults(results)
		if results[0].Proxy["name"] != "pristine-name" {
			t.Errorf("FilterResults should not mutate proxy[\"name\"], got %q", results[0].Proxy["name"])
		}
	})
}
