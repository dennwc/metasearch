package duckduckgo

import (
	"context"
	"net/url"

	"github.com/dennwc/metasearch/autocomplete"
)

const (
	baseURLAuto = "https://ac.duckduckgo.com"
)

var (
	_ autocomplete.Service = (*Service)(nil)
)

func (s *Service) AutoComplete(ctx context.Context, text string) ([]string, error) {
	params := make(url.Values)
	params.Set("q", text)
	params.Set("type", "json")

	var list []struct {
		Text string `json:"phrase"`
	}
	if err := s.GetJSON(ctx, baseURLAuto+"/ac", params, &list); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(list))
	for _, v := range list {
		out = append(out, v.Text)
	}
	return out, nil
}
