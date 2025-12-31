package model

import "github.com/doug-benn/dux/internal/database/queries"

type CategoryGroup struct {
	Name  string
	Links []queries.Link
}
