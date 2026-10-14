package routeApi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	entrypoint "github.com/yusing/godoxy/internal/entrypoint"
	"github.com/yusing/godoxy/internal/routing"
	apitypes "github.com/yusing/goutils/apitypes"
	"github.com/yusing/goutils/http/httpheaders"
	"github.com/yusing/goutils/http/websocket"
)

// @x-id				"list"
// @BasePath		/api/v1
// @Summary		List routes
// @Description	List routes
// @Tags			route,websocket
// @Accept			json
// @Produce		json
// @Param			provider	query		string	false	"Provider"
// @Success		200			{array}		RouteType
// @Failure		403			{object}	apitypes.ErrorResponse
// @Router			/route/list [get]
func Routes(c *gin.Context) {
	if httpheaders.IsWebsocket(c.Request.Header) {
		RoutesWS(c)
		return
	}

	ep := entrypoint.FromCtx(c.Request.Context())
	if ep == nil { // impossible for the main API, but debug server has no entrypoint context.
		c.JSON(http.StatusInternalServerError, apitypes.Error("entrypoint not initialized"))
		return
	}

	provider := c.Query("provider")
	c.JSON(http.StatusOK, routeResponses(ep, provider))
}

func RoutesWS(c *gin.Context) {
	ep := entrypoint.FromCtx(c.Request.Context())
	if ep == nil { // impossible for the main API, but debug server has no entrypoint context.
		c.JSON(http.StatusInternalServerError, apitypes.Error("entrypoint not initialized"))
		return
	}

	provider := c.Query("provider")
	websocket.PeriodicWrite(c, 3*time.Second, func() (any, error) {
		return routeResponses(ep, provider), nil
	})
}

func routeResponses(ep routing.Entrypoint, provider string) []routeResponse {
	rts := make([]routeResponse, 0, ep.NumRoutes())
	for r := range ep.IterRoutes {
		if provider == "" || r.ProviderName() == provider {
			rts = append(rts, routeResponse{r, ep.PublicURLs(r)})
		}
	}
	return rts
}
