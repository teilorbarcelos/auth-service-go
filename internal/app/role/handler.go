package role

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"backend-go/internal/core/handler"
	"backend-go/internal/core/models"
	"backend-go/pkg/database"
	"context"
)

type RoleServiceI interface {
	ListFeatures(ctx context.Context) ([]models.Feature, error)
	Create(ctx context.Context, dto CreateRoleDTO) (*models.Role, error)
	Update(ctx context.Context, id string, dto CreateRoleDTO) (*models.Role, error)
	List(ctx context.Context, params database.FilterParams) ([]models.Role, int64, error)
	GetByID(ctx context.Context, id string) (*models.Role, error)
	Delete(ctx context.Context, id string) error
	SetStatus(ctx context.Context, id string, active bool) error
}

type RoleHandler struct {
	Service RoleServiceI
}

type RoleListResponse struct {
	Items []models.Role `json:"items"`
	Total int64         `json:"total"`
	Page  int           `json:"page"`
	Limit int           `json:"limit"`
}

type RoleListAllResponse struct {
	Items []models.Role `json:"items"`
	Total int64         `json:"total"`
}

func NewRoleHandler(service RoleServiceI) *RoleHandler {
	return &RoleHandler{Service: service}
}

type UpdateStatusRequest struct {
	Active bool `json:"active"`
}

// ListFeatures retorna todas as funcionalidades do sistema
// @Summary Listar Funcionalidades
// @Description Retorna a lista de todas as funcionalidades disponíveis para atribuição de permissões.
// @Tags Role
// @Produce json
// @Security Bearer
// @Success 200 {array} models.Feature
// @Failure 500 {object} map[string]string "Erro interno"
// @Router /role/features [get]
func (h *RoleHandler) ListFeatures(c *gin.Context) {
	res, err := h.Service.ListFeatures(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

// Create cria um novo papel
// @Summary Criar Papel (Role)
// @Description Cria um novo papel com permissões específicas.
// @Tags Role
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body CreateRoleDTO true "Dados do papel"
// @Success 201 {object} models.Role
// @Failure 400 {object} map[string]string "Dados inválidos"
// @Failure 500 {object} map[string]string "Erro interno"
// @Router /role [post]
func (h *RoleHandler) Create(c *gin.Context) {
	var dto CreateRoleDTO
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

// Update atualiza um papel existente
// @Summary Atualizar Papel (Role)
// @Description Atualiza os dados e permissões de um papel baseado no ID.
// @Tags Role
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "ID do Papel"
// @Param request body CreateRoleDTO true "Dados do papel"
// @Success 200 {object} models.Role
// @Failure 400 {object} map[string]string "Dados inválidos"
// @Failure 500 {object} map[string]string "Erro interno"
// @Router /role/{id} [put]
func (h *RoleHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var dto CreateRoleDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.Service.Update(c.Request.Context(), id, dto)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

// GetByID retorna um papel pelo ID
// @Summary Obter Papel (Role)
// @Description Retorna os detalhes de um papel e suas permissões.
// @Tags Role
// @Produce json
// @Security Bearer
// @Param id path string true "ID do Papel"
// @Success 200 {object} models.Role
// @Failure 404 {object} map[string]string "Papel não encontrado"
// @Router /role/{id} [get]
func (h *RoleHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	res, err := h.Service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "papel não encontrado"})
		return
	}

	c.JSON(http.StatusOK, res)
}

// List retorna uma lista paginada de papéis
// @Summary Listar Papéis (Roles)
// @Description Retorna papéis com paginação e filtros.
// @Tags Role
// @Produce json
// @Security Bearer
// @Param page query int false "Número da página (padrão: 1)"
// @Param size query int false "Itens por página (padrão: 25)"
// @Param searchWord query string false "Termo de busca"
// @Param searchFields query string false "Campos para busca (ex: name)"
// @Param orderBy query string false "Campo para ordenação"
// @Param orderDirection query string false "Direção da ordenação (asc/desc)"
// @Success 200 {object} RoleListResponse
// @Router /role [get]
func (h *RoleHandler) List(c *gin.Context) {
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

// ListAll retorna todos os papéis sem paginação
// @Summary Listar Todos os Papéis (Roles)
// @Description Retorna todos os papéis (usado para selects/lookups).
// @Tags Role
// @Produce json
// @Security Bearer
// @Param searchWord query string false "Termo de busca"
// @Param searchFields query string false "Campos para busca"
// @Param orderBy query string false "Campo para ordenação"
// @Param orderDirection query string false "Direção da ordenação"
// @Success 200 {object} RoleListAllResponse
// @Router /role/all [get]
func (h *RoleHandler) ListAll(c *gin.Context) {
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

// Delete remove um papel
// @Summary Excluir Papel (Role)
// @Description Remove um papel baseado no ID.
// @Tags Role
// @Produce json
// @Security Bearer
// @Param id path string true "ID do Papel"
// @Success 200 {object} map[string]string "message: papel excluído com sucesso"
// @Failure 500 {object} map[string]string "Erro interno"
// @Router /role/{id} [delete]
func (h *RoleHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.Service.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "papel excluído com sucesso"})
}

// SetStatus ativa ou desativa um papel
// @Summary Alterar Status do Papel (Role)
// @Description Define se um papel está ativo ou inativo.
// @Tags Role
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "ID do Papel"
// @Param request body UpdateStatusRequest true "Novo status"
// @Success 200 {object} map[string]string "message: status atualizado com sucesso"
// @Failure 500 {object} map[string]string "Erro interno"
// @Router /role/{id}/status [patch]
func (h *RoleHandler) SetStatus(c *gin.Context) {
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
