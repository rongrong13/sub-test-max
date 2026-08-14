package providers

import (
	"github.com/beck-8/subs-check/internal/mediatest/core"
)

func Popcornflix(c core.HttpClient) core.Result {
	resp, err := core.GET(c, "https://popcornflix-prod.cloud.seachange.com/cms/popcornflix/clientconfiguration/versions/2")
	if err != nil {
		return core.Result{Status: core.StatusNetworkErr, Err: err}
	}
	defer resp.Body.Close()

	return core.ResultFromMapping(resp.StatusCode, core.ResultMap{
		200: {Status: core.StatusOK},
		403: {Status: core.StatusNo},
	}, core.Result{Status: core.StatusUnexpected})
}
