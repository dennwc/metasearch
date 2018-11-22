package duckduckgo

import (
	"context"
	"strings"
	"testing"

	"github.com/dennwc/metasearch/search"
	"github.com/stretchr/testify/require"
)

func TestSearchRaw(t *testing.T) {
	s := New()
	ctx := context.Background()

	req := SearchReq{
		Query: "solar",
	}
	resp, next, err := s.search(ctx, s.newSearch(req))
	require.NoError(t, err)
	t.Logf("%d %q", len(resp.Results), resp)
	require.True(t, len(resp.Results) > 2)

	r := resp.Results[0]
	require.True(t, r.URL != "" && r.Title != "" && r.Content != "")
	require.True(t, strings.HasPrefix(r.URL, "http"))

	require.True(t, len(next) != 0)
}

func TestSearch(t *testing.T) {
	s := New()
	ctx := context.Background()

	it := s.Search(ctx, search.Request{Query: "solar"})
	defer it.Close()

	const perPage = 30

	seen := make(map[string][2]int)
	dups := 0
	for i := 0; i < 2; i++ {
		if !it.NextPage(ctx) {
			require.Fail(t, "expected more pages")
		}
		require.NoError(t, it.Err())

		n := it.Buffered()
		require.True(t, n >= perPage)
		if n > perPage {
			t.Log("unexpectedly large page:", n)
		}

		for j := 0; j < n; j++ {
			if !it.Next(ctx) {
				require.Fail(t, "expected more results")
			}
			require.Equal(t, n-(j+1), it.Buffered())

			r := it.Result()
			require.NotNil(t, r)

			ind := [2]int{i, j}
			url := r.GetURL().String()
			old, ok := seen[url]
			if ok {
				dups++
				t.Logf("already seen this url: %v vs %v (%q)", old, ind, url)
			} else {
				seen[url] = ind
			}
		}
		require.Equal(t, 0, it.Buffered())
	}
	require.True(t, dups <= 3)
}
