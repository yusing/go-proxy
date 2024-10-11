package server

var proxyServer, apiServer *Server

func InitProxyServer(opt Options) *Server {
	if proxyServer == nil {
		proxyServer = NewServer(opt)
	}
	return proxyServer
}

func InitAPIServer(opt Options) *Server {
	if apiServer == nil {
		apiServer = NewServer(opt)
	}
	return apiServer
}

func GetProxyServer() *Server {
	return proxyServer
}

func GetAPIServer() *Server {
	return apiServer
}
