# QueryHelper

A Go library for building safe and flexible database queries with pagination, search, filtering, and sorting capabilities. Designed for GORM but can be adapted for other ORMs.

## Features

- **Pagination**: Automatic pagination with configurable page size and limits
- **Search**: Multi-field text search with LIKE queries
- **Filtering**: Advanced filtering with multiple operators (=, !=, >, <, >=, <=, BETWEEN, IN, NOT IN, LIKE)
- **Sorting**: Multi-field sorting with ascending/descending support
- **Security**: Whitelist-based field and operator validation
- **Field Aliasing**: Map frontend field names to database column names
- **Two-Stage Architecture**: Clean separation between API layer and business logic

## Installation

```bash
go get github.com/yourusername/queryhelper
```

## Quick Start

```go
import "github.com/yourusername/queryhelper"

// Create a query helper
qh := queryhelper.NewQueryHelper(
    queryhelper.WithPage(1),
    queryhelper.WithPageSize(20),
    queryhelper.WithSearchText("keyword"),
    queryhelper.WithSearchFields([]string{"name", "description"}),
    queryhelper.WithFilters([]queryhelper.FilterCondition{
        {Field: "price", Operator: ">=", Value: 100},
        {Field: "status", Operator: "=", Value: "active"},
    }),
)

// Define security settings
settings := &queryhelper.QuerySettings{
    AllowedSearch: []string{"name", "description"},
    AllowedOrderBy: []string{"created_at", "price"},
    AllowedFilters: map[string][]string{
        "price":  {">=", "<=", ">", "<"},
        "status": {"=", "IN"},
    },
}

// Apply to GORM query
query, err := qh.Apply(settings, db.Model(&Product{}))

// Execute query
var products []Product
query.Find(&products)

// Get pagination info
info := qh.Info()
fmt.Printf("Page: %d, Total: %d, TotalPages: %d\n",
    info.Pagination.Page,
    info.Pagination.Total,
    info.Pagination.TotalPages)
```

## Two-Stage Architecture

QueryHelper is designed to work in a two-stage architecture pattern:

### Stage 1: API Layer - Receive and Build Query

The API layer receives requests and constructs the QueryHelper object:

```go
// handlers/product_handler.go
package handlers

import (
    "github.com/gin-gonic/gin"
    "github.com/yourusername/queryhelper"
)

type ListProductsRequest struct {
    Page         int                           `json:"page"`
    PageSize     int                           `json:"page_size"`
    SearchText   string                        `json:"search_text"`
    SearchFields []string                      `json:"search_fields"`
    OrderBy      []string                      `json:"order_by"`
    SortFactor   int                           `json:"sort_factor"` // 1: asc, -1: desc
    Filters      []queryhelper.FilterCondition `json:"filters"`
}

func (h *ProductHandler) ListProducts(c *gin.Context) {
    var req ListProductsRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    // Build query helper
    qh := queryhelper.NewQueryHelper(
        queryhelper.WithPage(req.Page),
        queryhelper.WithPageSize(req.PageSize),
        queryhelper.WithSearchText(req.SearchText),
        queryhelper.WithSearchFields(req.SearchFields),
        queryhelper.WithOrderBy(req.OrderBy),
        queryhelper.WithSortFactor(req.SortFactor),
        queryhelper.WithFilters(req.Filters),
    )

    // Pass to service layer
    products, info, err := h.productService.ListProducts(c.Request.Context(), qh)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, gin.H{
        "data":       products,
        "pagination": info.Pagination,
        "conditions": info.Conditions,
    })
}
```

### Stage 2: Service Layer - Apply and Execute Query

The service layer applies business rules and executes the query:

```go
// services/product_service.go
package services

import (
    "context"
    "github.com/yourusername/queryhelper"
    "gorm.io/gorm"
)

type ProductService struct {
    db *gorm.DB
}

func (s *ProductService) ListProducts(ctx context.Context, qh *queryhelper.QueryHelper) ([]Product, *queryhelper.QueryHelperInfo, error) {
    var products []Product

    // Define business rules and security settings
    settings := &queryhelper.QuerySettings{
        AllowedSearch: []string{"name", "description", "sku"},
        AllowedOrderBy: []string{"created_at", "updated_at", "price", "name"},
        AllowedFilters: map[string][]string{
            "price":       {">=", "<=", ">", "<", "="},
            "stock":       {">", ">=", "="},
            "status":      {"=", "IN"},
            "category_id": {"=", "IN"},
            "brand":       {"=", "LIKE"},
        },
        ColumnAlias: map[string]string{
            // Map frontend field names to database columns if needed
            "category": "category_id",
        },
        DefaultSortFactor: -1, // Default to descending
    }

    // Apply query helper with settings
    query, err := qh.Apply(settings, s.db.Model(&Product{}))
    if err != nil {
        return nil, nil, err
    }

    // Execute query
    if err := query.Find(&products).Error; err != nil {
        return nil, nil, err
    }

    return products, qh.Info(), nil
}
```

