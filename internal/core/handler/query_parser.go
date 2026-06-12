package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"backend-go/pkg/config"
	"backend-go/pkg/database"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func isReservedQueryParam(key string) bool {
	return key == "page" || key == "limit" || key == "size" || key == "orderBy" || key == "orderDirection" || key == "sort" || key == "searchWord" || key == "searchFields"
}

func normalizeFilterKey(key string) string {
	normalizedKey := key
	for _, prefix := range []string{"createdAt", "updatedAt"} {
		if normalizedKey == prefix || normalizedKey == prefix+"_start" || normalizedKey == prefix+"_end" {
			snake := prefix[:7] + "_" + strings.ToLower(prefix[7:])
			normalizedKey = strings.Replace(normalizedKey, prefix, snake, 1)
			break
		}
	}
	return normalizedKey
}

func ParseFilterParams(c *gin.Context) database.FilterParams {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	sizeStr := c.Query("size")
	if sizeStr == "" {
		sizeStr = c.DefaultQuery("limit", "25")
	}
	limit, _ := strconv.Atoi(sizeStr)

	params := database.FilterParams{
		Pagination: database.Pagination{
			Page:  page,
			Limit: limit,
		},
		Order: database.Order{
			OrderBy:        c.Query("orderBy"),
			OrderDirection: c.Query("orderDirection"),
		},
		SearchWord:   c.Query("searchWord"),
		SearchFields: c.Query("searchFields"),
		Filters:      make(map[string]interface{}),
	}

	for key, values := range c.Request.URL.Query() {
		if isReservedQueryParam(key) {
			continue
		}

		if len(values) > 0 {
			val := values[0]
			normalizedKey := normalizeFilterKey(key)

			switch val {
			case "true":
				params.Filters[normalizedKey] = true
			case "false":
				params.Filters[normalizedKey] = false
			default:
				params.Filters[normalizedKey] = val
			}
		}
	}

	return params
}

func HandleError(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "recurso não encontrado"})
		return
	}

	errMsg := err.Error()
	if strings.Contains(errMsg, "não está disponível") || 
		strings.Contains(errMsg, "obrigatório") || 
		strings.Contains(errMsg, "não é permitida") ||
		strings.Contains(errMsg, "não é permitido") {
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		return
	}

	if config.AppConfig.Environment == "production" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erro interno do servidor"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg})
}
