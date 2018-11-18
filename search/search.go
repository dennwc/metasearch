package search

import (
	"context"
	"net/url"

	"github.com/dennwc/metasearch/base"
)

type Searcher interface {
	Search(ctx context.Context, req Request) ResultIterator
	ContinueSearch(ctx context.Context, tok Token) ResultIterator
}

type Service interface {
	base.Provider
	Languages(ctx context.Context) ([]Language, error)

	Searcher
}

type Request struct {
	Query   string
	Lang    LangCode
	Country CountryCode
}

type ResultIterator interface {
	base.Iterator
	Result() Result
	Token() Token
}

var _ ResultIterator = Empty{}

type Empty struct {
	base.Empty
}

func (Empty) Result() Result {
	return nil
}

func (Empty) Token() Token {
	return nil
}

type Result interface {
	GetURL() *url.URL
	GetTitle() string
	GetDesc() string
}

type Token []byte
