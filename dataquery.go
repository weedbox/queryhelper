package queryhelper

import (
	"gorm.io/gorm"
)

type QueryHelperInfo struct {
	Pagination *PaginationInfo
	Conditions *QueryConditions
}

type QueryHelper struct {
	queryConditions   *QueryConditions
	paginationRequest *PaginationRequest
	pagination        *PaginationHandle
	conditions        *ConditionsHandle
}

type Option func(*QueryHelper)

func WithPage(page int) Option {
	return func(dq *QueryHelper) {
		dq.paginationRequest.Page = page
	}
}

func WithPageSize(pageSize int) Option {
	return func(dq *QueryHelper) {
		dq.paginationRequest.PageSize = pageSize
	}
}

func WithSearchText(text string) Option {
	return func(dq *QueryHelper) {
		dq.queryConditions.SearchText = text
	}
}

func WithSearchFields(fields []string) Option {
	return func(dq *QueryHelper) {
		dq.queryConditions.SearchFields = fields
	}
}

func WithOrderBy(fields []string) Option {
	return func(dq *QueryHelper) {
		dq.queryConditions.OrderBy = fields
	}
}

func WithSortFactor(factor int) Option {
	return func(dq *QueryHelper) {
		dq.queryConditions.SortFactor = factor
	}
}

func WithFilters(filters []FilterCondition) Option {
	return func(dq *QueryHelper) {
		dq.queryConditions.Filters = filters
	}
}

func NewQueryHelper(opts ...Option) *QueryHelper {

	dq := &QueryHelper{
		queryConditions:   &QueryConditions{},
		paginationRequest: &PaginationRequest{},
	}

	for _, opt := range opts {
		opt(dq)
	}

	// Initialize pagination handle
	dq.pagination = NewPaginationHandle(dq.paginationRequest)

	return dq
}

func (dq *QueryHelper) GetPaginationRequest() *PaginationRequest {
	return dq.paginationRequest
}

func (dq *QueryHelper) GetQueryConditions() *QueryConditions {
	return dq.queryConditions
}

func (dq *QueryHelper) Info() *QueryHelperInfo {
	return &QueryHelperInfo{
		Pagination: dq.pagination.CurrentInfo(),
		Conditions: dq.conditions.CurrentInfo(),
	}
}

func (dq *QueryHelper) Apply(settings *QuerySettings, query *gorm.DB) (*gorm.DB, error) {

	// Prepare dataquery handle
	dqh := NewConditionsHandle(settings)
	dqh.UpdateConditions(dq.queryConditions)

	// Apply conditions to query
	if query != nil {

		q, err := dqh.Apply(query)
		if err != nil {
			return nil, err
		}

		query = q
	}

	dq.conditions = dqh

	// Apply pagination to query
	if query != nil {
		q, err := dq.pagination.Apply(query)
		if err != nil {
			return nil, err
		}

		query = q
	}

	return query, nil
}
