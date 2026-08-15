package check

import (
	"fmt"
	"log/slog"
	"regexp"

	"github.com/rongrong13/sub-test-max/config"
)

// CompileFilterPatterns compiles the configured filter regex list.
// Invalid patterns are dropped with a warning; returns an empty slice
// when filtering is disabled or all patterns failed to compile.
func CompileFilterPatterns() []*regexp.Regexp {
	if len(config.GlobalConfig.Filter) == 0 {
		return nil
	}
	var patterns []*regexp.Regexp
	for _, pattern := range config.GlobalConfig.Filter {
		re, err := regexp.Compile(pattern)
		if err != nil {
			slog.Warn(fmt.Sprintf("过滤正则表达式编译失败，已跳过: %s, 错误: %v", pattern, err))
			continue
		}
		patterns = append(patterns, re)
	}
	if len(patterns) == 0 && len(config.GlobalConfig.Filter) > 0 {
		slog.Warn("所有过滤正则表达式编译失败，跳过过滤")
	}
	return patterns
}

// CompileExcludePatterns compiles the configured exclude-filter regex list
// (原版 subs-check 同款: 命中任一排除正则的节点直接丢弃,
// 未带标签/未命中的节点保留)。
func CompileExcludePatterns() []*regexp.Regexp {
	if len(config.GlobalConfig.ExcludeFilter) == 0 {
		return nil
	}
	var patterns []*regexp.Regexp
	for _, pattern := range config.GlobalConfig.ExcludeFilter {
		re, err := regexp.Compile(pattern)
		if err != nil {
			slog.Warn(fmt.Sprintf("排除过滤正则表达式编译失败，已跳过: %s, 错误: %v", pattern, err))
			continue
		}
		patterns = append(patterns, re)
	}
	return patterns
}

// MatchesFilter reports whether r's rendered name (without speed tag)
// passes the include filter AND the exclude filter.
//   - include: 空 pattern 视为全部通过;否则需匹配任一 pattern
//   - exclude: 命中任一 exclude 正则即被排除(先排除,再包含)
func MatchesFilter(r Result, patterns, excludes []*regexp.Regexp) bool {
	if r.Proxy == nil {
		return false
	}
	name := RenderName(r, false)
	// 排除规则优先: 命中任一排除正则 → 丢弃
	for _, re := range excludes {
		if re.MatchString(name) {
			return false
		}
	}
	// 包含规则: 空列表视为全部通过
	if len(patterns) == 0 {
		return true
	}
	for _, re := range patterns {
		if re.MatchString(name) {
			return true
		}
	}
	return false
}

// FilterResults 根据配置的正则表达式过滤节点。
//
// 只有渲染后的展示名(不含速度标签)匹配任一正则的节点才会被保留。
// 这里用 RenderName(r, false) 而不是 r.Proxy["name"] 是为了让 filter 能看到
// 国家+媒体标签的完整视图,同时保持 proxy["name"] 不被修改。
func FilterResults(results []Result) []Result {
	patterns := CompileFilterPatterns()
	excludes := CompileExcludePatterns()
	if len(patterns) == 0 && len(excludes) == 0 {
		return results
	}

	slog.Info(fmt.Sprintf("应用节点过滤规则，保留规则 %d 条, 排除规则 %d 条", len(patterns), len(excludes)))

	var filtered []Result
	for _, r := range results {
		if MatchesFilter(r, patterns, excludes) {
			filtered = append(filtered, r)
		}
	}

	slog.Info(fmt.Sprintf("过滤后节点数量: %d (过滤前: %d)", len(filtered), len(results)))
	return filtered
}
