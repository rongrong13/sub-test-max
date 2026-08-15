package providers

import (
	"io"

	"github.com/rongrong13/sub-test-max/internal/mediatest/core"
)

// Facebook 检测节点能否正常访问 Facebook。
// 200/301/302 = 可访问; 403 = 区域封锁; 429 = 限流; 其它 = 意外。
func Facebook(c core.HttpClient) core.Result {
	resp, err := core.GET(c, "https://www.facebook.com/")
	if err != nil {
		return core.Result{Status: core.StatusNetworkErr, Err: err}
	}
	defer resp.Body.Close()
	if _, err := io.ReadAll(resp.Body); err != nil {
		return core.Result{Status: core.StatusNetworkErr, Err: err}
	}

	switch resp.StatusCode {
	case 200, 301, 302:
		return core.Result{Status: core.StatusOK}
	case 403, 404:
		return core.Result{Status: core.StatusNo}
	case 429:
		return core.Result{Status: core.StatusBanned}
	}
	return core.Result{Status: core.StatusUnexpected}
}
