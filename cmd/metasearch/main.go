package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nwca/metasearch"
	_ "github.com/nwca/metasearch/providers/all"
	"github.com/nwca/metasearch/search"
)

var Root = &cobra.Command{
	Use:   "metasearch",
	Short: "runs a metasearch engine",
}

func init() {
	cmdQuery := &cobra.Command{
		Use:     "query",
		Aliases: []string{"qu", "q"},
		RunE: func(cmd *cobra.Command, args []string) error {
			qu := strings.Join(args, " ")
			ctx := context.Background()
			s, err := metasearch.NewEngine(ctx)
			if err != nil {
				return err
			}
			limit, _ := cmd.Flags().GetInt("limit")

			it := s.Search(ctx, search.Request{
				Query: qu,
			})
			defer it.Close()
			for i := 0; i < limit && it.Next(ctx); i++ {
				fmt.Printf("%v\n\n", it.Result())
			}
			if err := it.Err(); err != nil {
				return err
			}
			tok := it.Token()
			fmt.Println("\n\ntoken:", hex.EncodeToString([]byte(tok)))
			return nil
		},
	}
	cmdQuery.Flags().IntP("limit", "n", 10, "limit the number of results")
	Root.AddCommand(cmdQuery)
}

func main() {
	if err := Root.Execute(); err != nil {
		log.Fatal(err)
	}
}
