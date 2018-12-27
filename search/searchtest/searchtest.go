package searchtest

import (
	"context"
	"strings"
	"testing"

	"github.com/dennwc/metasearch/search"
	"github.com/stretchr/testify/require"
)

func RunSearchTest(t *testing.T, s search.Searcher) {
	t.Run("lang", func(t *testing.T) {
		testSearchLang(t, s)
	})
}

func pageContains(it search.ResultIterator, text string) bool {
	ctx := context.Background()
	n := it.Buffered()
	for i := 0; i < n && it.Next(ctx); i++ {
		r := it.Result()
		if strings.Contains(r.GetTitle(), text) || strings.Contains(r.GetDesc(), text) {
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
	require.True(t, pageContains(it, "die Sonne"))
}
