package router

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/doug-benn/dux/internal/components"
	"github.com/doug-benn/dux/internal/database"
	"github.com/patrickmn/go-cache"
)

func HandleModal(logger *slog.Logger, cache *cache.Cache, database database.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := components.Modal(components.AddLinkForm()).Render(r.Context(), w); err != nil {
			logger.Error("Failed to render component", "error", err)
		}

	}
}

func HandleEditModal(logger *slog.Logger, cache *cache.Cache, database database.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		idStr := r.URL.Query().Get("id")
		editId, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || editId == 0 {
			http.Error(w, "Invalid Link ID", http.StatusBadRequest)
			return
		}

		link, err := database.Queries().QueryLink(r.Context(), editId)
		if err != nil {
			http.Error(w, "Link not found", http.StatusNotFound)
			return
		}

		if err := components.Modal(components.EditLinkForm(link)).Render(ctx, w); err != nil {
			logger.Error("Failed to render component", "error", err)
		}

	}
}
