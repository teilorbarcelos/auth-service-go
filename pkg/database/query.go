package database

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

const filterConditionFormat = "%s %s ?"

type Pagination struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

type Order struct {
	OrderBy        string `json:"orderBy"`
	OrderDirection string `json:"orderDirection"`
}

type FilterParams struct {
	Pagination
	Order
	SearchWord   string                 `json:"searchWord"`
	SearchFields string                 `json:"searchFields"`
	Filters      map[string]interface{} `json:"filters"`
}

type FilterConfig struct {
	Type      string
	Operator  string
	Relation  string
	TargetKey string
}

type SearchConfig struct {
	Key      string
	Relation string
}

func getMainTable(query *gorm.DB) string {
	if query.Statement.Table != "" {
		return query.Statement.Table
	} else if query.Statement.Model != nil {
		if err := query.Statement.Parse(query.Statement.Model); err == nil {
			return query.Statement.Schema.Table
		}
	}
	return ""
}

func quoteField(query *gorm.DB, fieldTarget string) string {
	if query.Statement.Schema != nil {
		return query.Statement.Quote(fieldTarget)
	}
	return fieldTarget
}

func resolveDateFilter(query *gorm.DB, quotedKey string, dateStr string, operator string) (*gorm.DB, error) {
	_, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, fmt.Errorf("formato de data inválido")
	}
	t := time.Now()
	_, offset := t.Zone()
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	tzOffset := fmt.Sprintf("%s%02d:%02d", sign, offset/3600, (offset%3600)/60)

	var value interface{}
	if operator == "" {
		start := dateStr + " 00:00:00" + tzOffset
		end := dateStr + " 23:59:59.999" + tzOffset
		return query.Where(fmt.Sprintf("%s >= ? AND %s <= ?", quotedKey, quotedKey), start, end), nil
	} else if operator == ">=" {
		value = dateStr + " 00:00:00.000" + tzOffset
	} else if operator == "<=" {
		value = dateStr + " 23:59:59.999" + tzOffset
	}
	return query.Where(fmt.Sprintf(filterConditionFormat, quotedKey, operator), value), nil
}

func getFilterConfig(fieldKey string, filterable map[string]FilterConfig) (FilterConfig, error) {
	config, ok := filterable[fieldKey]
	if !ok {
		if fieldKey == "createdAt" {
			config, ok = filterable["created_at"]
		} else if fieldKey == "updatedAt" {
			config, ok = filterable["updated_at"]
		}
		if !ok {
			return config, fmt.Errorf("filtro '%s' não está disponível", fieldKey)
		}
		if config.TargetKey == "" {
			if fieldKey == "createdAt" {
				config.TargetKey = "created_at"
			}
			if fieldKey == "updatedAt" {
				config.TargetKey = "updated_at"
			}
		}
	}
	if config.TargetKey == "" {
		config.TargetKey = fieldKey
	}
	return config, nil
}

func extractFieldKeyAndOperator(key string) (string, string) {
	if strings.HasSuffix(key, "_start") {
		return strings.TrimSuffix(key, "_start"), ">="
	}
	if strings.HasSuffix(key, "_end") {
		return strings.TrimSuffix(key, "_end"), "<="
	}
	return key, ""
}

func resolveFilterOperator(config FilterConfig, operator string, value interface{}) (string, interface{}) {
	if operator != "" {
		return operator, value
	}
	if config.Operator == "contains" {
		return "ILIKE", "%" + fmt.Sprint(value) + "%"
	}
	if config.Operator != "" {
		op := strings.ToUpper(config.Operator)
		if op == "EQUALS" {
			op = "="
		}
		return op, value
	}
	return "=", value
}

func shouldSkipFilter(key string, value interface{}) bool {
	if value == nil || value == "" {
		return true
	}
	switch key {
	case "ignoreDefaultFilters", "page", "limit", "size", "orderBy", "orderDirection", "sort", "searchWord", "searchFields":
		return true
	}
	return false
}

func joinRelationIfNeeded(query *gorm.DB, configRelation, fieldTarget string, joinedRelations map[string]bool) *gorm.DB {
	if configRelation == "nested" && strings.Contains(fieldTarget, ".") {
		relation := strings.Split(fieldTarget, ".")[0]
		if !joinedRelations[relation] {
			joinedRelations[relation] = true
			return query.Joins(relation)
		}
	}
	return query
}

func getTargetKeyWithMainTable(targetKey string, mainTable string) string {
	if !strings.Contains(targetKey, ".") && mainTable != "" {
		return fmt.Sprintf("%s.%s", mainTable, targetKey)
	}
	return targetKey
}

