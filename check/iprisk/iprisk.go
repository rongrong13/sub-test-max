package iprisk

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
)

// ipapiResult 复刻 IP-Stream-Checker 中 core/sources/ipapi.py 的启发式 IP 风险检测。
// 数据源: ip-api.com(免费、无反爬、支持代理,获取的是节点出口 IP 的真实属性)。

type ipapiResponse struct {
	Status      string `json:"status"`
	Message     string `json:"message"`
	Query       string `json:"query"`
	CountryCode string `json:"countryCode"`
	ISP         string `json:"isp"`
	Org         string `json:"org"`
	Proxy       bool   `json:"proxy"`
	Hosting     bool   `json:"hosting"`
	Mobile      bool   `json:"mobile"`
}

// CLOUD_ASNS 常见云厂商 / 机房 ASN(命中即判定为机房类 IP)。
var cloudASNs = map[string]bool{
	"13335": true, "16509": true, "14618": true, "15169": true, "396982": true,
	"8075": true, "14061": true, "16276": true, "63949": true, "20473": true,
	"24940": true, "213230": true, "51167": true, "44684": true, "21859": true,
	"9009": true, "45102": true, "37963": true, "45090": true, "139341": true,
	"13432": true, "54113": true, "20940": true, "16625": true, "26496": true,
	"22822": true, "3333": true, "32244": true, "46606": true, "7018": true,
	"36351": true, "32097": true, "8100": true, "53850": true, "57374": true,
	"152194": true, "132203": true, "55990": true, "136907": true, "136958": true,
	"58519": true, "38895": true, "138692": true, "55967": true,
}

// cloudKeywords ISP/组织名中的机房关键词(中英文)。
var cloudKeywords = []string{
	"cloud", "hosting", "data center", "datacenter", "vps", "idc", "server",
	"机房", "数据中心", "阿里云", "腾讯云", "华为云", "aws", "azure", "gcp",
	"digitalocean", "linode", "vultr", "ovh", "hetzner", "contabo",
	"colocrossing", "kamatera", "serverius", "ionos", "hostwinds", "namecheap",
	"hostgator", "godaddy", "bluehost", "dreamhost", "liquidweb", "cogent",
	"zayo", "cologix", "equinix", "fiberhub", "leaseweb", "porkbun",
	"sakura", "conoha", "aws amazon", "amazon.com", "microsoft", "google llc",
}

// residentialKeywords 明显的住宅宽带 ISP 关键词(降低机房误判)。
var residentialKeywords = []string{
	"china telecom", "china unicom", "china mobile", "chunghwa telecom",
	"vodafone", "o2", "telefonica", "deutsche telekom", "orange",
	"comcast", "verizon", "at&t", "time warner", "charter", "cox",
	"sk telecom", "kt corp", "lg u+", "nokia", "中国电信", "中国联通", "中国移动",
	"台湾大哥大", "远传电信", "中华电信", "香港电讯", "和记电讯",
}

// Result 是一次 IP 风险检测的结构化结果。
type Result struct {
	IP        string
	Country   string
	RiskScore int    // 0-100
	RiskPct   string // "61%"
	Emoji     string
	IPAttr    string // 机房 / 住宅
	IPSource  string // 代理 / 原生
	Error     string
}

// isCloudOrg 判断 org/isp 是否命中云厂商 ASN 或机房关键词。
func isCloudOrg(org string) bool {
	orgLower := strings.ToLower(org)
	m := regexp.MustCompile(`AS(\d+)`).FindStringSubmatch(org)
	if len(m) > 1 && cloudASNs[m[1]] {
		return true
	}
	for _, kw := range cloudKeywords {
		if strings.Contains(orgLower, kw) {
			return true
		}
	}
	return false
}

// isResidentialOrg 判断 ISP/组织名是否命中明显住宅宽带关键词。
func isResidentialOrg(org string) bool {
	orgLower := strings.ToLower(org)
	for _, kw := range residentialKeywords {
		if strings.Contains(orgLower, kw) {
			return true
		}
	}
	return false
}

// scoreRisk 按 ipapi.py 的启发式规则计算风险度(0-100)。
func scoreRisk(data *ipapiResponse) int {
	hosting := data.Hosting
	proxy := data.Proxy
	mobile := data.Mobile
	org := data.Org + " " + data.ISP
	orgLower := strings.ToLower(org)

	isCloud := isCloudOrg(org)
	isResidential := isResidentialOrg(org)
	if isResidential {
		isCloud = false
	}

	score := 0
	if hosting && !isResidential {
		score += 45
	}
	if proxy {
		score += 35
	}
	if isCloud && !hosting {
		score += 15
	}
	if mobile {
		score -= 10
	}
	if isResidential {
		score -= 15
	}
	_ = orgLower
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score
}

// emojiFor 按阈值返回风险 emoji。
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

// Check 通过节点上的 http.Client 查询 ip-api.com,返回启发式 IP 风险检测结果。
func Check(httpClient *http.Client) *Result {
	const url = "http://ip-api.com/json/?fields=status,message,query,countryCode,isp,org,proxy,hosting,mobile"

	res := &Result{
		IP: "?", RiskPct: "?", Emoji: "❓",
		IPAttr: "未知", IPSource: "未知",
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		res.Error = err.Error()
		return res
	}

	// 使用与浏览器一致的轻量 HTTP(subs-check 的 mediaClient 已带节点代理与超时)
	resp, err := httpClient.Do(req)
	if err != nil {
		slog.Debug(fmt.Sprintf("ip-api 查询失败: %v", err))
		res.Error = err.Error()
		return res
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		res.Error = err.Error()
		return res
	}

	if resp.StatusCode == 429 {
		res.Error = "ip-api 限流(45次/分),请稍后再试"
		return res
	}
	if resp.StatusCode != http.StatusOK {
		res.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		return res
	}

	var data ipapiResponse
	if err := json.Unmarshal(body, &data); err != nil {
		res.Error = fmt.Sprintf("JSON 解析失败: %v", err)
		return res
	}
	if data.Status != "success" {
		res.Error = data.Message
		if res.Error == "" {
			res.Error = "ip-api 查询失败"
		}
		return res
	}

	score := scoreRisk(&data)
	org := data.Org + " " + data.ISP

	res.IP = data.Query
	res.Country = data.CountryCode
	res.RiskScore = score
	res.RiskPct = fmt.Sprintf("%d%%", score)
	res.Emoji = emojiFor(score)

	// 属性: 机房/住宅
	if data.Hosting || isCloudOrg(org) {
		res.IPAttr = "机房"
		if isResidentialOrg(org) && !data.Hosting {
			res.IPAttr = "住宅"
		}
	} else {
		res.IPAttr = "住宅"
	}
	// 来源: 代理/原生
	if data.Proxy {
		res.IPSource = "代理"
	} else {
		res.IPSource = "原生"
	}

	return res
}

// FullString 返回形如 "【emoji 机房|代理】" 的完整标注。
func (r *Result) FullString() string {
	return fmt.Sprintf("【%s %s|%s】", r.Emoji, r.IPAttr, r.IPSource)
}
