package utils

import (
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
