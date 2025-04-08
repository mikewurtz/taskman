package grpc

type contextKey string

const (
    ClientCNKey = contextKey("clientCN")
    
    // TODO: in a real environment these would be set via environment variables and not be hardcoded
    // We would likely want to use a config package to manage these values such as https://github.com/kelseyhightower/envconfig
    CaCertPath = "certs/ca.crt"
    ServerCertPath = "certs/server.crt"
    ServerKeyPath = "certs/server.key"
)



