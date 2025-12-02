package queryhelper

import (
	"gorm.io/gorm"
)

const (
	DefaultPage        = 1
	DefaultPageSize    = 10
	DefaultMaxPageSize = 100
)

type PaginationRequest struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

type PaginationInfo struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type PaginationHandle struct {
	Info *PaginationInfo `json:"info"`
}

func NewPaginationHandle(req *PaginationRequest) *PaginationHandle {

	if req == nil {
		req = &PaginationRequest{}
	}

	if req.Page <= 0 {
		req.Page = DefaultPage
	}

	if req.PageSize <= 0 {
		req.PageSize = DefaultPageSize
	}

	if req.PageSize > DefaultMaxPageSize {
		req.PageSize = DefaultMaxPageSize
	}

	return &PaginationHandle{
		Info: &PaginationInfo{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
	}
}

func (p *PaginationHandle) Page() int {
	return p.Info.Page
}

func (p *PaginationHandle) PageSize() int {
	return p.Info.PageSize
}

func (p *PaginationHandle) Offset() int {
	return (p.Page() - 1) * p.PageSize()
}

func (p *PaginationHandle) TotalPages() int {
	return p.Info.TotalPages
}

func (p *PaginationHandle) Total() int64 {
	return p.Info.Total
}

func (p *PaginationHandle) Apply(query *gorm.DB) (*gorm.DB, error) {

	if query == nil {
		return nil, nil
	}

	// Count total records for current query
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return query, err
	}

	p.Info.Total = total
	if total == 0 {
		p.Info.TotalPages = 1
	} else {
		p.Info.TotalPages = int((total + int64(p.Info.PageSize) - 1) / int64(p.Info.PageSize))
	}

	// Apply offset and limit
	query = query.
		Offset(p.Offset()).
		Limit(p.Info.PageSize)

	return query, nil
}

func (p *PaginationHandle) CurrentInfo() *PaginationInfo {
	return p.Info
}
