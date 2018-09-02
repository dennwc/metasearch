package wikipedia

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"github.com/nwca/metasearch/providers"
)

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

type SearchReq struct {
	Language  string
	Titles    string
	Prop      []Property
	ThumbSize int
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
