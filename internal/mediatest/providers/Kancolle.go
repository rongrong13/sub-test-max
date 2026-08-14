package providers

import (
	"github.com/rongrong13/sub-test-max/internal/mediatest/core"
)

func Kancolle(c core.HttpClient) core.Result {
	return core.CheckDalvikStatus(c, "https://w00g.kancolle-server.com/kcscontents/news/", core.ResultMap{
		200: {Status: core.StatusOK},
		403: {Status: core.StatusNo},
		302: {Status: core.StatusNo},
	}, core.Result{Status: core.StatusUnexpected})
}
