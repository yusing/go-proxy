package route

import "github.com/yusing/go-proxy/internal/metrics"

func init() {
	metrics.InitRouterMetrics(httpRoutes.Size, streamRoutes.Size)
}
