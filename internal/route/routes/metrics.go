package routes

import "github.com/yusing/go-proxy/internal/metrics"

func init() {
	metrics.InitRouterMetrics(httpRoutes.Size, streamRoutes.Size)
}