## API Request Example

```json
POST /api/products
{
  "page": 1,
  "page_size": 20,
  "search_text": "smartphone",
  "search_fields": ["name", "description"],
  "order_by": ["price", "created_at"],
  "sort_factor": -1,
  "filters": [
    {
      "field": "price",
      "operator": ">=",
      "value": 100
    },
    {
      "field": "price",
      "operator": "<=",
      "value": 1000
    },
    {
      "field": "status",
      "operator": "=",
      "value": "active"
    },
    {
      "field": "category_id",
      "operator": "IN",
      "value": [1, 2, 3]
    }
  ]
}
```

## API Documentation

### Core Types

#### QueryHelper

The main query builder object.

**Constructor:**
```go
func NewQueryHelper(opts ...Option) *QueryHelper
```

**Methods:**
```go
// Apply settings and generate GORM query
func (qh *QueryHelper) Apply(settings *QuerySettings, query *gorm.DB) (*gorm.DB, error)

// Get current query information
func (qh *QueryHelper) Info() *QueryHelperInfo

// Get pagination request
func (qh *QueryHelper) GetPaginationRequest() *PaginationRequest

// Get query conditions
func (qh *QueryHelper) GetQueryConditions() *QueryConditions
```

#### Options

```go
// Pagination options
WithPage(page int) Option
WithPageSize(pageSize int) Option

// Search options
WithSearchText(text string) Option
WithSearchFields(fields []string) Option

// Sorting options
WithOrderBy(fields []string) Option
WithSortFactor(factor int) Option  // 1: ascending, -1: descending

// Filtering options
WithFilters(filters []FilterCondition) Option
```

#### QuerySettings

Security and configuration settings applied at the service layer.

```go
type QuerySettings struct {
    // Map frontend field names to database columns
    ColumnAlias map[string]string

    // Allowed fields for ORDER BY
    AllowedOrderBy []string

    // Allowed fields for search
    AllowedSearch []string

    // Allowed fields and their operators for filtering
    // Example: {"price": {">=", "<=", "="}}
    AllowedFilters map[string][]string

    // Default sort direction: 1 (asc) or -1 (desc)
    DefaultSortFactor int
}
```

#### FilterCondition

Represents a single filter condition.

```go
type FilterCondition struct {
    Field    string      `json:"field"`
    Operator string      `json:"operator"`
    Value    interface{} `json:"value"`
}
```

### Supported Operators

| Operator | Description | Value Type | Example |
|----------|-------------|------------|---------|
| `=` | Equal | Any | `{"field": "status", "operator": "=", "value": "active"}` |
| `!=` | Not equal | Any | `{"field": "status", "operator": "!=", "value": "deleted"}` |
| `>` | Greater than | Number | `{"field": "price", "operator": ">", "value": 100}` |
| `<` | Less than | Number | `{"field": "price", "operator": "<", "value": 1000}` |
| `>=` | Greater or equal | Number | `{"field": "stock", "operator": ">=", "value": 10}` |
| `<=` | Less or equal | Number | `{"field": "stock", "operator": "<=", "value": 100}` |
| `BETWEEN` | Between range | Array [min, max] | `{"field": "price", "operator": "BETWEEN", "value": [100, 500]}` |
| `IN` | In list | Array | `{"field": "category_id", "operator": "IN", "value": [1, 2, 3]}` |
| `NOT IN` | Not in list | Array | `{"field": "status", "operator": "NOT IN", "value": ["deleted", "archived"]}` |
| `LIKE` | Pattern match | String | `{"field": "name", "operator": "LIKE", "value": "%phone%"}` |

## Security Features

### Whitelist Validation

QueryHelper automatically filters out unauthorized fields and operators:

```go
settings := &queryhelper.QuerySettings{
    AllowedFilters: map[string][]string{
        "price": {">=", "<="},  // Only allows >= and <= for price
        "status": {"="},        // Only allows = for status
    },
}

// This filter will be IGNORED (field not in whitelist)
filters := []FilterCondition{
    {Field: "internal_cost", Operator: ">=", Value: 100},
}

// This filter will be IGNORED (operator not allowed)
filters := []FilterCondition{
    {Field: "price", Operator: ">", Value: 100},  // Only >= and <= allowed
}
```

