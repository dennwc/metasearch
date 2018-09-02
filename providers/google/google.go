package google

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/nwca/metasearch/providers"
	"github.com/nwca/metasearch/search"
)

var _ search.Service = (*Service)(nil)

const (
	perPage         = 10
	defaultHostname = "www.google.com"
	defaultCountry  = "US"
	defaultLanguage = "en"
	languagesURL    = "https://" + defaultHostname + "/preferences?#languages"
	searchPath      = "/search"
)

var (
	countryHostname = map[string]string{
		"BG": "www.google.bg",     // Bulgaria
		"CZ": "www.google.cz",     // Czech Republic
		"DE": "www.google.de",     // Germany
		"DK": "www.google.dk",     // Denmark
		"AT": "www.google.at",     // Austria
		"CH": "www.google.ch",     // Switzerland
		"GR": "www.google.gr",     // Greece
		"AU": "www.google.com.au", // Australia
		"CA": "www.google.ca",     // Canada
		"GB": "www.google.co.uk",  // United Kingdom
		"ID": "www.google.co.id",  // Indonesia
		"IE": "www.google.ie",     // Ireland
		"IN": "www.google.co.in",  // India
		"MY": "www.google.com.my", // Malaysia
		"NZ": "www.google.co.nz",  // New Zealand
		"PH": "www.google.com.ph", // Philippines
		"SG": "www.google.com.sg", // Singapore
		// "US": "www.google.us",  // United States, redirect to .com
		"ZA": "www.google.co.za",  // South Africa
		"AR": "www.google.com.ar", // Argentina
		"CL": "www.google.cl",     // Chile
		"ES": "www.google.es",     // Spain
		"MX": "www.google.com.mx", // Mexico
		"EE": "www.google.ee",     // Estonia
		"FI": "www.google.fi",     // Finland
		"BE": "www.google.be",     // Belgium
		"FR": "www.google.fr",     // France
		"IL": "www.google.co.il",  // Israel
		"HR": "www.google.hr",     // Croatia
		"HU": "www.google.hu",     // Hungary
		"IT": "www.google.it",     // Italy
		"JP": "www.google.co.jp",  // Japan
		"KR": "www.google.co.kr",  // South Korea
		"LT": "www.google.lt",     // Lithuania
		"LV": "www.google.lv",     // Latvia
		"NO": "www.google.no",     // Norway
		"NL": "www.google.nl",     // Netherlands
		"PL": "www.google.pl",     // Poland
		"BR": "www.google.com.br", // Brazil
		"PT": "www.google.pt",     // Portugal
		"RO": "www.google.ro",     // Romania
		"RU": "www.google.ru",     // Russia
		"SK": "www.google.sk",     // Slovakia
		"SI": "www.google.si",     // Slovenia
		"SE": "www.google.se",     // Sweden
		"TH": "www.google.co.th",  // Thailand
		"TR": "www.google.com.tr", // Turkey
		"UA": "www.google.com.ua", // Ukraine
		// "CN": "www.google.cn",  // China, only from China ?
		"HK": "www.google.com.hk", // Hong Kong
		"TW": "www.google.com.tw", // Taiwan
	}
)

func New() *Service {
	return &Service{
		HTTPClient: providers.NewHTTPClient(""),
	}
}

type Service struct {
	providers.HTTPClient

	UseLocalDomain bool

	lang struct {
		sync.RWMutex
		list []search.Language
	}
}

func (s *Service) Search(ctx context.Context, req search.Request) search.ResultIterator {
	r := SearchReq{
		Query:    req.Query,
		Language: req.Lang.Country().String(),
		Country:  req.Country.String(),
		Offset:   0,
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
		return search.Result{}
	}
	r := it.page[it.i]
	u, err := url.Parse(r.URL)
	if err != nil {
		it.err = err
		return search.Result{}
	}
	return search.Result{
		URL: *u, Title: r.Title, Text: r.Content,
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
	Results []Result
}

func (s *Service) SearchRaw(ctx context.Context, r SearchReq) (*SearchResp, error) {
	if r.Country == "" {
		r.Country = defaultCountry
		// FIXME: derive country from the language
	}
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

	base := "https://" + hostname
	req, err := s.GetRequest(base+searchPath, params)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept-Language", r.Language+","+r.Language+"-"+r.Country)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.AddCookie(&http.Cookie{
		Name: "GOOGLE_ABUSE_EXEMPTION", Value: "x",
	})
	resp, err := s.DoRaw(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// FIXME: handle "sorry" page
	var rd io.Reader = resp.Body
	if false {
		rd = io.TeeReader(rd, os.Stderr)
	}
	doc, err := goquery.NewDocumentFromReader(rd)
	if err != nil {
		return nil, err
	}
	// FIXME: parse instant answers
	out := &SearchResp{}
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

func (s *Service) fetchLanguages(ctx context.Context) ([]search.Language, error) {
	doc, err := s.GetHTML(ctx, languagesURL, nil)
	if err != nil {
		return nil, err
	}
	var out []search.Language
	doc.Find(`input[name="lang"]`).Each(func(_ int, sel *goquery.Selection) {
		code := sel.AttrOr("id", "")
		if code == "" {
			return
		}
		code = code[1:]
		title := sel.AttrOr("data-name", "")
		if title == "" {
			return
		}
		l, er := search.ParseLangCode(code)
		if er != nil {
			err = er
			return
		}
		out = append(out, search.Language{
			Code: l, Name: title,
		})
	})
	if err != nil {
		return nil, err
	} else if len(out) == 0 {
		return nil, fmt.Errorf("cannot parse languages list")
	}
	return out, nil
}

func (s *Service) Languages(ctx context.Context) ([]search.Language, error) {
	s.lang.RLock()
	list := s.lang.list
	s.lang.RUnlock()
	if list != nil {
		return append([]search.Language{}, list...), nil
	}
	s.lang.Lock()
	defer s.lang.Unlock()
	if list = s.lang.list; list != nil {
		return append([]search.Language{}, list...), nil
	}
	list, err := s.fetchLanguages(ctx)
	if err != nil {
		return nil, err
	}
	s.lang.list = list
	return append([]search.Language{}, list...), nil
}
