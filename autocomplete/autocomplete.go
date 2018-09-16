package autocomplete

import (
	"context"

	"github.com/dennwc/metasearch/base"
)

type Service interface {
	base.Provider
	AutoComplete(ctx context.Context, text string) ([]string, error)
}
