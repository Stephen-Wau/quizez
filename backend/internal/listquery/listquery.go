// Package listquery menyeragamkan cara parse query param search, sort, & pagination buat semua
// endpoint list yang FE-nya pakai DataTableComponent (search box, klik header buat sort, pager).
// Kontrak param standar: ?searchword=...&sort_by=...&sort_dir=asc|desc&page=1&per_page=10
// Response standar: {"data": [...], "meta": {"page", "per_page", "total", "total_pages"}}
package listquery

import (
	"net/http"
	"strconv"
	"strings"
)

const (
	defaultPerPage = 10
	maxPerPage     = 100
)

type Params struct {
	SearchWord string
	SortBy     string
	SortDir    string // selalu "asc" atau "desc", default "asc" kalau kosong/invalid
	Page       int    // 1-indexed, minimal 1
	PerPage    int    // minimal 1, dibatasi maxPerPage
}

// Parse baca query param standar (searchword, sort_by, sort_dir, page, per_page) dari request,
// sekalian normalize/clamp nilai yang gak valid biar handler gak perlu validasi ulang.
func Parse(r *http.Request) Params {
	q := r.URL.Query()

	dir := strings.ToLower(strings.TrimSpace(q.Get("sort_dir")))
	if dir != "asc" && dir != "desc" {
		dir = "asc"
	}

	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}

	perPage, _ := strconv.Atoi(q.Get("per_page"))
	if perPage < 1 {
		perPage = defaultPerPage
	} else if perPage > maxPerPage {
		perPage = maxPerPage
	}

	return Params{
		SearchWord: strings.TrimSpace(q.Get("searchword")),
		SortBy:     strings.TrimSpace(q.Get("sort_by")),
		SortDir:    dir,
		Page:       page,
		PerPage:    perPage,
	}
}

// SortColumn map sort_by dari FE ke nama kolom SQL asli lewat whitelist `allowed`, biar gak ada
// celah SQL injection dari raw sort_by. Balik ke defaultCol kalau sort_by gak dikenal/kosong.
func (p Params) SortColumn(allowed map[string]string, defaultCol string) string {
	if col, ok := allowed[p.SortBy]; ok {
		return col
	}
	return defaultCol
}

// SortDirSQL konversi SortDir ke keyword SQL yang valid buat ORDER BY.
func (p Params) SortDirSQL() string {
	if p.SortDir == "desc" {
		return "DESC"
	}
	return "ASC"
}

// Offset hitung offset SQL dari Page & PerPage buat dipakai di klausa LIMIT/OFFSET.
func (p Params) Offset() int {
	return (p.Page - 1) * p.PerPage
}

type Meta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// BuildMeta susun object meta pagination buat response FE (page, per_page, total, total_pages).
func BuildMeta(p Params, total int) Meta {
	totalPages := 1
	// Guard div-by-zero & kasus data kosong; default totalPages 1 biar FE gak nampilin "page 0 dari 0".
	if p.PerPage > 0 && total > 0 {
		totalPages = (total + p.PerPage - 1) / p.PerPage
	}
	return Meta{Page: p.Page, PerPage: p.PerPage, Total: total, TotalPages: totalPages}
}

type ListResponse[T any] struct {
	Data []T  `json:"data"`
	Meta Meta `json:"meta"`
}
