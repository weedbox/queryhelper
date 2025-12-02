package queryhelper

import (
	"errors"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FilterCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // =, !=, >, <, >=, <=, BETWEEN, IN, NOT IN, LIKE
	Value    interface{} `json:"value"`
}

type QueryConditions struct {
	SearchText   string            `json:"search_text"`
	SearchFields []string          `json:"search_fields"`
	OrderBy      []string          `json:"order_by"`
	SortFactor   int               `json:"sort_factor"` // 1, -1
	Filters      []FilterCondition `json:"filters"`
}

type ConditionsHandle struct {
	Settings   *QuerySettings   `json:"settings"`
	Conditions *QueryConditions `json:"conditions"`
}

type QuerySettings struct {
	ColumnAlias       map[string]string            `json:"column_alias"`
	AllowedOrderBy    []string                     `json:"allowed_order_by"`
	AllowedSearch     []string                     `json:"allowed_search"`
	AllowedFilters    map[string][]string          `json:"allowed_filters"` // field -> allowed operators
	DefaultSortFactor int                          `json:"default_sort_factor"`
}

var DefaultQuerySettings = &QuerySettings{
	AllowedOrderBy:    []string{"created_at"},
	AllowedSearch:     []string{},
	AllowedFilters:    map[string][]string{},
	DefaultSortFactor: 1, // ascending
}

func getRealColumns(alias map[string]string, columns []string) []string {

	fields := make([]string, len(columns))

	for i, col := range columns {
		if v, ok := alias[col]; ok {
			fields[i] = v
			continue
		}

		fields[i] = col
	}

	return fields
}

func NewConditionsHandle(settings *QuerySettings) *ConditionsHandle {

	if settings == nil {
		settings = DefaultQuerySettings
	}

	return &ConditionsHandle{
		Settings:   settings,
		Conditions: nil,
	}
}

func (ch *ConditionsHandle) UpdateConditions(conditions *QueryConditions) error {

	settings := ch.Settings

	// check search fields
	var allowedSearch []string
	// If no search fields provided, SearchFields = [""]
	if len(conditions.SearchFields) == 0 || (len(conditions.SearchFields) == 1 && conditions.SearchFields[0] == "") {
		allowedSearch = settings.AllowedSearch
	} else {

		// filter search fields
		allowedSearch = make([]string, 0)
		for _, field := range conditions.SearchFields {
			for _, allowed := range settings.AllowedSearch {
				if field == allowed {
					allowedSearch = append(allowedSearch, field)
					break
				}
			}
		}
	}

	// map search fields
	conditions.SearchFields = getRealColumns(settings.ColumnAlias, allowedSearch)

	// check order by
	var orderBy []string
	if conditions.OrderBy == nil || len(conditions.OrderBy) == 0 {
		orderBy = settings.AllowedOrderBy
	} else {

		// filter order by fields
		orderBy = make([]string, 0)
		for _, ob := range conditions.OrderBy {
			for _, allowed := range settings.AllowedOrderBy {
				if ob == allowed {
					orderBy = append(orderBy, ob)
					break
				}
			}
		}
	}

	// map order by fields
	conditions.OrderBy = getRealColumns(settings.ColumnAlias, orderBy)

	// check sort factor
	if conditions.SortFactor == 0 {
		conditions.SortFactor = settings.DefaultSortFactor
	} else if conditions.SortFactor > 1 {
		conditions.SortFactor = 1
	} else if conditions.SortFactor < -1 {
		conditions.SortFactor = -1
	}

	// check and filter allowed filters
	if len(conditions.Filters) > 0 {
		validFilters := make([]FilterCondition, 0)
		for _, filter := range conditions.Filters {
			// Check if field is allowed
			allowedOps, fieldAllowed := settings.AllowedFilters[filter.Field]
			if !fieldAllowed {
				continue
			}

			// Check if operator is allowed for this field
			operatorAllowed := false
			for _, op := range allowedOps {
				if filter.Operator == op {
					operatorAllowed = true
					break
				}
			}
			if !operatorAllowed {
				continue
			}

			// Map field alias to real column name
			realField := filter.Field
			if alias, ok := settings.ColumnAlias[filter.Field]; ok {
				realField = alias
			}
			filter.Field = realField

			validFilters = append(validFilters, filter)
		}
		conditions.Filters = validFilters
	}

	ch.Conditions = conditions

	return nil
}

func (ch *ConditionsHandle) CurrentInfo() *QueryConditions {
	return ch.Conditions
}

func (ch *ConditionsHandle) Apply(db *gorm.DB) (*gorm.DB, error) {

	if db == nil {
		return nil, nil
	}

	if ch.Conditions == nil {
		return db, errors.New("conditions not set")
	}

	query := db

	// Apply filters
	for _, filter := range ch.Conditions.Filters {
		switch filter.Operator {
		case "=":
			query = query.Where(filter.Field+" = ?", filter.Value)
		case "!=":
			query = query.Where(filter.Field+" != ?", filter.Value)
		case ">":
			query = query.Where(filter.Field+" > ?", filter.Value)
		case "<":
			query = query.Where(filter.Field+" < ?", filter.Value)
		case ">=":
			query = query.Where(filter.Field+" >= ?", filter.Value)
		case "<=":
			query = query.Where(filter.Field+" <= ?", filter.Value)
		case "BETWEEN":
			// Value should be an array with 2 elements
			if vals, ok := filter.Value.([]interface{}); ok && len(vals) == 2 {
				query = query.Where(filter.Field+" BETWEEN ? AND ?", vals[0], vals[1])
			}
		case "IN":
			query = query.Where(filter.Field+" IN ?", filter.Value)
		case "NOT IN":
			query = query.Where(filter.Field+" NOT IN ?", filter.Value)
		case "LIKE":
			query = query.Where(filter.Field+" LIKE ?", filter.Value)
		}
	}

	// Apply search conditions
	keywords := strings.TrimSpace(ch.Conditions.SearchText)
	if keywords != "" && len(ch.Conditions.SearchFields) > 0 {
		like := "%" + keywords + "%"

		// Build the OR conditions
		var orQuery string
		orArgs := []interface{}{}
		for i, field := range ch.Conditions.SearchFields {
			if i > 0 {
				orQuery += " OR "
			}
			orQuery += field + " LIKE ?"
			orArgs = append(orArgs, like)
		}

		// Apply to the main query with AND
		query = query.Where(orQuery, orArgs...)
	}

	// Apply order by
	orderCols := make([]clause.OrderByColumn, 0)
	for _, v := range ch.Conditions.OrderBy {
		o := clause.OrderByColumn{
			Column: clause.Column{Name: v},
			Desc:   ch.Conditions.SortFactor < 0,
		}
		orderCols = append(orderCols, o)
	}

	orderClause := clause.OrderBy{
		Columns:    orderCols,
		Expression: nil,
	}

	if len(orderCols) > 0 {
		query = query.Order(orderClause)
	}

	return query, nil
}
