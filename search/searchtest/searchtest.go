package searchtest

import (
	"context"
	"strings"
	"testing"

	"github.com/dennwc/metasearch/search"
	"github.com/stretchr/testify/require"
)

type Config struct {
	Safe bool
}

func RunSearchTest(t *testing.T, s search.Searcher, c *Config) {
	if c == nil {
		c = &Config{}
	}
	t.Run("lang", func(t *testing.T) {
		testSearchLang(t, s)
	})
	t.Run("safe", func(t *testing.T) {
		if !c.Safe {
			t.SkipNow()
		}
		testSearchSafe(t, s)
	})
}

func PageContains(it search.ResultIterator, text string) bool {
	text = strings.ToLower(text)
	ctx := context.Background()
	n := it.Buffered()
	for i := 0; i < n && it.Next(ctx); i++ {
		r := it.Result()
		if strings.Contains(strings.ToLower(r.GetTitle()), text) ||
			strings.Contains(strings.ToLower(r.GetDesc()), text) {
			return true
		}
	}
	return false
}

func testSearchLang(t *testing.T, s search.Searcher) {
	ctx := context.Background()

	it := s.Search(ctx, search.Request{
		Query: "solar",
		Lang:  search.MustParseLangCode("de-de"),
	})
	defer it.Close()
	require.True(t, it.NextPage(ctx))
	require.True(t, PageContains(it, "die Sonne"))
}

func testSearchSafe(t *testing.T, s search.Searcher) {
	ctx := context.Background()

	const (
		ambigous     = "scissoring"
		safeTerm     = "OpenGL"
		explicitTerm = "lesbian"
	)

	it1 := s.Search(ctx, search.Request{
		Query: ambigous,
		Safe:  true,
	})
	defer it1.Close()
	require.True(t, it1.NextPage(ctx))
	require.True(t, PageContains(it1, safeTerm))

	it2 := s.Search(ctx, search.Request{
		Query: ambigous,
		Safe:  true,
	})
	defer it2.Close()
	require.True(t, it2.NextPage(ctx))
	require.True(t, !PageContains(it2, explicitTerm))

	it3 := s.Search(ctx, search.Request{
		Query: ambigous,
	})
	defer it3.Close()
	require.True(t, it3.NextPage(ctx))
	require.True(t, PageContains(it3, explicitTerm))
}
