package router

import (
	"log/slog"
	"net/http"

	"github.com/doug-benn/dux/internal/database"
	"github.com/doug-benn/dux/internal/middleware"
	"github.com/patrickmn/go-cache"
)

type GlobalState struct {
	Count int
}

var global GlobalState

func AddRoutes(
	mux *http.ServeMux,
	logger *slog.Logger,
	cache *cache.Cache,
	database database.Database,
) {

	//CRUD
	mux.Handle("POST /add", HandleAddLink(logger, cache, database))
	mux.Handle("POST /log_click", HandleLinkClicked(logger, cache, database))

	//UI components
	mux.Handle("/{$}", HandleRoot(logger, cache, database))

	mux.Handle("GET /add_form", HandleModal(logger, cache, database))
	mux.Handle("GET /edit_form", HandleEditModal(logger, cache, database))

	//Search
	mux.Handle("POST /search", HandleSearch(logger, cache, database))

	//Lock and unlock the UI
	mux.Handle("POST /unlock", HandleUnlock(logger, cache, database))
	mux.Handle("POST /lock", HandleLock(logger, cache, database))

	//Static files
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", middleware.CacheHeaderMiddleware(http.FileServer(http.Dir("./assets")))))

	mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))

	// System Routes for debugging
	mux.Handle("GET /health", HandleGetHealth())

}
