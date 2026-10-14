package api

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yusing/godoxy/internal/env"
	apitypes "github.com/yusing/goutils/apitypes"
)

const readOnlyBearerKey = "godoxy.readOnlyBearer"

// readOnlyBearerMiddleware owns API-only machine credentials. Construct it only
// for authenticated handlers, never for local_api or disabled authentication.
func readOnlyBearerMiddleware() gin.HandlerFunc {
	// Retain only a fixed-size digest, not the environment's plaintext token.
	token := env.String("API_READONLY_TOKEN")
	enabled := token != ""
	digest := sha256.Sum256([]byte(token))
	return func(c *gin.Context) {
		values, present := c.Request.Header["Authorization"]
		if !present {
			c.Next()
			return
		}
		// An explicit credential must never fall back to a more privileged cookie.
		if len(values) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apitypes.Error("Unauthorized"))
			return
		}
		scheme, credential, ok := strings.Cut(values[0], " ")
		candidate := sha256.Sum256([]byte(credential))
		if !ok || !strings.EqualFold(scheme, "Bearer") || strings.ContainsAny(credential, " \t\r\n") || !enabled ||
			subtle.ConstantTimeCompare(candidate[:], digest[:]) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apitypes.Error("Unauthorized"))
			return
		}
		if !readOnlyRequest(c) {
			c.AbortWithStatusJSON(http.StatusForbidden, apitypes.Error("read-only token cannot access this endpoint"))
			return
		}
		c.Set(readOnlyBearerKey, true)
		c.Next()
	}
}

func readOnlyRequest(c *gin.Context) bool {
	if c.Request.Method != http.MethodGet || len(c.Request.Header.Values("Upgrade")) != 0 {
		return false
	}
	for _, value := range c.Request.Header.Values("Connection") {
		for part := range strings.SplitSeq(value, ",") {
			if strings.EqualFold(strings.TrimSpace(part), "upgrade") {
				return false
			}
		}
	}
	// Match registered route patterns, not path prefixes or all GET requests.
	// Only these handlers have been audited for side-effect-free HTTP reads.
	// Static siblings (e.g. /route/validate) do not match /route/:which.
	switch c.FullPath() {
	case "/api/v1/route/list", "/api/v1/route/:which":
		return true
	default:
		return false
	}
}
