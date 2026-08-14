package utils

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rongrong13/sub-test-max/config"
)

// WarpUrl 对订阅 URL 做预处理:先替换时间占位符,再对 raw.githubusercontent.com
// 链接套用 github 代理前缀。供 proxy 拉取订阅时使用。
func WarpUrl(url string) string {
	url = formatTimePlaceholders(url, time.Now())

	// 如果url中以https://raw.githubusercontent.com开头，那么就使用github代理
	if strings.HasPrefix(url, "https://raw.githubusercontent.com") {
		return config.GlobalConfig.GithubProxy + url
	}
	return url
}

// 动态时间占位符
// 支持在链接中使用时间占位符，会自动替换成当前日期/时间:
// - {Y}   - 四位年份 (2023)
// - {m}   - 两位月份 (01-12)
// - {d}   - 两位日期 (01-31)
// - {Ymd} - 组合日期 (20230131)
// 所有占位符均支持可选的天偏移后缀 ±N(单位:天),如 {Ymd-1} 表示昨天。
var timePlaceholderRe = regexp.MustCompile("\\{(Ymd|Y_m_d|Y-m-d|Y|m|d)([+-]\\d+)?\\}")

var timePlaceholderLayouts = map[string]string{
	"Y":     "2006",
	"m":     "01",
	"d":     "02",
	"Ymd":   "20060102",
	"Y_m_d": "2006_01_02",
	"Y-m-d": "2006-01-02",
}

func formatTimePlaceholders(url string, t time.Time) string {
	return timePlaceholderRe.ReplaceAllStringFunc(url, func(s string) string {
		m := timePlaceholderRe.FindStringSubmatch(s)
		offset := 0
		if m[2] != "" {
			offset, _ = strconv.Atoi(m[2])
		}
		return t.AddDate(0, 0, offset).Format(timePlaceholderLayouts[m[1]])
	})
}
