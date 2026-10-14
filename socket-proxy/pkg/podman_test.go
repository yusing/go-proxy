package socketproxy_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/yusing/godoxy/socketproxy/pkg"
)

func TestPodmanInspectOnly(t *testing.T) {
	oldContainers, oldPost := DockerContainers, DockerPost
	t.Cleanup(func() { DockerContainers, DockerPost = oldContainers, oldPost })
	DockerPost = true
	for _, enabled := range []bool{false, true} {
		DockerContainers = enabled
		h := NewHandler()
		for _, prefix := range []string{"", "/v4.0.0"} {
			for _, tc := range []struct {
				method, path string
				allowed      bool
			}{
				{"GET", "/libpod/containers/app/json", true},
				{"POST", "/libpod/containers/app/json", false},
				{"GET", "/libpod/containers/json", false},
				{"POST", "/libpod/containers/app/checkpoint", false},
			} {
				w := httptest.NewRecorder()
				h.ServeHTTP(w, httptest.NewRequest(tc.method, prefix+tc.path, nil))
				allowed := w.Code == http.StatusOK
				if allowed != (enabled && tc.allowed) {
					t.Errorf("containers=%v %s %s: status %d", enabled, tc.method, prefix+tc.path, w.Code)
				}
			}
		}
	}
}
