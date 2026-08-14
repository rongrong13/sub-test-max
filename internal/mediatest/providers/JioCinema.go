package providers

import (
	"github.com/beck-8/subs-check/internal/mediatest/core"
)

func JioCinema(c core.HttpClient) core.Result {
	return core.CheckGETStatus(c, "https://content-jiovoot.voot.com/psapi/", core.ResultMap{
		200: {Status: core.StatusOK},
		474: {Status: core.StatusNo},
	}, core.Result{Status: core.StatusUnexpected})
}
