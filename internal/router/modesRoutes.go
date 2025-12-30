package router

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/doug-benn/dux/internal/components"
	"github.com/doug-benn/dux/internal/database"
	"github.com/patrickmn/go-cache"
)

func HandleUnlock(logger *slog.Logger, cache *cache.Cache, database database.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		pin := r.Header.Get("HX-Prompt")

		//TODO configurable pin number
		if pin != "1234" {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, "Invalid PIN")
			return
		}

		links, err := database.Queries().ListLinks(r.Context())
		if err != nil {
			logger.Error("failed to fetch category data", "error", err)
		}

		groupedLinks := GroupAndSortLinks(links)

		if err := components.UI(groupedLinks, true).Render(r.Context(), w); err != nil {
			logger.Error("Failed to render component", "error", err)
		}

	}
}

func HandleLock(logger *slog.Logger, cache *cache.Cache, database database.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		links, err := database.Queries().ListLinks(r.Context())
		if err != nil {
			logger.Error("failed to fetch category data", "error", err)
		}

		groupedLinks := GroupAndSortLinks(links)

		if err := components.UI(groupedLinks, false).Render(r.Context(), w); err != nil {
			logger.Error("Failed to render component", "error", err)
		}

	}
}
