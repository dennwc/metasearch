package base

import "context"

type Iterator interface {
	Next(ctx context.Context) bool
	Close() error
	Err() error
}

var _ Iterator = Empty{}

type Empty struct{}

func (Empty) Next(ctx context.Context) bool {
	return false
}

func (Empty) Close() error {
	return nil
}

func (Empty) Err() error {
	return nil
}
