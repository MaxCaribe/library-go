package dto

import (
	"net/http"
	"strconv"
)

const (
	defaultPageSize = 10
	maxPageSize     = 100
)

type PaginatedResponse[T any] struct {
	Data       []T `json:"data"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalPages int `json:"total_pages"`
}

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

func ComputePagination(page, pageSize int) (limit, offset int) {
	return pageSize, (page - 1) * pageSize
}

func NewPaginatedResponse[T any](data []T, total, page, pageSize int) PaginatedResponse[T] {
	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 0 {
		totalPages = 0
	}
	return PaginatedResponse[T]{
		Data:       data,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}
