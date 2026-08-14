package providers

import (
	"github.com/rongrong13/sub-test-max/internal/mediatest/core"
)

func ZDF(c core.HttpClient) core.Result {
	return core.CheckStatusWithTimeout(c, "https://ssl.zdf.de/geo/de/geo.txt", core.ResultMap{
		403: {Status: core.StatusNo},
		200: {Status: core.StatusOK},
	}, core.Result{Status: core.StatusUnexpected}, 15)
}
