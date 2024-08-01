package proxy

var (
	PathMode_Forward     = "forward"
	PathMode_RemovedPath = ""
)

const (
	StreamType_UDP string = "udp"
	StreamType_TCP string = "tcp"
	// StreamType_UDP_TCP Scheme = "udp-tcp"
	// StreamType_TCP_UDP Scheme = "tcp-udp"
	// StreamType_TLS Scheme = "tls"
)

var (
	// TODO: support "tcp-udp", "udp-tcp", etc.
	StreamSchemes = []string{StreamType_TCP, StreamType_UDP}
	HTTPSchemes   = []string{"http", "https"}
	ValidSchemes  = append(StreamSchemes, HTTPSchemes...)
)

