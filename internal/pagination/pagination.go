package pagination

import (
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
)

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

type Params struct {
	Enabled bool
	Page    int
	Limit   int
	Offset  int
}

type Metadata struct {
	Page        int   `json:"page"`
	Limit       int   `json:"limit"`
	Total       int64 `json:"total"`
	TotalPages  int   `json:"total_pages"`
	HasNext     bool  `json:"has_next"`
	HasPrevious bool  `json:"has_previous"`
}

func Parse(values url.Values) (Params, error) {
	pageProvided := values.Has("page")
	limitProvided := values.Has("limit")
	if !pageProvided && !limitProvided {
		return Params{}, nil
	}

	params := Params{Enabled: true, Page: 1, Limit: DefaultLimit}
	if pageProvided {
		page, err := strconv.Atoi(strings.TrimSpace(values.Get("page")))
		if err != nil || page < 1 {
			return Params{}, fmt.Errorf("page deve ser maior ou igual a 1")
		}
		params.Page = page
	}
	if limitProvided {
		limit, err := strconv.Atoi(strings.TrimSpace(values.Get("limit")))
		if err != nil || limit < 1 || limit > MaxLimit {
			return Params{}, fmt.Errorf("limit deve estar entre 1 e %d", MaxLimit)
		}
		params.Limit = limit
	}
	if params.Page-1 > math.MaxInt/params.Limit {
		return Params{}, fmt.Errorf("pagina solicitada e muito alta")
	}
	params.Offset = (params.Page - 1) * params.Limit
	return params, nil
}

func NewMetadata(params Params, total int64) Metadata {
	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(params.Limit) - 1) / int64(params.Limit))
	}
	return Metadata{
		Page:        params.Page,
		Limit:       params.Limit,
		Total:       total,
		TotalPages:  totalPages,
		HasNext:     params.Page < totalPages,
		HasPrevious: params.Page > 1,
	}
}
