package google

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/dennwc/metasearch/providers"
	"github.com/dennwc/metasearch/search"
)

var _ search.Service = (*Service)(nil)

const (
	provName        = "google"
	perPage         = 10
	defaultHostname = "www.google.com"
	searchPath      = "/search"
)

func init() {
	providers.Register(provName, func(ctx context.Context) (providers.Provider, error) {
		return New(), nil
	})
}

func New() *Service {
	return &Service{
		HTTPClient: providers.NewHTTPClient(""),
	}
}

type Service struct {
	providers.HTTPClient

	UseLocalDomain bool
}

func (*Service) ID() string {
	return provName
}

func (s *Service) Search(ctx context.Context, req search.Request) search.ResultIterator {
	r := SearchReq{
		Query:  req.Query,
		Offset: 0,
	}
	if req.Lang != (search.LangCode{}) {
		r.Language = req.Lang.String()
	}
	if req.Region != (search.RegionCode{}) {
		r.Country = req.Region.String()
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

func (it *searchIter) NextPage(ctx context.Context) bool {
	if it.err != nil || it.s == nil {
		return false
	}
	it.cur.Offset += len(it.page)
	it.page = nil
	resp, err := it.s.SearchRaw(ctx, it.cur)
	if err != nil {
		it.err = err
		return false
	}
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
	Cur SearchReq `json:"req"`
	Off int       `json:"off,omitempty"`
}

type SearchReq struct {
	Query    string `json:"q"`
	Offset   int    `json:"off"`
	Language string `json:"lang"`
	Country  string `json:"country"`
}

type Result struct {
	Title   string
	URL     string
	Content string
}

type SearchResp struct {
	Total   uint
	Results []Result
}

func (s *Service) searchPage(ctx context.Context, r SearchReq) (io.ReadCloser, string, error) {
	if r.Language == "" {
		r.Language = defaultLanguage
	}
	hostname := defaultHostname
	if !s.UseLocalDomain {
		if h, ok := countryHostname[r.Country]; ok {
			hostname = h
		}
	}
	params := make(url.Values)
	params.Set("q", r.Query)
	params.Set("start", strconv.Itoa(r.Offset))
	params.Set("gws_rd", "cr")
	params.Set("gbv", "1")
	params.Set("lr", "lang_"+r.Language)
	params.Set("hl", r.Language)
	params.Set("ei", "x")
	if r.Country != "" {
		// TODO: derive country from the language?
		params.Set("cr", "country"+strings.ToUpper(r.Country))
	}

	base := "https://" + hostname
	req, err := s.GetRequest(base+searchPath, params)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept-Language", r.Language+","+r.Language+"-"+r.Country)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.AddCookie(&http.Cookie{
		Name: "GOOGLE_ABUSE_EXEMPTION", Value: "x",
	})
	resp, err := s.DoRaw(ctx, req)
	if err != nil {
		return nil, "", err
	}
	return resp.Body, base, nil
}

var reTotal = regexp.MustCompile(`([\d,]+)`)

func (s *Service) SearchRaw(ctx context.Context, r SearchReq) (*SearchResp, error) {
	body, base, err := s.searchPage(ctx, r)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	// FIXME: handle "sorry" page
	var rd io.Reader = body
	if false {
		rd = io.TeeReader(rd, os.Stderr)
	}
	doc, err := goquery.NewDocumentFromReader(rd)
	if err != nil {
		return nil, err
	}
	// FIXME: parse instant answers
	out := &SearchResp{}
	doc.Find(`#resultStats`).Each(func(_ int, sel *goquery.Selection) {
		s := sel.Text() // About 1,100,000,000 results
		s = reTotal.FindString(s)
		if s == "" {
			return
		}
		s = strings.Replace(s, ",", "", -1)
		v, _ := strconv.ParseUint(s, 10, 64)
		out.Total = uint(v)
	})
	doc.Find(`div.g`).Each(func(_ int, sel *goquery.Selection) {
		h := sel.Find("h3").First()
		link := h.Find(`a`).First().AttrOr("href", "")
		if link == "" {
			return
		}
		if strings.HasPrefix(link, searchPath) {
			return // TODO: parse these results as well
		}
		title := h.Text()
		if strings.HasPrefix(link, "/url?") {
			if u, err := url.ParseQuery(link[5:]); err == nil {
				v := u.Get("q")
				if strings.HasPrefix(v, "http") {
					link = v
				}
			}
		}
		if !strings.HasPrefix(link, "http") {
			link = base + link
		}
		content := sel.Find(`span.st`).First().Text()
		out.Results = append(out.Results, Result{
			Title: title, URL: link, Content: content,
		})
	})
	return out, nil
}