### Field Aliasing

Protect internal column names by mapping frontend fields:

```go
settings := &queryhelper.QuerySettings{
    AllowedFilters: map[string][]string{
        "category": {"=", "IN"},
    },
    ColumnAlias: map[string]string{
        "category": "category_id",  // Frontend uses "category", DB uses "category_id"
    },
}
```

## Configuration

### Pagination Defaults

```go
const (
    DefaultPage        = 1
    DefaultPageSize    = 10
    DefaultMaxPageSize = 100
)
```

You can customize these by modifying the constants or using options:

```go
qh := queryhelper.NewQueryHelper(
    queryhelper.WithPage(1),
    queryhelper.WithPageSize(50),  // Custom page size
)
```

### Default Query Settings

```go
var DefaultQuerySettings = &QuerySettings{
    AllowedOrderBy:    []string{"created_at"},
    AllowedSearch:     []string{},
    AllowedFilters:    map[string][]string{},
    DefaultSortFactor: 1, // ascending
}
```

## Advanced Usage

### Combining with Additional Conditions

```go
// Start with base query
query := db.Model(&Product{}).Where("deleted_at IS NULL")

// Add user-specific filter
if userRole != "admin" {
    query = query.Where("visibility = ?", "public")
}

// Apply QueryHelper
query, err := qh.Apply(settings, query)

// Execute
query.Find(&products)
```

### Custom Filters in Service Layer

```go
func (s *ProductService) ListProducts(ctx context.Context, qh *queryhelper.QueryHelper) ([]Product, *queryhelper.QueryHelperInfo, error) {
    // Get user from context
    user := getUserFromContext(ctx)

    // Start with base query
    query := s.db.Model(&Product{})

    // Apply role-based filtering
    if user.Role != "admin" {
        query = query.Where("user_id = ?", user.ID)
    }

    // Apply QueryHelper
    query, err := qh.Apply(settings, query)
    if err != nil {
        return nil, nil, err
    }

    // Execute
    var products []Product
    query.Find(&products)

    return products, qh.Info(), nil
}
```

### Multiple Search Fields with Different Logic

```go
// Text search across multiple fields (OR logic)
qh := queryhelper.NewQueryHelper(
    queryhelper.WithSearchText("keyword"),
    queryhelper.WithSearchFields([]string{"name", "description", "sku"}),
)
// Generates: WHERE (name LIKE '%keyword%' OR description LIKE '%keyword%' OR sku LIKE '%keyword%')

// Exact filters (AND logic)
qh := queryhelper.NewQueryHelper(
    queryhelper.WithFilters([]queryhelper.FilterCondition{
        {Field: "category_id", Operator: "=", Value: 1},
        {Field: "status", Operator: "=", Value: "active"},
    }),
)
// Generates: WHERE category_id = 1 AND status = 'active'
```

## Response Structure

### QueryHelperInfo

```go
type QueryHelperInfo struct {
    Pagination *PaginationInfo
    Conditions *QueryConditions
}
```

### PaginationInfo

```go
type PaginationInfo struct {
    Page       int   `json:"page"`        // Current page number
    PageSize   int   `json:"page_size"`   // Items per page
    Total      int64 `json:"total"`       // Total number of items
    TotalPages int   `json:"total_pages"` // Total number of pages
}
```

### Example Response

```json
{
  "data": [
    {
      "id": 1,
      "name": "Product A",
      "price": 299.99
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 156,
    "total_pages": 8
  },
  "conditions": {
    "search_text": "smartphone",
    "search_fields": ["name", "description"],
    "order_by": ["price"],
    "sort_factor": -1,
    "filters": [
      {
        "field": "price",
        "operator": ">=",
        "value": 100
      }
    ]
  }
}
```

## Best Practices

1. **Always define AllowedFilters**: Never leave it empty in production
2. **Use field aliasing**: Hide internal column names from API
3. **Set appropriate page limits**: Prevent performance issues with large queries
4. **Validate in both layers**: API layer for format, Service layer for business rules
5. **Use context**: Pass context for cancellation and timeout support
6. **Log filtered conditions**: Monitor what filters are being rejected

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

Apache License 2.0 - feel free to use this in your projects.

## Author

Created with ❤️ for the Go community
