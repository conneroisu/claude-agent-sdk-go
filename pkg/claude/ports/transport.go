package ports

import "context"

// Transport defines what the domain needs from a transport layer
type Transport interface {
	Connect(ctx context.Context) error
	Write(ctx context.Context, data string) error
	ReadMessages(ctx context.Context) (<-chan map[string]any, <-chan error)
	EndInput() error
	Close() error
	IsReady() bool
}
