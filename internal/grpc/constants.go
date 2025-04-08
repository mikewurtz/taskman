package grpc

import "time"

type contextKey string

const (
	ClientCNKey = contextKey("clientCN")

	// In a real environment these would be set via environment variables and not be hardcoded
	// we would want to use a config package to manage these values and could load these in from env vars and utilize
	// https://github.com/kelseyhightower/envconfig

	ServerCertName = "server"
	CACertFileName = "ca.crt"

	// Timeout config

	// Starting a task may take a bit longer; use a conservative upper bound.
	StartTaskTimeout = 1 * time.Minute
	// Stopping should be quick; 10 seconds is usually sufficient.
	StopTaskTimeout = 10 * time.Second
	// Retrieving task status should be fast, so we use a shorter timeout.
	GetTaskStatusTimeout = 10 * time.Second
)
