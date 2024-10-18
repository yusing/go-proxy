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

func ReloadServer() E.NestedError {
	resp, err := U.Post(fmt.Sprintf("%s/v1/reload", common.APIHTTPURL), "", nil)
	if err != nil {
		return E.From(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		failure := E.Failure("server reload").Extraf("status code: %v", resp.StatusCode)
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return failure.Extraf("unable to read response body: %s", err)
		}
		reloadErr, ok := E.FromJSON(b)
		if ok {
			return E.Join("reload success, but server returned error", reloadErr)
		}
		return failure.Extraf("unable to read response body")
	}
	return nil
}

func List[T any](what string) (_ T, outErr E.NestedError) {
	resp, err := U.Get(fmt.Sprintf("%s/v1/list/%s", common.APIHTTPURL, what))
	if err != nil {
		outErr = E.From(err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		outErr = E.Failure("list "+what).Extraf("status code: %v", resp.StatusCode)
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

func ListRoutes() (map[string]map[string]any, E.NestedError) {
	return List[map[string]map[string]any](v1.ListRoutes)
}

func ListMiddlewareTraces() (middleware.Traces, E.NestedError) {
	return List[middleware.Traces](v1.ListMiddlewareTraces)
}

func DebugListTasks() (map[string]any, E.NestedError) {
	return List[map[string]any](v1.ListTasks)
}
