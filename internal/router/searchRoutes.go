package router

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/doug-benn/dux/internal/components"
	"github.com/doug-benn/dux/internal/database"
	"github.com/doug-benn/dux/internal/database/queries"
	"github.com/patrickmn/go-cache"
)

func HandleSearch(logger *slog.Logger, cache *cache.Cache, database database.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		searchTerm := r.FormValue("search")

		fmt.Println(searchTerm)

		var links []queries.Link
		var err error

		if searchTerm == "" {
			links, err = database.Queries().ListLinks(ctx)
		} else {
			links, err = database.Queries().SearchLinks(ctx, sql.NullString{
				String: searchTerm,
				Valid:  true,
			})
		}

		if err != nil {
			http.Error(w, "Search failed", http.StatusInternalServerError)
			logger.Error("failed to search", "error", err)
			return
		}

		groupedLinks := GroupAndSortLinks(links)

		if err := components.Links(groupedLinks, false).Render(ctx, w); err != nil {
			logger.Error("Failed to render component", "error", err)
		}
	}
}
