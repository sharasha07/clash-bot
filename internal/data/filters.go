package data

import "strings"

type Filters struct {
	Page     int `validate:"min=1"`
	PageSize int `validate:"min=0,max=10"`
	Sort     string
}

func (f Filters) Offset() int {
	return (f.Page - 1) * f.PageSize
}

func (f Filters) SortOrder() string {
	if strings.HasPrefix(f.Sort, "-") {
		return "DESC"
	}

	return "ASC"
}

func (f Filters) SortColumn() string {
	return strings.TrimPrefix(f.Sort, "-")
}
