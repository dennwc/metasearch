package duckduckgo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"sort"
	"strings"

	"github.com/dennwc/metasearch/search"
)

var langAliases = map[string]string{
	"jp-jp": "jp-ja",
}

const (
	defaultRegion regionCode = "us-en"
	allRegions    regionCode = "wt-wt"
)

type regionCode string

func (c regionCode) toLang() string {
	s := string(c)
	if c, ok := langAliases[s]; ok {
		s = c
	}
	i := strings.Index(s, "-")
	if i < 0 {
		panic(s)
	}
	return s[i+1:] + "-" + s[:i]
}

func toRegion(s string) regionCode {
	s = strings.ToLower(s)
	i := strings.Index(s, "-")
	if i < 0 {
		panic(s)
	}
	return regionCode(s[i+1:] + "-" + s[:i])
}

type region struct {
	Code regionCode
	Name string
}

func (s *Service) fetchRegions(ctx context.Context) ([]region, error) {
	resp, err := s.Get(ctx, baseURL+"/util/u172.js", nil)
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
	out := make([]region, 0, len(regions))
	for code, name := range regions {
		out = append(out, region{Code: regionCode(code), Name: name})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Code < out[j].Code
	})
	return out, nil
}

func (s *Service) Languages(ctx context.Context) ([]search.Language, error) {
	list, err := s.fetchRegions(ctx)
	if err != nil {
		return nil, err
	}
	langs := make([]search.Language, 0, len(list))
	for _, r := range list {
		if r.Code == allRegions {
			continue
		}
		code := r.Code.toLang()
		c, err := search.ParseLangCode(code)
		if err != nil {
			return nil, fmt.Errorf("lang %q %q: %v", r.Name, code, err)
		}
		langs = append(langs, search.Language{
			Name: r.Name,
			Code: c,
		})
	}
	return langs, nil
}

func (s *Service) Regions(ctx context.Context) ([]search.Region, error) {
	// caller should call Languages
	return nil, nil // TODO: load from DDG?
}
