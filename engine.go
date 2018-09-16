package metasearch

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/nwca/metasearch/search"
)

var _ search.Searcher = (*Engine)(nil)

func NewEngine(ctx context.Context, provs ...search.Service) (*Engine, error) {
	if len(provs) == 0 {
		provs = nil
		for _, fnc := range search.ListServices() {
			p, err := fnc(ctx)
			if err != nil {
				return nil, err
			}
			provs = append(provs, p)
		}
	}
	if len(provs) == 0 {
		return nil, fmt.Errorf("none providers were selected")
	}
	// TODO: request supported languages, convert language from the request to the right one for this provider
	s := &Engine{
		provs: provs,
		byID:  make(map[string]search.Service),
	}
	for _, p := range provs {
		s.byID[p.ID()] = p
	}
	return s, nil
}

type Engine struct {
	provs []search.Service
	byID  map[string]search.Service
}

func (s *Engine) Search(ctx context.Context, req search.Request) search.ResultIterator {
	its := make([]search.ResultIterator, 0, len(s.provs))
	ids := make([]string, 0, len(s.provs))
	for _, p := range s.provs {
		it := p.Search(ctx, req)
		if err := it.Err(); err != nil {
			it.Close()
			log.Println(err)
			continue
		}
		its = append(its, it)
		ids = append(ids, p.ID())
	}
	return &multiIterator{its: its, ids: ids, i: -1}
}

func (s *Engine) ContinueSearch(ctx context.Context, tok search.Token) search.ResultIterator {
	var t multiToken
	if err := json.Unmarshal([]byte(tok), &t); err != nil {
		return &multiIterator{err: err}
	}
	it := &multiIterator{i: int(t.Cur) - 1}
	discard := func() {
		for _, it := range it.its {
			it.Close()
		}
	}
	for _, pt := range t.Provs {
		p, ok := s.byID[pt.ID]
		if !ok {
			discard()
			return &multiIterator{err: fmt.Errorf("provider %q is not defined", pt.ID)}
		}
		sit := p.ContinueSearch(ctx, pt.Tok)
		if err := sit.Err(); err != nil {
			sit.Close()
			discard()
			return &multiIterator{err: err}
		}
		it.its = append(it.its, sit)
		it.ids = append(it.ids, pt.ID)
	}
	return it
}

type multiIterator struct {
	its []search.ResultIterator
	ids []string
	i   int
	err error
}

func (it *multiIterator) Next(ctx context.Context) bool {
	for it.err == nil && len(it.its) > 0 {
		it.i++
		if it.i >= len(it.its) {
			it.i = 0
		}
		i := it.i
		cur := it.its[i]
		if cur.Next(ctx) {
			return true
		}
		if err := cur.Err(); err != nil {
			log.Println(err)
		}
		cur.Close()
		it.its = append(it.its[:i], it.its[i+1:]...)
		it.ids = append(it.ids[:i], it.ids[i+1:]...)
		it.i--
	}
	return false
}

func (it *multiIterator) Close() error {
	for _, it := range it.its {
		it.Close()
	}
	it.its = nil
	return nil
}

func (it *multiIterator) Err() error {
	return it.err
}

func (it *multiIterator) Result() search.Result {
	if it.i < 0 || it.i >= len(it.its) {
		return nil
	}
	return it.its[it.i].Result()
}

func (it *multiIterator) Token() search.Token {
	if len(it.its) == 0 {
		return nil
	}
	var tok multiToken
	for i, sit := range it.its {
		t := sit.Token()
		if t == nil {
			if err := sit.Err(); err != nil {
				it.err = err
			}
			continue
		}
		if i == it.i {
			tok.Cur = uint(len(tok.Provs))
		}
		tok.Provs = append(tok.Provs, provToken{
			ID:  it.ids[i],
			Tok: t,
		})
	}

	data, err := json.Marshal(tok)
	if err != nil {
		it.err = err
		return nil
	}
	return search.Token(data)
}

type provToken struct {
	ID  string       `json:"id"`
	Tok search.Token `json:"tok"`
}

type multiToken struct {
	Provs []provToken `json:"provs"`
	Cur   uint        `json:"cur"`
}
