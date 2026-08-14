package providers

import (
	"github.com/beck-8/subs-check/internal/mediatest/core"
)

func PandaTV(c core.HttpClient) core.Result {
	resp, err := core.GET(c, "https://api.pandalive.co.kr/v1/live/play")
	if err != nil {
		return core.Result{Status: core.StatusNetworkErr, Err: err}
	}
	defer resp.Body.Close()

	return core.ResultFromMapping(resp.StatusCode, core.ResultMap{
		400: {Status: core.StatusOK},
		403: {Status: core.StatusNo},
	}, core.Result{Status: core.StatusUnexpected})
}
