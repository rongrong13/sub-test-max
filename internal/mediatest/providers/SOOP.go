package providers

import (
	"github.com/beck-8/subs-check/internal/mediatest/core"
)

func SOOP(c core.HttpClient) core.Result {
	resp, err := core.GET(c, "https://vod.sooplive.co.kr/player/97464151")
	if err != nil {
		return core.Result{Status: core.StatusNetworkErr, Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return core.Result{Status: core.StatusOK}
	}

	return core.Result{Status: core.StatusUnexpected}
}
