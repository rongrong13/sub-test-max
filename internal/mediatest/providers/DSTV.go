package providers

import (
	"github.com/beck-8/subs-check/internal/mediatest/core"
)

func DSTV(c core.HttpClient) core.Result {
	return core.CheckGETStatus(c, "https://now.dstv.com/", core.ResultMap{
		451: {Status: core.StatusNo},
		200: {Status: core.StatusOK},
	}, core.Result{Status: core.StatusUnexpected})
}
