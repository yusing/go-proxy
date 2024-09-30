package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
)

func ReloadServer() E.NestedError {
	resp, err := httpClient.Post(fmt.Sprintf("%s/v1/reload", common.APIHTTPURL), "", nil)
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

func ListRoutes() (map[string]map[string]any, E.NestedError) {
	resp, err := httpClient.Get(fmt.Sprintf("%s/v1/list/routes", common.APIHTTPURL))
	if err != nil {
		return nil, E.From(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, E.Failure("list routes").Extraf("status code: %v", resp.StatusCode)
	}
	var routes map[string]map[string]any
	err = json.NewDecoder(resp.Body).Decode(&routes)
	if err != nil {
		return nil, E.From(err)
	}
	return routes, nil
}
