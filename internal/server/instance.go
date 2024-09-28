package server

var proxyServer, apiServer *server

func InitProxyServer(opt Options) *server {
	if proxyServer == nil {
		proxyServer = NewServer(opt)
	}
	return proxyServer
}

func InitAPIServer(opt Options) *server {
	if apiServer == nil {
		apiServer = NewServer(opt)
	}
	return apiServer
}

func GetProxyServer() *server {
	return proxyServer
}

func GetAPIServer() *server {
	return apiServer
}
