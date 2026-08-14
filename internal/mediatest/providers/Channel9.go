package providers

import (
	"github.com/beck-8/subs-check/internal/mediatest/core"
)

func Channel9(c core.HttpClient) core.Result {
	return core.CheckGETStatus(c, "https://login.nine.com.au", core.ResultMap{
		403: {Status: core.StatusNo},
		302: {Status: core.StatusOK},
	}, core.Result{Status: core.StatusUnexpected})
}
