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
)

const baseURL = "https://duckduckgo.com"

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

type SearchReq struct {
	Language string
	Query    string
	Offset   int
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

func (s *Service) getRegionCode(ctx context.Context, lang string) string {
	if lang == "" {
		return defaultRegion
	}
	list, err := s.Regions(ctx)
	if err != nil {
		return defaultRegion
	}
	for _, r := range list {
		// TODO: match correctly
		if strings.Contains(r.Code, lang) {
			return r.Code
		}
	}
	return defaultRegion
}

type Result struct {
	URL     string
	Title   string
	Content string
}

func (s *Service) SearchRaw(ctx context.Context, r SearchReq) (*SearchResp, error) {
	region := s.getRegionCode(ctx, r.Language)
	params := make(url.Values)
	params.Set("q", r.Query)
	params.Set("kl", region)
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
