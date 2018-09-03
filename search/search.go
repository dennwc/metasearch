package search

import (
	"context"
	"net/url"

	"github.com/nwca/metasearch/base"
)

type Service interface {
	Languages(ctx context.Context) ([]Language, error)

	Search(ctx context.Context, req Request) ResultIterator
	ContinueSearch(ctx context.Context, tok Token) ResultIterator
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

type Result interface {
	GetURL() url.URL
	GetTitle() string
	GetDesc() string
}

type Token []byte
