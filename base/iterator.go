package base

import "context"

type Iterator interface {
	Next(ctx context.Context) bool
	Close() error
	Err() error
}
