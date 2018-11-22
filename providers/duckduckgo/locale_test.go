package duckduckgo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

const expectedLang = 65

func TestRegionsRaw(t *testing.T) {
	s := New()
	ctx := context.Background()

	list, err := s.fetchRegions(ctx)
	require.NoError(t, err)
	t.Logf("%d %q", len(list), list)
	require.True(t, len(list) >= expectedLang)

	left := map[region]struct{}{
		{Code: "be-fr", Name: "Belgium (fr)"}: {},
		{Code: "be-nl", Name: "Belgium (nl)"}: {},
	}
	for _, r := range list {
		delete(left, r)
	}
	require.Empty(t, left)
}

func TestLanguages(t *testing.T) {
	s := New()
	ctx := context.Background()

	list, err := s.Languages(ctx)
	require.NoError(t, err)
	t.Logf("%d %q", len(list), list)
	require.True(t, len(list) >= expectedLang-1) // -"All"
}
