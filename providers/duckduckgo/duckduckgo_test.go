package duckduckgo

import (
	"context"
	"github.com/nwca/metasearch/search"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegions(t *testing.T) {
	s := New()
	ctx := context.Background()
	list, err := s.Regions(ctx)
	require.NoError(t, err)
	t.Logf("%d %q", len(list), list)
	require.True(t, len(list) >= 65)

	langs, err := s.Languages(ctx)
	require.NoError(t, err)
	t.Logf("%d %q", len(langs), langs)
	require.True(t, len(langs) == len(list))
}

func TestSearchRaw(t *testing.T) {
	s := New()
	ctx := context.Background()

	req := SearchReq{
		Query: "solar",
	}
	resp, err := s.SearchRaw(ctx, req)
	require.NoError(t, err)
	t.Logf("%d %q", len(resp.Results), resp)
	require.True(t, len(resp.Results) > 2)

	r := resp.Results[0]
	require.True(t, r.URL != "" && r.Title != "" && r.Content != "")
	require.True(t, strings.HasPrefix(r.URL, "http"))

	req.Offset += len(resp.Results)
	resp, err = s.SearchRaw(ctx, req)
	require.NoError(t, err)
	t.Logf("%d %q", len(resp.Results), resp)
	require.True(t, len(resp.Results) > 2)

	r2 := resp.Results[0]
	require.True(t, r2.URL != "" && r2.Title != "" && r2.Content != "")
	require.True(t, r.URL != r2.URL)

	it := s.Search(ctx, search.Request{Query: req.Query})
	defer it.Close()
	var got []search.Result
	for i := 0; i < perPage*2 && it.Next(ctx); i++ {
		got = append(got, it.Result())
	}
	require.NoError(t, it.Err())
	require.True(t, len(got) >= perPage*2 && len(got) <= (perPage+1)*2)
}
