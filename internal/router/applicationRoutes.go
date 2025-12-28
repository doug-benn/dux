package router

import (
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/doug-benn/dux/internal/components"
	"github.com/doug-benn/dux/internal/database"
	"github.com/doug-benn/dux/internal/database/queries"
	"github.com/doug-benn/dux/internal/model"
	"github.com/patrickmn/go-cache"
)

func HandleRoot(logger *slog.Logger, cache *cache.Cache, database database.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		links, err := database.Queries().ListLinks(r.Context())
		if err != nil {
			logger.Error("failed to fetch category data", "error", err)
		}

		groupedLinks := GroupAndSortLinks(links)

		if err := components.HTML(groupedLinks).Render(r.Context(), w); err != nil {
			logger.Error("Failed to render component", "error", err)
		}
	}
}

func HandleCreateLink(logger *slog.Logger, cache *cache.Cache, database database.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Parse multipart form (max 10MB)
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "Form too large", http.StatusBadRequest)
			return
		}

		// 2. Extract Text Fields
		name := r.FormValue("name")
		url := r.FormValue("url")
		category := sql.NullString{
			String: r.FormValue("category"),
			Valid:  r.FormValue("category") != "",
		}
		colour := sql.NullString{
			String: r.FormValue("colour"),
			Valid:  r.FormValue("colour") != "",
		}

		if name == "" || url == "" {
			http.Error(w, "Name and URL are required", http.StatusBadRequest)
			return
		}

		// 3. Handle File Upload (Optional)
		var iconPath sql.NullString
		file, handler, err := r.FormFile("icon")

		if err == nil {
			defer file.Close()

			// Create unique filename and save to disk
			fileName := fmt.Sprintf("%d-%s", time.Now().Unix(), handler.Filename)
			fullPath := filepath.Join("./uploads/", fileName)

			dst, err := os.Create(fullPath)
			if err != nil {
				logger.Error("failed to save icon", "error", err)
				http.Error(w, "Failed to save icon", http.StatusInternalServerError)
				return
			}
			defer dst.Close()
			io.Copy(dst, file)

			iconPath = sql.NullString{String: fullPath, Valid: true}
		}

		// 4. Single Database Insert
		// Uses the COALESCE logic we defined in your SQL earlier
		_, err = database.Queries().CreateLink(r.Context(), queries.CreateLinkParams{
			Name:     name,
			Url:      url,
			Icon:     iconPath,
			Category: category,
			Colour:   colour,
		})

		if err != nil {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		links, err := database.Queries().ListLinks(r.Context())
		if err != nil {
			logger.Error("failed to fetch category data", "error", err)
		}

		groupedLinks := GroupAndSortLinks(links)

		if err := components.Links(groupedLinks, true).Render(r.Context(), w); err != nil {
			logger.Error("Failed to render component", "error", err)
		}
	}
}

func HandleLinkClicked(logger *slog.Logger, cache *cache.Cache, database database.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		linkID, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
		if err != nil {
			// If the ID isn't a valid number, log the error and stop
			logger.Error("Invalid link ID received", "id", r.FormValue("id"))
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if err := database.Queries().RecordHit(r.Context(), linkID); err != nil {
			http.Error(w, "Failed to record link hit", http.StatusInternalServerError)
			logger.Error("failed to record link hit", "error", err)
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

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

func HandleModal(logger *slog.Logger, cache *cache.Cache, database database.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := components.Modal().Render(r.Context(), w); err != nil {
			logger.Error("Failed to render component", "error", err)
		}

	}
}

func GroupAndSortLinks(links []queries.Link) []model.CategoryGroup {
	if len(links) == 0 {
		return nil
	}

	sort.Slice(links, func(i, j int) bool {
		catI := getCat(links[i])
		catJ := getCat(links[j])

		if catI != catJ {
			if catI == "Other" {
				return false
			}
			if catJ == "Other" {
				return true
			}
			return catI < catJ
		}
		return strings.ToLower(links[i].Name) < strings.ToLower(links[j].Name)
	})

	var groups []model.CategoryGroup
	for _, link := range links {
		catName := getCat(link)

		if len(groups) == 0 || groups[len(groups)-1].Name != catName {
			groups = append(groups, model.CategoryGroup{
				Name:  catName,
				Links: []queries.Link{link},
			})
		} else {
			groups[len(groups)-1].Links = append(groups[len(groups)-1].Links, link)
		}
	}

	return groups
}

func getCat(l queries.Link) string {
	name := strings.TrimSpace(l.Category.String)
	if !l.Category.Valid || name == "" {
		return "Other"
	}
	return name
}
