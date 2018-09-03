package wikipedia

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/nwca/metasearch/providers"
	"github.com/nwca/metasearch/search"
)

var _ search.Service = (*Service)(nil)

func New() *Service {
	return &Service{
		HTTPClient: providers.NewHTTPClient(""),
	}
}

var (
	DefaultThumbnailSize = 300
)

type Service struct {
	providers.HTTPClient
}

func (s *Service) Languages(ctx context.Context) ([]search.Language, error) {
	return nil, nil // FIXME
}

func (s *Service) Search(ctx context.Context, req search.Request) search.ResultIterator {
	r := SearchReq{
		Language: req.Lang.Country().String(),
		Titles:   req.Query,
		Prop: []Property{
			PropExtracts,
			PropPageImages,
		},
	}
	if r.Language == "" {
		r.Language = "en"
	}
	return &searchIter{s: s, cur: r}
}

func (s *Service) ContinueSearch(ctx context.Context, tok search.Token) search.ResultIterator {
	var t token
	if err := json.Unmarshal([]byte(tok), &t); err != nil {
		return &searchIter{err: err}
	}
	resp, err := s.SearchRaw(ctx, t.Cur)
	return &searchIter{s: s, cur: t.Cur, page: resp.Query.Pages, i: t.Off, err: err}
}

type searchIter struct {
	s   *Service
	cur SearchReq

	page []Page
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
	if len(it.page) == 0 {
		resp, err := it.s.SearchRaw(ctx, it.cur)
		if err != nil {
			it.err = err
			return false
		}
		it.page = resp.Query.Pages
		return len(it.page) > 0
	}
	// FIXME: query the next page
	return false
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
	id := strings.Replace(r.Title, " ", "_", -1)
	surl := "https://" + it.cur.Language + ".wikipedia.org/wiki/" + url.PathEscape(id)
	u, err := url.Parse(surl)
	if err != nil {
		it.err = err
		return nil
	}
	res := &search.EntityResult{
		LinkResult: search.LinkResult{
			URL:   *u,
			Title: r.Title,
			Desc:  r.Extract,
		},
	}
	if t := r.Thumbnail; t != nil && t.Source != "" {
		u, err := url.Parse(t.Source)
		if err != nil {
			it.err = err
			return nil
		}
		res.Image = &search.Image{
			URL: *u, Width: t.Width, Height: t.Height,
		}
	}
	return res
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
	Language  string     `json:"lang"`
	Titles    string     `json:"titles"`
	Prop      []Property `json:"props"`
	ThumbSize int        `json:"thumb_size"`
}

func (r *SearchReq) includesProp(p Property) bool {
	for _, p2 := range r.Prop {
		if p == p2 {
			return true
		}
	}
	return false
}

type Property string

const (
	PropExtracts   = Property("extracts")
	PropPageImages = Property("pageimages")
)

type SearchResp struct {
	Warnings map[string]struct {
		Text string `json:"warnings"`
	} `json:"warnings"`
	BatchComplete bool          `json:"batchcomplete"`
	Query         QueryResponse `json:"query"`
}

type QueryResponse struct {
	Pages []Page `json:"pages"`
}

type Page struct {
	ID        int64  `json:"pageid"`
	NS        int    `json:"ns"`
	Title     string `json:"title"`
	Extract   string `json:"extract"`
	Thumbnail *Image `json:"thumbnail"`
	PageImage string `json:"pageimage"`
}

type Image struct {
	Source string `json:"source"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

func (s *Service) SearchRaw(ctx context.Context, r SearchReq) (*SearchResp, error) {
	if r.Language == "" {
		r.Language = "en"
	}
	if r.ThumbSize == 0 {
		r.ThumbSize = DefaultThumbnailSize
	}

	params := make(url.Values)
	params.Set("titles", r.Titles)
	params.Set("action", "query")
	params.Set("format", "json")
	params.Set("formatversion", "2")
	if len(r.Prop) != 0 {
		arr := make([]string, 0, len(r.Prop))
		for _, p := range r.Prop {
			arr = append(arr, string(p))
		}
		params.Set("prop", strings.Join(arr, "|"))
	}
	if r.includesProp(PropExtracts) {
		params.Set("exintro", "")
		params.Set("explaintext", "")
	}
	if r.includesProp(PropPageImages) {
		params.Set("pithumbsize", strconv.Itoa(r.ThumbSize))
	}
	params.Set("redirects", "")

	var out SearchResp
	err := s.GetJSON(ctx, "https://"+r.Language+".wikipedia.org/w/api.php", params, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
