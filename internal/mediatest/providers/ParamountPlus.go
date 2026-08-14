package providers

import (
	"github.com/rongrong13/sub-test-max/internal/mediatest/core"
	"io"
	"strings"
)

func ParamountPlus(c core.HttpClient) core.Result {
	resp, err := core.GET(c, "https://www.paramountplus.com/")
	if err != nil {
		return core.Result{Status: core.StatusNetworkErr, Err: err}
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return core.Result{Status: core.StatusNetworkErr, Err: err}
	}
	s := string(b)
	if strings.Contains(s, "intl") {
		return core.Result{Status: core.StatusOK}
	}
	return core.Result{Status: core.StatusNo}
}
