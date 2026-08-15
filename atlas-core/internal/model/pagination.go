package model

import (
	"net/url"
	"strconv"
)

type Pagination struct {
	Limit  int
	Offset int
}

const (
	DefaultLimit = 50
	MaxLimit     = 100
)

func ParsePagination(values url.Values) (Pagination, *CoreError) {
	pagination := Pagination{Limit: DefaultLimit, Offset: 0}
	if raw := values.Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > MaxLimit {
			return Pagination{}, ValidationError(FieldError{Field: "limit", Code: "out_of_range", Message: "limit must be between 1 and 100"})
		}
		pagination.Limit = parsed
	}
	if raw := values.Get("offset"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			return Pagination{}, ValidationError(FieldError{Field: "offset", Code: "out_of_range", Message: "offset must be 0 or greater"})
		}
		pagination.Offset = parsed
	}
	return pagination, nil
}
