package duckduckgo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/dennwc/metasearch/providers"
	"github.com/dennwc/metasearch/search"
)

const (
	provName = "duckduckgo"
	baseURL  = "https://duckduckgo.com"
)

func init() {
	providers.Register(provName, func(ctx context.Context) (providers.Provider, error) {
		return New(), nil
	})
}

var (
	_ search.Service = (*Service)(nil)
)

func New() *Service {
	return &Service{
		HTTPClient: providers.NewHTTPClient(""),
	}
}

type Service struct {
	providers.HTTPClient
}

func (*Service) ID() string {
	return provName
}

func (s *Service) Search(ctx context.Context, req search.Request) search.ResultIterator {
	r := SearchReq{
		Query: req.Query,
	}
	if !req.Lang.IsRoot() {
		r.Region = toRegion(req.Lang.String())
	}
	return &searchIter{s: s, next: s.newSearch(r)}
}

func (s *Service) ContinueSearch(ctx context.Context, tok search.Token) search.ResultIterator {
	var t token
	if err := json.Unmarshal([]byte(tok), &t); err != nil {
		return &searchIter{err: err}
	}
	resp, next, err := s.search(ctx, t.Cur)
	return &searchIter{s: s, cur: t.Cur, next: next, page: resp.Results, i: t.Off, err: err}
}

type searchIter struct {
	s    *Service
	cur  url.Values
	next url.Values

	page []Result
	i    int
	err  error
}

func (it *searchIter) NextPage(ctx context.Context) bool {
	if it.err != nil || it.s == nil {
		return false
	}
	it.page = nil
	if it.next == nil {
		return false
	}
	it.cur = it.next
	resp, next, err := it.s.search(ctx, it.next)
	if err != nil {
		it.err = err
		return false
	}
	it.next = next
	it.page = resp.Results
	it.i = -1
	return len(it.page) > 0
}

func (it *searchIter) Buffered() int {
	n := len(it.page) - (it.i + 1)
	if n < 0 {
		n = 0
	}
	return n
}

func (it *searchIter) Next(ctx context.Context) bool {
	if it.err != nil || it.s == nil {
		return false
	}
	if it.i+1 >= len(it.page) {
		if !it.NextPage(ctx) {
			return false
		}
	}
	it.i++
	return true
}

func (it *searchIter) Close() error {
	it.page = nil
	return nil
}

func (it *searchIter) Err() error {
	return it.err
}

func (it *searchIter) Result() search.Result {
	if it.i >= len(it.page) {
		return nil
	}
	r := it.page[it.i]
	u, err := url.Parse(r.URL)
	if err != nil {
		it.err = err
		return nil
	}
	return &search.LinkResult{
		URL: *u, Title: r.Title, Desc: r.Content,
	}
}

func (it *searchIter) Token() search.Token {
	data, err := json.Marshal(token{
		Cur: it.cur,
		Off: it.i,
	})
	if err != nil {
		it.err = err
		return nil
	}
	return search.Token(data)
}

type token struct {
	Cur url.Values `json:"req"`
	Off int        `json:"off,omitempty"`
}

type SearchReq struct {
	Region regionCode `json:"lang"`
	Query  string     `json:"q"`
}

type SearchResp struct {
	Results []Result
}

type Result struct {
	URL     string
	Title   string
	Content string
}

func (s *Service) newSearch(r SearchReq) url.Values {
	if r.Region == "" {
		r.Region = defaultRegion
	}
	params := make(url.Values)
	params.Set("q", r.Query)
	params.Set("kl", string(r.Region))
	return params
}

func (s *Service) search(ctx context.Context, params url.Values) (*SearchResp, url.Values, error) {
	doc, err := s.GetHTML(ctx, baseURL+"/html", params)
	if err != nil {
		return nil, nil, err
	}
	// extract search results
	var out []Result
	doc.Find(`div.result.results_links.results_links_deep.result`).Each(func(_ int, sel *goquery.Selection) {
		var r Result
		a := sel.Find(`a.result__a`).First()
		if a.Size() == 0 {
			return
		}
		r.URL = a.AttrOr("href", "")
		if r.URL == "" {
			err = fmt.Errorf("cannot parse result url")
			return
		}
		if !strings.HasPrefix(r.URL, "http") {
			// relative link
			i := strings.Index(r.URL, "?")
			u, _ := url.ParseQuery(r.URL[i:])
			if v := u.Get("uddg"); v != "" {
				r.URL = v
			} else if v := u.Get("u3"); v != "" {
				r.URL = v
			} else {
				r.URL = baseURL + r.URL
			}
		}
		r.Title = a.Text()
		r.Content = sel.Find(`a.result__snippet`).First().Text()
		out = append(out, r)
	})
	if err != nil {
		return nil, nil, err
	}
	// extract continuation token
	var next url.Values
	doc.Find(`input[type='submit']`).Each(func(_ int, sel *goquery.Selection) {
		sel = sel.Parent()
		dc := sel.Find(`input[name='dc']`).First()
		if dc == nil || strings.HasPrefix(dc.AttrOr("value", "-"), "-") {
			return
		}
		if next != nil {
			err = fmt.Errorf("multiple continuation tokens found")
			return
		}
		next = make(url.Values)
		sel.Find(`input[value]`).Each(func(_ int, kv *goquery.Selection) {
			name, ok := kv.Attr("name")
			if !ok {
				return
			}
			value, ok := kv.Attr("value")
			if !ok {
				return
			}
			next.Set(name, value)
		})
		if len(next) == 0 {
			next = nil
			err = fmt.Errorf("cannot build continuation token")
			return
		}
	})
	return &SearchResp{Results: out}, next, err
}
