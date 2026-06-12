package database

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

type TestRole struct {
	ID   uint   `gorm:"primaryKey"`
	Name string
}

type TestModel struct {
	ID     uint   `gorm:"primaryKey"`
	Name   string
	Email  string
	Age    int
	IDRole uint
	Role   TestRole `gorm:"foreignKey:IDRole"`
}

func TestApplyFilters_Functionality(t *testing.T) {
	db := testDB.Session(&gorm.Session{DryRun: true})

	t.Run("Basic Equality Filter", func(t *testing.T) {
		params := FilterParams{
			Filters: map[string]interface{}{
				"name": "John",
			},
		}
		filterable := map[string]FilterConfig{
			"name": {},
		}
		query, err := ApplyFilters(db.Model(&TestModel{}), params, filterable, nil)
		assert.NoError(t, err)
		sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]TestModel{}) })
		// Verifica se incluiu o prefixo da tabela (test_model) para evitar ambiguidade
		assert.Contains(t, strings.ToLower(sql), "test_model")
		assert.Contains(t, strings.ToLower(sql), "name")
		assert.Contains(t, sql, "'John'")
	})

	t.Run("Range Filters _start and _end", func(t *testing.T) {
		params := FilterParams{
			Filters: map[string]interface{}{
				"age_start": 20,
				"age_end":   30,
			},
		}
		filterable := map[string]FilterConfig{
			"age": {},
		}
		query, err := ApplyFilters(db.Model(&TestModel{}), params, filterable, nil)
		assert.NoError(t, err)
		sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]TestModel{}) })
		assert.Contains(t, sql, "age")
		assert.Contains(t, sql, ">= 20")
		assert.Contains(t, sql, "<= 30")
	})

	t.Run("Global Search Word", func(t *testing.T) {
		params := FilterParams{
			SearchWord:   "test",
			SearchFields: "name,email",
		}
		searchable := []SearchConfig{
			{Key: "name"},
			{Key: "email"},
		}
		query, err := ApplyFilters(db.Model(&TestModel{}), params, nil, searchable)
		assert.NoError(t, err)
		sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]TestModel{}) })
		assert.Contains(t, strings.ToLower(sql), "name")
		assert.Contains(t, strings.ToLower(sql), "email")
		assert.Contains(t, sql, "'%test%'")
	})

	t.Run("Sorting ASC and DESC", func(t *testing.T) {
		filterable := map[string]FilterConfig{"name": {}}
		paramsASC := FilterParams{Order: Order{OrderBy: "name", OrderDirection: "asc"}}
		queryASC, err := ApplyFilters(db.Model(&TestModel{}), paramsASC, filterable, nil)
		assert.NoError(t, err)
		sqlASC := queryASC.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]TestModel{}) })
		assert.Contains(t, strings.ToLower(sqlASC), "order by")
		assert.Contains(t, strings.ToLower(sqlASC), "name")
		assert.Contains(t, strings.ToLower(sqlASC), "asc")

		paramsDESC := FilterParams{Order: Order{OrderBy: "name", OrderDirection: "desc"}}
		queryDESC, err := ApplyFilters(db.Model(&TestModel{}), paramsDESC, filterable, nil)
		assert.NoError(t, err)
		sqlDESC := queryDESC.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]TestModel{}) })
		assert.Contains(t, strings.ToLower(sqlDESC), "order by")
		assert.Contains(t, strings.ToLower(sqlDESC), "name")
		assert.Contains(t, strings.ToLower(sqlDESC), "desc")
	})

	t.Run("Pagination", func(t *testing.T) {
		params := FilterParams{Pagination: Pagination{Page: 2, Limit: 10}}
		query, err := ApplyFilters(db.Model(&TestModel{}), params, nil, nil)
		assert.NoError(t, err)
		sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]TestModel{}) })
		assert.Contains(t, sql, "LIMIT 10 OFFSET 10")
	})

	t.Run("Dot-prefixed Filter with Joins", func(t *testing.T) {
		params := FilterParams{
			Filters: map[string]interface{}{
				"Role.name": "Admin",
			},
		}
		filterable := map[string]FilterConfig{
			"Role.name": {Relation: "nested"},
		}
		query, err := ApplyFilters(db.Model(&TestModel{}), params, filterable, nil)
		assert.NoError(t, err)
		sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]TestModel{}) })
		
		// Verifica se incluiu o JOIN
		assert.Contains(t, strings.ToUpper(sql), "JOIN")
		assert.Contains(t, strings.ToUpper(sql), "TEST_ROLE")
		assert.Contains(t, strings.ToLower(sql), "role")
		assert.Contains(t, strings.ToLower(sql), "name")
		assert.Contains(t, sql, "'Admin'")
	})

	t.Run("Operator contains and custom TargetKey", func(t *testing.T) {
		params := FilterParams{
			Filters: map[string]interface{}{
				"search_name": "John",
			},
		}
		filterable := map[string]FilterConfig{
			"search_name": {Operator: "contains", TargetKey: "name"},
		}
		query, err := ApplyFilters(db.Model(&TestModel{}), params, filterable, nil)
		assert.NoError(t, err)
		sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]TestModel{}) })
		assert.Regexp(t, `(?i)ILIKE`, sql)
		assert.Contains(t, sql, "'%John%'")
		assert.Contains(t, strings.ToLower(sql), "name")
	})

	t.Run("Global Search with Nested Relation", func(t *testing.T) {
		params := FilterParams{
			SearchWord:   "Admin",
			SearchFields: "Role.name",
		}
		searchable := []SearchConfig{
			{Key: "Role.name", Relation: "nested"},
		}
		query, err := ApplyFilters(db.Model(&TestModel{}), params, nil, searchable)
		assert.NoError(t, err)
		sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]TestModel{}) })
		assert.Contains(t, strings.ToUpper(sql), "JOIN")
		assert.Contains(t, strings.ToUpper(sql), "TEST_ROLE")
		assert.Regexp(t, `(?i)ILIKE`, sql)
	})

	t.Run("Ordering with created_at and Page < 1", func(t *testing.T) {
		params := FilterParams{
			Order:      Order{OrderBy: "created_at"},
			Pagination: Pagination{Page: 0, Limit: 5},
		}
		query, err := ApplyFilters(db.Model(&TestModel{}), params, nil, nil)
		assert.NoError(t, err)
		sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]TestModel{}) })
		assert.Contains(t, strings.ToLower(sql), "order by")
		assert.Contains(t, strings.ToLower(sql), "created_at")
		assert.Contains(t, strings.ToUpper(sql), "LIMIT 5")
	})

	t.Run("EQUALS operator and Multiple Nested Filters", func(t *testing.T) {
		params := FilterParams{
			Filters: map[string]interface{}{
				"Role.name": "Admin",
				"Role.id":   1,
			},
		}
		filterable := map[string]FilterConfig{
			"Role.name": {Relation: "nested", Operator: "equals"},
			"Role.id":   {Relation: "nested"},
		}
		query, err := ApplyFilters(db.Model(&TestModel{}), params, filterable, nil)
		assert.NoError(t, err)
		sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]TestModel{}) })
		
		// Deve ter apenas um JOIN
		assert.Equal(t, 1, strings.Count(strings.ToUpper(sql), "JOIN"))
		assert.Contains(t, sql, " = 'Admin'")
		assert.Contains(t, sql, " = 1")
	})

	t.Run("Ordering with updated_at and Empty Search Fields", func(t *testing.T) {
		params := FilterParams{
			Order:        Order{OrderBy: "updated_at", OrderDirection: "desc"},
			SearchWord:   "test",
			SearchFields: "name, , email", // Espaço vazio no meio
		}
		searchable := []SearchConfig{
			{Key: "name"},
			{Key: "email"},
		}
		query, err := ApplyFilters(db.Model(&TestModel{}), params, nil, searchable)
		assert.NoError(t, err)
		sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]TestModel{}) })
		assert.Contains(t, strings.ToLower(sql), "order by")
		assert.Contains(t, strings.ToLower(sql), "updated_at")
		assert.Contains(t, strings.ToLower(sql), "desc")
	})

	t.Run("Empty mainTable fallback", func(t *testing.T) {
		params := FilterParams{
			Filters: map[string]interface{}{"name": "John"},
		}
		filterable := map[string]FilterConfig{"name": {}}
		// Usamos db diretamente sem Model()
		query, err := ApplyFilters(db, params, filterable, nil)
		assert.NoError(t, err)
		sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Table("test_model").Find(&[]TestModel{}) })
		// Não deve ter o prefixo da tabela na cláusula WHERE se mainTable for ""
		assert.NotContains(t, sql, "test_model.name")
		assert.Contains(t, sql, "name = 'John'")
	})

	t.Run("Empty SearchFields and Invalid Model", func(t *testing.T) {
		params := FilterParams{
			SearchWord:   "test",
			SearchFields: ",,",
		}
		// db.Model("string") deve falhar no Parse do GORM
		query, err := ApplyFilters(db.Model("not-a-model"), params, nil, nil)
		assert.NoError(t, err)
		sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]TestModel{}) })
		// Não deve ter cláusula WHERE de busca pois orConditions ficou vazio
		assert.NotContains(t, strings.ToUpper(sql), "LIKE")
	})

	t.Run("Default Ordering without mainTable", func(t *testing.T) {
		params := FilterParams{}
		query, err := ApplyFilters(db, params, nil, nil)
		assert.NoError(t, err)
		sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Table("test_model").Find(&[]TestModel{}) })
		assert.Contains(t, strings.ToUpper(sql), "ORDER BY")
		assert.Contains(t, strings.ToLower(sql), "created_at")
		assert.Contains(t, strings.ToUpper(sql), "DESC")
	})

	t.Run("Explicit Table name in Statement", func(t *testing.T) {
		params := FilterParams{
			Filters: map[string]interface{}{"name": "John"},
		}
		filterable := map[string]FilterConfig{"name": {}}
		// Usamos Table() explicitamente
		query, err := ApplyFilters(db.Table("custom_table"), params, filterable, nil)
		assert.NoError(t, err)
		sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]TestModel{}) })
		assert.Contains(t, sql, "custom_table.name")
	})

	t.Run("Nested Relation without Dot", func(t *testing.T) {
		params := FilterParams{
			Filters: map[string]interface{}{"nested_key": "val"},
			SearchWord: "test",
			SearchFields: "nested_search",
		}
		filterable := map[string]FilterConfig{
			"nested_key": {Relation: "nested"}, // Sem ponto
		}
		searchable := []SearchConfig{
			{Key: "nested_search", Relation: "nested"}, // Sem ponto
		}
		query, err := ApplyFilters(db.Model(&TestModel{}), params, filterable, searchable)
		assert.NoError(t, err)
		sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]TestModel{}) })
		assert.NotContains(t, strings.ToUpper(sql), "JOIN")
	})

	t.Run("Custom Operator", func(t *testing.T) {
		params := FilterParams{
			Filters: map[string]interface{}{
				"age": 25,
			},
		}
		filterable := map[string]FilterConfig{
			"age": {Operator: ">"},
		}
		query, err := ApplyFilters(db.Model(&TestModel{}), params, filterable, nil)
		assert.NoError(t, err)
		sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]TestModel{}) })
		assert.Contains(t, sql, ">")
		assert.Contains(t, sql, "25")
	})

	t.Run("Multiple Search Fields with Same Relation", func(t *testing.T) {
		params := FilterParams{
			SearchWord:   "Admin",
			SearchFields: "Role.name, Role.email",
		}
		searchable := []SearchConfig{
			{Key: "Role.name", Relation: "nested"},
			{Key: "Role.email", Relation: "nested"},
		}
		query, err := ApplyFilters(db.Model(&TestModel{}), params, nil, searchable)
		assert.NoError(t, err)
		sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]TestModel{}) })
		assert.Contains(t, strings.ToUpper(sql), "JOIN")
		assert.Contains(t, strings.ToUpper(sql), "TEST_ROLE")
		assert.Regexp(t, `(?i)ILIKE`, sql)
	})

	t.Run("Filters with Ignored Keys", func(t *testing.T) {
		params := FilterParams{
			Filters: map[string]interface{}{
				"empty":                "",
				"nil_val":              nil,
				"ignoreDefaultFilters": true,
				"page":                 1,
				"limit":                10,
				"size":                 10,
				"orderBy":              "name",
				"orderDirection":       "asc",
				"sort":                 "asc",
				"searchWord":           "test",
				"searchFields":         "name",
			},
		}
		query, err := ApplyFilters(db.Model(&TestModel{}), params, nil, nil)
		assert.NoError(t, err)
		sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]TestModel{}) })
		assert.NotContains(t, sql, "empty")
		assert.NotContains(t, sql, "page")
	})

	t.Run("SearchWord without SearchFields", func(t *testing.T) {
		params := FilterParams{
			SearchWord: "test",
		}
		_, err := ApplyFilters(db.Model(&TestModel{}), params, nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "o parâmetro 'searchFields' é obrigatório")
	})

}