func applyFiltersLogic(query *gorm.DB, params FilterParams, filterable map[string]FilterConfig, mainTable string, joinedRelations map[string]bool) (*gorm.DB, error) {
	for key, value := range params.Filters {
		if shouldSkipFilter(key, value) {
			continue
		}

		fieldKey, operator := extractFieldKeyAndOperator(key)
		config, err := getFilterConfig(fieldKey, filterable)
		if err != nil {
			return nil, err
		}

		query = joinRelationIfNeeded(query, config.Relation, fieldKey, joinedRelations)
		targetKey := getTargetKeyWithMainTable(config.TargetKey, mainTable)
		quotedKey := quoteField(query, targetKey)

		if config.Type == "date" {
			if dateStr, isStr := value.(string); isStr {
				newQuery, err := resolveDateFilter(query, quotedKey, dateStr, operator)
				if err != nil {
					return nil, fmt.Errorf("formato de data para o filtro '%s' não é permitido", fieldKey)
				}
				query = newQuery
				continue
			}
		}

		op, val := resolveFilterOperator(config, operator, value)
		query = query.Where(fmt.Sprintf(filterConditionFormat, quotedKey, op), val)
	}
	return query, nil
}

func getSearchConfig(requestedField string, searchable []SearchConfig) (*SearchConfig, error) {
	for _, s := range searchable {
		if s.Key == requestedField {
			return &s, nil
		}
	}
	return nil, fmt.Errorf("campo de busca '%s' não está disponível", requestedField)
}

func applySearchLogic(query *gorm.DB, params FilterParams, searchable []SearchConfig, mainTable string, joinedRelations map[string]bool) (*gorm.DB, error) {
	if params.SearchWord == "" {
		return query, nil
	}
	if params.SearchFields == "" {
		return nil, fmt.Errorf("o parâmetro 'searchFields' é obrigatório")
	}
	
	var orConditions []string
	var orValues []interface{}
	
	for _, requestedField := range strings.Split(params.SearchFields, ",") {
		requestedField = strings.TrimSpace(requestedField)
		if requestedField == "" {
			continue
		}

		foundConfig, err := getSearchConfig(requestedField, searchable)
		if err != nil {
			return nil, err
		}

		fieldTarget := foundConfig.Key
		query = joinRelationIfNeeded(query, foundConfig.Relation, fieldTarget, joinedRelations)
		fieldTarget = getTargetKeyWithMainTable(fieldTarget, mainTable)
		quotedField := quoteField(query, fieldTarget)

		orConditions = append(orConditions, fmt.Sprintf(filterConditionFormat, quotedField, "ILIKE"))
		orValues = append(orValues, "%"+params.SearchWord+"%")
	}

	if len(orConditions) > 0 {
		query = query.Where(strings.Join(orConditions, " OR "), orValues...)
	}
	return query, nil
}

func applyOrderAndPaginationLogic(query *gorm.DB, params FilterParams, filterable map[string]FilterConfig, mainTable string) (*gorm.DB, error) {
	if params.OrderBy != "" {
		orderBy := params.OrderBy
		if _, ok := filterable[orderBy]; ok || orderBy == "created_at" || orderBy == "updated_at" {
			orderBy = getTargetKeyWithMainTable(orderBy, mainTable)
			quotedOrder := quoteField(query, orderBy)
			
			direction := "ASC"
			if strings.ToUpper(params.OrderDirection) == "DESC" {
				direction = "DESC"
			}
			query = query.Order(fmt.Sprintf("%s %s", quotedOrder, direction))
		} else {
			return nil, fmt.Errorf("ordenação por '%s' não está disponível", orderBy)
		}
	} else {
		defaultOrder := getTargetKeyWithMainTable("created_at", mainTable)
		query = query.Order(quoteField(query, defaultOrder) + " DESC")
	}

	if params.Limit > 0 {
		page := params.Page
		if page < 1 {
			page = 1
		}
		query = query.Offset((page - 1) * params.Limit).Limit(params.Limit)
	}

	return query, nil
}

func ApplyFilters(db *gorm.DB, params FilterParams, filterable map[string]FilterConfig, searchable []SearchConfig) (*gorm.DB, error) {
	if params.Limit > 100 {
		return nil, fmt.Errorf("limite de paginação '%d' não é permitido", params.Limit)
	}
	query := db
	joinedRelations := make(map[string]bool)
	mainTable := getMainTable(query)

	query, err := applyFiltersLogic(query, params, filterable, mainTable, joinedRelations)
	if err != nil {
		return nil, err
	}

	query, err = applySearchLogic(query, params, searchable, mainTable, joinedRelations)
	if err != nil {
		return nil, err
	}

	return applyOrderAndPaginationLogic(query, params, filterable, mainTable)
}
