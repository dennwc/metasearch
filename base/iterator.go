package base

import "context"

type Iterator interface {
	Next(ctx context.Context) bool
	Close() error
	Err() error
}

var _ PagedIterator = Empty{}

type Empty struct{}

func (Empty) Next(ctx context.Context) bool {
	return false
}

func (Empty) NextPage(ctx context.Context) bool {
	return false
}

func (Empty) Buffered() int {
	return 0
}

func (Empty) Close() error {
	return nil
}

func (Empty) Err() error {
	return nil
}

type PagedIterator interface {
	Iterator
	NextPage(ctx context.Context) bool
	Buffered() int
}
