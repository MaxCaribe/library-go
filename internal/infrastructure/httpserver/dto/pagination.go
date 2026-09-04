package dto

import (
	"net/http"
	"strconv"
)

const (
	defaultPageSize = 10
	maxPageSize     = 100
)

func ParsePagination(r *http.Request) (int, int) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

// PaginatedResponse is the shape of every collection response. Data is generic
// so each resource reuses it; PaginationMeta is embedded rather than restated
// so its fields stay generated from the spec.
type PaginatedResponse[T any] struct {
	Data []T `json:"data"`
	PaginationMeta
}

func NewPaginatedResponse[T any](data []T, total, page, pageSize int) PaginatedResponse[T] {
	if data == nil {
		data = []T{}
	}
	return PaginatedResponse[T]{
		Data: data,
		PaginationMeta: PaginationMeta{
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: TotalPages(total, pageSize),
		},
	}
}

func ComputePagination(page, pageSize int) (limit, offset int) {
	return pageSize, (page - 1) * pageSize
}

func TotalPages(total, pageSize int) int {
	if pageSize < 1 {
		return 0
	}
	return (total + pageSize - 1) / pageSize
}
