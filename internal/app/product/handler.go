package product

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"backend-go/internal/core/handler"
	"backend-go/internal/core/models"
	"backend-go/pkg/database"
	"context"
)

type ProductServiceI interface {
	Create(ctx context.Context, dto CreateProductDTO) (*models.Product, error)
	Update(ctx context.Context, id string, updates map[string]interface{}) (*models.Product, error)
	List(ctx context.Context, params database.FilterParams) ([]models.Product, int64, error)
	GetByID(ctx context.Context, id string) (*models.Product, error)
	Delete(ctx context.Context, id string) error
	SetStatus(ctx context.Context, id string, active bool) error
}

type ProductHandler struct {
	Service ProductServiceI
}

type ProductListResponse struct {
	Items []models.Product `json:"items"`
	Total int64            `json:"total"`
	Page  int              `json:"page"`
	Limit int              `json:"limit"`
}

type ProductListAllResponse struct {
	Items []models.Product `json:"items"`
	Total int64            `json:"total"`
}

func NewProductHandler(service ProductServiceI) *ProductHandler {
	return &ProductHandler{Service: service}
}

type UpdateStatusRequest struct {
	Active bool `json:"active"`
}

// Create cria um novo produto
// @Summary Criar Produto
// @Description Cria um novo produto com os dados fornecidos.
// @Tags Product
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body CreateProductDTO true "Dados do produto"
// @Success 201 {object} models.Product
// @Failure 400 {object} map[string]string "Dados inválidos"
// @Failure 500 {object} map[string]string "Erro interno"
// @Router /product [post]
func (h *ProductHandler) Create(c *gin.Context) {
	var dto CreateProductDTO
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

// Update atualiza um produto existente
// @Summary Atualizar Produto
// @Description Atualiza parcialmente um produto baseado no ID.
// @Tags Product
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "ID do Produto"
// @Param request body map[string]interface{} true "Campos a serem atualizados"
// @Success 200 {object} models.Product
// @Failure 400 {object} map[string]string "Dados inválidos"
// @Failure 500 {object} map[string]string "Erro interno"
// @Router /product/{id} [put]
func (h *ProductHandler) Update(c *gin.Context) {
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

// GetByID retorna um produto pelo ID
// @Summary Obter Produto
// @Description Retorna os detalhes de um produto específico.
// @Tags Product
// @Produce json
// @Security Bearer
// @Param id path string true "ID do Produto"
// @Success 200 {object} models.Product
// @Failure 404 {object} map[string]string "Produto não encontrado"
// @Router /product/{id} [get]
func (h *ProductHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	res, err := h.Service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "produto não encontrado"})
		return
	}

	c.JSON(http.StatusOK, res)
}

// List retorna uma lista paginada de produtos
// @Summary Listar Produtos
// @Description Retorna produtos com paginação e filtros.
// @Tags Product
// @Produce json
// @Security Bearer
// @Param page query int false "Número da página (padrão: 1)"
// @Param size query int false "Itens por página (padrão: 25)"
// @Param searchWord query string false "Termo de busca"
// @Param searchFields query string false "Campos para busca (ex: name,sku,category)"
// @Param orderBy query string false "Campo para ordenação"
// @Param orderDirection query string false "Direção da ordenação (asc/desc)"
// @Success 200 {object} ProductListResponse
// @Router /product [get]
func (h *ProductHandler) List(c *gin.Context) {
	params := handler.ParseFilterParams(c)

	items, total, err := h.Service.List(c.Request.Context(), params)
	if err != nil {
		handler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"total": total,
		"page":  params.Page,
		"limit": params.Limit,
	})
}

// ListAll retorna todos os produtos sem paginação
// @Summary Listar Todos os Produtos
// @Description Retorna todos os produtos (usado para selects/lookups).
// @Tags Product
// @Produce json
// @Security Bearer
// @Param searchWord query string false "Termo de busca"
// @Param searchFields query string false "Campos para busca"
// @Param orderBy query string false "Campo para ordenação"
// @Param orderDirection query string false "Direção da ordenação"
// @Success 200 {object} ProductListAllResponse
// @Router /product/all [get]
func (h *ProductHandler) ListAll(c *gin.Context) {
	params := handler.ParseFilterParams(c)
	params.Filters["ignoreDefaultFilters"] = true

	items, total, err := h.Service.List(c.Request.Context(), params)
	if err != nil {
		handler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"total": total,
		"page":  params.Page,
		"limit": params.Limit,
	})
}

// Delete remove um produto
// @Summary Excluir Produto
// @Description Remove um produto logicamente ou fisicamente baseado no ID.
// @Tags Product
// @Produce json
// @Security Bearer
// @Param id path string true "ID do Produto"
// @Success 200 {object} map[string]string "message: produto excluído com sucesso"
// @Failure 500 {object} map[string]string "Erro interno"
// @Router /product/{id} [delete]
func (h *ProductHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.Service.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "produto excluído com sucesso"})
}

// SetStatus ativa ou desativa um produto
// @Summary Alterar Status do Produto
// @Description Define se um produto está ativo ou inativo.
// @Tags Product
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "ID do Produto"
// @Param request body UpdateStatusRequest true "Novo status"
// @Success 200 {object} map[string]string "message: status atualizado com sucesso"
// @Failure 500 {object} map[string]string "Erro interno"
// @Router /product/{id}/status [patch]
func (h *ProductHandler) SetStatus(c *gin.Context) {
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

	c.JSON(http.StatusOK, gin.H{"message": "status atualizado com sucesso", "active": body.Active})
}
