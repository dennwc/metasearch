package duckduckgo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAutoComplete(t *testing.T) {
	s := New()
	list, err := s.AutoComplete(context.TODO(), "sola")
	require.NoError(t, err)
	require.NotEmpty(t, list)

	t.Logf("%d results: %q", len(list), list)
	for _, v := range list {
		require.True(t, v != "", "empty item")
	}
}
