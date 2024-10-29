package query

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	v1 "github.com/yusing/go-proxy/internal/api/v1"
	U "github.com/yusing/go-proxy/internal/api/v1/utils"
	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/net/http/middleware"
)

func ReloadServer() E.Error {
	resp, err := U.Post(fmt.Sprintf("%s/v1/reload", common.APIHTTPURL), "", nil)
	if err != nil {
		return E.From(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		failure := E.Errorf("server reload status %v", resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return failure.With(err)
		}
		reloadErr := string(body)
		return failure.Withf(reloadErr)
	}
	return nil
}

func List[T any](what string) (_ T, outErr E.Error) {
	resp, err := U.Get(fmt.Sprintf("%s/v1/list/%s", common.APIHTTPURL, what))
	if err != nil {
		outErr = E.From(err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		outErr = E.Errorf("list %s: failed, status %v", what, resp.StatusCode)
		return
	}
	var res T
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		outErr = E.From(err)
		return
	}
	return res, nil
}

func ListRoutes() (map[string]map[string]any, E.Error) {
	return List[map[string]map[string]any](v1.ListRoutes)
}

func ListMiddlewareTraces() (middleware.Traces, E.Error) {
	return List[middleware.Traces](v1.ListMiddlewareTraces)
}

func DebugListTasks() (map[string]any, E.Error) {
	return List[map[string]any](v1.ListTasks)
}
