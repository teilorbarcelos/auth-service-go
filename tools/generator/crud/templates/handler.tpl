package {{.LowerName}}

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"backend-go/internal/core/handler"
	"backend-go/internal/core/models"
	"backend-go/pkg/database"
	"context"
)

type {{.Name}}ServiceI interface {
	Create(ctx context.Context, dto Create{{.Name}}DTO) (*models.{{.Name}}, error)
	Update(ctx context.Context, id string, updates map[string]interface{}) (*models.{{.Name}}, error)
	List(ctx context.Context, params database.FilterParams) ([]models.{{.Name}}, int64, error)
	GetByID(ctx context.Context, id string) (*models.{{.Name}}, error)
	Delete(ctx context.Context, id string) error
	SetStatus(ctx context.Context, id string, active bool) error
}

type {{.Name}}Handler struct {
	Service {{.Name}}ServiceI
}

func New{{.Name}}Handler(service {{.Name}}ServiceI) *{{.Name}}Handler {
	return &{{.Name}}Handler{Service: service}
}

// Create cria um novo registro
// @Summary Criar {{.Name}}
// @Description Cria um novo {{.LowerName}} com os dados fornecidos.
// @Tags {{.Name}}
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body Create{{.Name}}DTO true "Dados do {{.LowerName}}"
// @Success 201 {object} models.{{.Name}}
// @Failure 400 {object} map[string]string "Dados inválidos"
// @Failure 500 {object} map[string]string "Erro interno"
// @Router /{{.LowerName}} [post]
func (h *{{.Name}}Handler) Create(c *gin.Context) {
	var dto Create{{.Name}}DTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.Service.Create(c.Request.Context(), dto)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, res)
}

// Update atualiza um registro existente
// @Summary Atualizar {{.Name}}
// @Description Atualiza parcialmente um {{.LowerName}} baseado no ID.
// @Tags {{.Name}}
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "ID do {{.Name}}"
// @Param request body map[string]interface{} true "Campos a serem atualizados"
// @Success 200 {object} models.{{.Name}}
// @Failure 400 {object} map[string]string "Dados inválidos"
// @Failure 500 {object} map[string]string "Erro interno"
// @Router /{{.LowerName}}/{id} [put]
func (h *{{.Name}}Handler) Update(c *gin.Context) {
	id := c.Param("id")
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.Service.Update(c.Request.Context(), id, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

// GetByID retorna um registro pelo ID
// @Summary Obter {{.Name}}
// @Description Retorna os detalhes de um {{.LowerName}} específico.
// @Tags {{.Name}}
// @Produce json
// @Security Bearer
// @Param id path string true "ID do {{.Name}}"
// @Success 200 {object} models.{{.Name}}
// @Failure 404 {object} map[string]string "Não encontrado"
// @Router /{{.LowerName}}/{id} [get]
func (h *{{.Name}}Handler) GetByID(c *gin.Context) {
	id := c.Param("id")
	res, err := h.Service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "{{.LowerName}} não encontrado"})
		return
	}

	c.JSON(http.StatusOK, res)
}

// List retorna uma lista paginada
// @Summary Listar {{.Name}}s
// @Description Retorna {{.LowerName}}s com paginação e filtros.
// @Tags {{.Name}}
// @Produce json
// @Security Bearer
// @Param page query int false "Número da página (padrão: 1)"
// @Param size query int false "Itens por página (padrão: 25)"
// @Param searchWord query string false "Termo de busca"
// @Success 200 {object} map[string]interface{}
// @Router /{{.LowerName}} [get]
func (h *{{.Name}}Handler) List(c *gin.Context) {
	params := handler.ParseFilterParams(c)

	items, total, err := h.Service.List(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"total": total,
		"page":  params.Page,
		"limit": params.Limit,
	})
}

// ListAll retorna todos os registros sem paginação
// @Summary Listar Todos os {{.Name}}s
// @Description Retorna todos os {{.LowerName}}s.
// @Tags {{.Name}}
// @Produce json
// @Security Bearer
// @Success 200 {object} map[string]interface{}
// @Router /{{.LowerName}}/all [get]
func (h *{{.Name}}Handler) ListAll(c *gin.Context) {
	params := handler.ParseFilterParams(c)
	params.Limit = 0

	items, total, err := h.Service.List(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"total": total,
	})
}

// Delete remove um registro
// @Summary Excluir {{.Name}}
// @Description Remove um {{.LowerName}} baseado no ID.
// @Tags {{.Name}}
// @Produce json
// @Security Bearer
// @Param id path string true "ID do {{.Name}}"
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /{{.LowerName}}/{id} [delete]
func (h *{{.Name}}Handler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.Service.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "{{.LowerName}} excluído com sucesso"})
}

// SetStatus ativa ou desativa um registro
// @Summary Alterar Status do {{.Name}}
// @Description Define se um {{.LowerName}} está ativo ou inativo.
// @Tags {{.Name}}
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "ID do {{.Name}}"
// @Param request body map[string]bool true "Novo status"
// @Success 200 {object} map[string]string
// @Router /{{.LowerName}}/{id}/status [patch]
func (h *{{.Name}}Handler) SetStatus(c *gin.Context) {
	id := c.Param("id")
	var body struct {
		Active bool `json:"active"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.Service.SetStatus(c.Request.Context(), id, body.Active); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "status atualizado com sucesso"})
}
