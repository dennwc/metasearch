package duckduckgo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/nwca/metasearch/providers"
	"github.com/nwca/metasearch/search"
)

const (
	provName = "duckduckgo"
	baseURL  = "https://duckduckgo.com"
	perPage  = 30
)

func init() {
	search.RegisterService(provName, func(ctx context.Context) (search.Service, error) {
		return New(), nil
	})
}

var _ search.Service = (*Service)(nil)

func New() *Service {
	return &Service{
		HTTPClient: providers.NewHTTPClient(baseURL),
	}
}

type Service struct {
	providers.HTTPClient

	regions struct {
		sync.RWMutex
		list []Region
	}
}

func (*Service) ID() string {
	return provName
}

func (s *Service) Languages(ctx context.Context) ([]search.Language, error) {
	list, err := s.Regions(ctx)
	if err != nil {
		return nil, err
	}
	langs := make([]search.Language, 0, len(list))
	for _, r := range list {
		c, err := search.ParseLangCode(r.Code) // be-fr
		if err != nil {
			return nil, err
		}
		langs = append(langs, search.Language{
			Name: r.Name, // Belgium (fr)
			Code: c,
		})
	}
	return langs, nil
}

func (s *Service) Search(ctx context.Context, req search.Request) search.ResultIterator {
	r := SearchReq{
		Region: strings.ToLower(req.Lang.String()),
		Query:  req.Query,
		Offset: 0,
	}
	return &searchIter{s: s, cur: r}
}

func (s *Service) ContinueSearch(ctx context.Context, tok search.Token) search.ResultIterator {
	var t token
	if err := json.Unmarshal([]byte(tok), &t); err != nil {
		return &searchIter{err: err}
	}
	resp, err := s.SearchRaw(ctx, t.Cur)
	return &searchIter{s: s, cur: t.Cur, page: resp.Results, i: t.Off, err: err}
}

type searchIter struct {
	s   *Service
	cur SearchReq

	page []Result
	i    int
	err  error
}

func (it *searchIter) Next(ctx context.Context) bool {
	if it.err != nil || it.s == nil {
		return false
	}
	if it.i+1 < len(it.page) {
		it.i++
		return true
	}
	it.cur.Offset += len(it.page)
	if n := it.cur.Offset % perPage; it.cur.Offset > perPage && n != 0 {
		// DDG places an ad result into the list but fails to fetch the next page with offset 31 instead of 30
		it.cur.Offset -= n
	}
	it.page = nil
	resp, err := it.s.SearchRaw(ctx, it.cur)
	if err != nil {
		it.err = err
		return false
	}
	it.page = resp.Results
	it.i = 0
	return len(it.page) > 0
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
	Cur SearchReq `json:"req"`
	Off int       `json:"off,omitempty"`
}

type SearchReq struct {
	Region string `json:"lang"`
	Query  string `json:"q"`
	Offset int    `json:"off"`
}

type SearchResp struct {
	Results []Result
}

var langAliases = map[string]string{
	"ar-SA":  "ar-XA",
	"es-419": "es-XL",
	"ja":     "jp-JP",
	"ko":     "kr-KR",
	"sl-SI":  "sl-SL",
	"zh-TW":  "tzh-TW",
	"zh-HK":  "tzh-HK",
}

const defaultRegion = "us-en"

type Result struct {
	URL     string
	Title   string
	Content string
}

func (s *Service) SearchRaw(ctx context.Context, r SearchReq) (*SearchResp, error) {
	if r.Region == "" {
		r.Region = defaultRegion
	}
	params := make(url.Values)
	params.Set("q", r.Query)
	params.Set("kl", r.Region)
	params.Set("s", strconv.Itoa(r.Offset))
	params.Set("dc", strconv.Itoa(r.Offset))

	doc, err := s.GetHTML(ctx, "/html", params)
	if err != nil {
		return nil, err
	}
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
			} else {
				r.URL = baseURL + r.URL
			}
		}
		r.Title = a.Text()
		r.Content = sel.Find(`a.result__snippet`).First().Text()
		out = append(out, r)
	})
	if err != nil {
		return nil, err
	}
	return &SearchResp{Results: out}, nil
}

type Region struct {
	Code string
	Name string
}

func (s *Service) fetchRegions(ctx context.Context) ([]Region, error) {
	resp, err := s.Get(ctx, "/util/u172.js", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, err
	}
	const start = `regions:{`
	i := bytes.Index(data, []byte(start))
	if i < 0 {
		return nil, fmt.Errorf("cannot parse languages list")
	}
	data = data[i+len(start)-1:]
	const end = `}`
	i = bytes.Index(data, []byte(end))
	if i < 0 {
		return nil, fmt.Errorf("cannot parse languages list")
	}
	data = data[:i+1]
	var regions map[string]string
	if err = json.Unmarshal(data, &regions); err != nil {
		return nil, fmt.Errorf("cannot parse languages list: %v", err)
	}
	out := make([]Region, 0, len(regions))
	for code, name := range regions {
		if c, ok := langAliases[code]; ok {
			code = c
		}
		out = append(out, Region{Code: code, Name: name})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Code < out[j].Code
	})
	return out, nil
}

func (s *Service) Regions(ctx context.Context) ([]Region, error) {
	s.regions.RLock()
	list := s.regions.list
	s.regions.RUnlock()
	if list != nil {
		return append([]Region{}, list...), nil
	}
	s.regions.Lock()
	defer s.regions.Unlock()
	if list = s.regions.list; list != nil {
		return append([]Region{}, list...), nil
	}
	list, err := s.fetchRegions(ctx)
	if err != nil {
		return nil, err
	}
	s.regions.list = list
	return append([]Region{}, list...), nil
}
