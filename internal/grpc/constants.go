package grpc

type contextKey string

const (
	ClientCNKey    = contextKey("clientCN")
	ServerCertName = "server"
	CACertFileName = "ca.crt"
)
