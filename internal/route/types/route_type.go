package types

type RouteType string

const (
	RouteTypeStream       RouteType = "stream"
	RouteTypeReverseProxy RouteType = "reverse_proxy"
)