func TestApplyFilters_Validation(t *testing.T) {
	db := testDB.Session(&gorm.Session{DryRun: true})

	t.Run("Pagination Limit", func(t *testing.T) {
		params := FilterParams{
			Pagination: Pagination{Limit: 101},
		}
		_, err := ApplyFilters(db.Model(&TestModel{}), params, nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "limite de paginação '101' não é permitido")
	})

	t.Run("Blocked Filter", func(t *testing.T) {
		filterable := map[string]FilterConfig{"name": {}}
		params := FilterParams{
			Filters: map[string]interface{}{
				"email": "john@test.com",
			},
		}
		_, err := ApplyFilters(db.Model(&TestModel{}), params, filterable, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "filtro 'email' não está disponível")
	})

	t.Run("Blocked Search Field", func(t *testing.T) {
		searchable := []SearchConfig{{Key: "name"}}
		params := FilterParams{
			SearchWord:   "test",
			SearchFields: "email",
		}
		_, err := ApplyFilters(db.Model(&TestModel{}), params, nil, searchable)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "campo de busca 'email' não está disponível")
	})

	t.Run("Blocked Order Field", func(t *testing.T) {
		filterable := map[string]FilterConfig{"name": {}}
		params := FilterParams{
			Order: Order{OrderBy: "email"},
		}
		_, err := ApplyFilters(db.Model(&TestModel{}), params, filterable, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ordenação por 'email' não está disponível")
	})

	t.Run("Ignored Default Filters", func(t *testing.T) {
		params := FilterParams{
			Filters: map[string]interface{}{
				"ignoreDefaultFilters": true,
			},
		}
		_, err := ApplyFilters(db.Model(&TestModel{}), params, nil, nil)
		assert.NoError(t, err)
	})
}

