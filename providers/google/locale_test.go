package google

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLanguages(t *testing.T) {
	s := New()
	ctx := context.Background()
	list, err := s.Languages(ctx)
	require.NoError(t, err)
	t.Logf("%d %q", len(list), list)
	require.True(t, len(list) >= 140)
}