func TestApplyFilters_Date(t *testing.T) {
	db := testDB.Session(&gorm.Session{DryRun: true})

	t.Run("Date Filter _end includes full day", func(t *testing.T) {
		params := FilterParams{
			Filters: map[string]interface{}{
				"created_at_end": "2026-05-29",
			},
		}
		filterable := map[string]FilterConfig{
			"created_at": {Type: "date"},
		}

		query, err := ApplyFilters(db.Model(&TestModel{}), params, filterable, nil)
		assert.NoError(t, err)
		sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]TestModel{}) })

		assert.Contains(t, sql, "2026-05-29 23:59:59.999")
	})

	t.Run("Date Filter _start includes start of day", func(t *testing.T) {
		params := FilterParams{
			Filters: map[string]interface{}{
				"created_at_start": "2026-05-01",
			},
		}
		filterable := map[string]FilterConfig{
			"created_at": {Type: "date"},
		}

		query, err := ApplyFilters(db.Model(&TestModel{}), params, filterable, nil)
		assert.NoError(t, err)
		sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]TestModel{}) })

		assert.Contains(t, sql, "2026-05-01 00:00:00.000")
	})

	t.Run("Date Filter exact date expands to full day range", func(t *testing.T) {
		params := FilterParams{
			Filters: map[string]interface{}{
				"created_at": "2026-05-14",
			},
		}
		filterable := map[string]FilterConfig{
			"created_at": {Type: "date"},
		}

		query, err := ApplyFilters(db.Model(&TestModel{}), params, filterable, nil)
		assert.NoError(t, err)
		sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]TestModel{}) })

		assert.Contains(t, sql, ">= '2026-05-14 00:00:00")
		assert.Contains(t, sql, "<= '2026-05-14 23:59:59.999")
	})

	t.Run("Date Filter fallback for createdAt camelCase", func(t *testing.T) {
		params := FilterParams{
			Filters: map[string]interface{}{
				"createdAt": "2026-05-14",
			},
		}
		filterable := map[string]FilterConfig{
			"created_at": {Type: "date"},
		}

		query, err := ApplyFilters(db.Model(&TestModel{}), params, filterable, nil)
		assert.NoError(t, err)
		sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]TestModel{}) })

		assert.Contains(t, sql, ">= '2026-05-14 00:00:00")
		assert.Contains(t, sql, "<= '2026-05-14 23:59:59.999")
	})

	t.Run("Date Filter fallback for updatedAt camelCase", func(t *testing.T) {
		params := FilterParams{
			Filters: map[string]interface{}{
				"updatedAt": "2026-06-01",
			},
		}
		filterable := map[string]FilterConfig{
			"updated_at": {Type: "date"},
		}

		query, err := ApplyFilters(db.Model(&TestModel{}), params, filterable, nil)
		assert.NoError(t, err)
		sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]TestModel{}) })

		assert.Contains(t, sql, ">= '2026-06-01 00:00:00")
		assert.Contains(t, sql, "<= '2026-06-01 23:59:59.999")
	})

	t.Run("Date Filter with non-string value", func(t *testing.T) {
		params := FilterParams{
			Filters: map[string]interface{}{
				"created_at": 12345,
			},
		}
		filterable := map[string]FilterConfig{
			"created_at": {Type: "date"},
		}
		query, err := ApplyFilters(db.Model(&TestModel{}), params, filterable, nil)
		assert.NoError(t, err)
		sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]TestModel{}) })
		assert.Contains(t, sql, "\"test_model\".\"created_at\" = 12345")
	})

	t.Run("Date Filter with short string value", func(t *testing.T) {
		params := FilterParams{
			Filters: map[string]interface{}{
				"created_at": "2026-05",
			},
		}
		filterable := map[string]FilterConfig{
			"created_at": {Type: "date"},
		}
		_, err := ApplyFilters(db.Model(&TestModel{}), params, filterable, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "formato de data para o filtro 'created_at' não é permitido")
	})

	t.Run("Date Filter with negative timezone offset", func(t *testing.T) {
		// Save original time.Local and restore it after the test
		origLocal := time.Local
		defer func() { time.Local = origLocal }()

		// Force a negative timezone offset (-3 hours = -10800 seconds)
		time.Local = time.FixedZone("BRT", -10800)
		
		params := FilterParams{
			Filters: map[string]interface{}{
				"created_at": "2026-05-14",
			},
		}
		filterable := map[string]FilterConfig{
			"created_at": {Type: "date"},
		}
		query, err := ApplyFilters(db.Model(&TestModel{}), params, filterable, nil)
		assert.NoError(t, err)
		sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Find(&[]TestModel{}) })
		assert.Contains(t, sql, "-03:00") // America/Sao_Paulo has a -03:00 offset
	})
}
