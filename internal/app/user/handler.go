package user

import (
	"context"
	"io"
	"net/http"

	"backend-go/internal/core/handler"
	"backend-go/internal/core/models"
	"backend-go/pkg/database"

	"github.com/gin-gonic/gin"
)

type UserServiceI interface {
	Create(ctx context.Context, dto CreateUserDTO) (*models.User, error)
	Update(ctx context.Context, id string, dto UpdateUserDTO) (*models.User, error)
	List(ctx context.Context, params database.FilterParams) ([]models.User, int64, error)
	GetByID(ctx context.Context, id string) (*models.User, error)
	Delete(ctx context.Context, id string) error
	SetStatus(ctx context.Context, id string, active bool) error
	ExportPdf(ctx context.Context, params database.FilterParams) (io.ReadCloser, error)
}

type UserHandler struct {
	Service UserServiceI
}

type UserListResponse struct {
	Items []models.User `json:"items"`
	Total int64         `json:"total"`
	Page  int           `json:"page"`
	Limit int           `json:"limit"`
}

type UserListAllResponse struct {
	Items []models.User `json:"items"`
	Total int64         `json:"total"`
}

func NewUserHandler(service UserServiceI) *UserHandler {
	return &UserHandler{Service: service}
}

type UpdateStatusRequest struct {
	Active bool `json:"active"`
}

// Create cria um novo usuário
// @Summary Criar Usuário
// @Description Cria um novo usuário com os dados fornecidos.
// @Tags User
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body CreateUserDTO true "Dados do usuário"
// @Success 201 {object} models.User
// @Failure 400 {object} map[string]string "Dados inválidos"
// @Failure 500 {object} map[string]string "Erro interno"
// @Router /user [post]
func (h *UserHandler) Create(c *gin.Context) {
	var dto CreateUserDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.Service.Create(c.Request.Context(), dto)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, res)
}

// Update atualiza um usuário existente
// @Summary Atualizar Usuário
// @Description Atualiza os dados de um usuário baseado no ID.
// @Tags User
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "ID do Usuário"
// @Param request body UpdateUserDTO true "Campos a serem atualizados"
// @Success 200 {object} models.User
// @Failure 400 {object} map[string]string "Dados inválidos"
// @Failure 500 {object} map[string]string "Erro interno"
// @Router /user/{id} [put]
func (h *UserHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var dto UpdateUserDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.Service.Update(c.Request.Context(), id, dto)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// GetByID retorna um usuário pelo ID
// @Summary Obter Usuário
// @Description Retorna os detalhes de um usuário específico.
// @Tags User
// @Produce json
// @Security Bearer
// @Param id path string true "ID do Usuário"
// @Success 200 {object} models.User
// @Failure 404 {object} map[string]string "Usuário não encontrado"
// @Router /user/{id} [get]
func (h *UserHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	res, err := h.Service.GetByID(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// List retorna uma lista paginada de usuários
// @Summary Listar Usuários
// @Description Retorna usuários com paginação e filtros.
// @Tags User
// @Produce json
// @Security Bearer
// @Param page query int false "Número da página (padrão: 1)"
// @Param size query int false "Itens por página (padrão: 25)"
// @Param searchWord query string false "Termo de busca"
// @Param searchFields query string false "Campos para busca (ex: name,email,Role.name)"
// @Param orderBy query string false "Campo para ordenação"
// @Param orderDirection query string false "Direção da ordenação (asc/desc)"
// @Success 200 {object} UserListResponse
// @Router /user [get]
func (h *UserHandler) List(c *gin.Context) {
	params := handler.ParseFilterParams(c)

	items, total, err := h.Service.List(c.Request.Context(), params)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"total": total,
		"page":  params.Page,
		"limit": params.Limit,
	})
}

// ListAll retorna todos os usuários sem paginação
// @Summary Listar Todos os Usuários
// @Description Retorna todos os usuários (usado para selects/lookups).
// @Tags User
// @Produce json
// @Security Bearer
// @Param searchWord query string false "Termo de busca"
// @Param searchFields query string false "Campos para busca"
// @Param orderBy query string false "Campo para ordenação"
// @Param orderDirection query string false "Direção da ordenação"
// @Success 200 {object} UserListAllResponse
// @Router /user/all [get]
func (h *UserHandler) ListAll(c *gin.Context) {
	params := handler.ParseFilterParams(c)
	params.Filters["ignoreDefaultFilters"] = true

	items, total, err := h.Service.List(c.Request.Context(), params)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"total": total,
		"page":  params.Page,
		"limit": params.Limit,
	})
}

// Delete remove um usuário
// @Summary Excluir Usuário
// @Description Remove um usuário logicamente baseado no ID.
// @Tags User
// @Produce json
// @Security Bearer
// @Param id path string true "ID do Usuário"
// @Success 200 {object} map[string]string "message: usuário excluído com sucesso"
// @Failure 500 {object} map[string]string "Erro interno"
// @Router /user/{id} [delete]
func (h *UserHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.Service.Delete(c.Request.Context(), id); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "usuário excluído com sucesso"})
}

// SetStatus ativa ou desativa um usuário
// @Summary Alterar Status do Usuário
// @Description Define se um usuário está ativo ou inativo.
// @Tags User
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "ID do Usuário"
// @Param request body UpdateStatusRequest true "Novo status"
// @Success 200 {object} map[string]string "message: status atualizado com sucesso"
// @Failure 500 {object} map[string]string "Erro interno"
// @Router /user/{id}/status [patch]
func (h *UserHandler) SetStatus(c *gin.Context) {
	id := c.Param("id")
	var body struct {
		Active bool `json:"active"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.Service.SetStatus(c.Request.Context(), id, body.Active); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "status atualizado com sucesso", "active": body.Active})
}

// ExportPdf exporta usuários para PDF
// @Summary Exportar Usuários para PDF
// @Description Exporta a lista filtrada de usuários para um arquivo PDF.
// @Tags User
// @Produce application/pdf
// @Security Bearer
// @Param searchWord query string false "Termo de busca"
// @Param searchFields query string false "Campos para busca"
// @Param orderBy query string false "Campo para ordenação"
// @Param orderDirection query string false "Direção da ordenação"
// @Success 200 {file} []byte "PDF file"
// @Failure 400 {object} map[string]string "Dados inválidos"
// @Failure 500 {object} map[string]string "Erro interno"
// @Router /user/export/pdf [get]
func (h *UserHandler) ExportPdf(c *gin.Context) {
	params := handler.ParseFilterParams(c)
	params.Limit = 0

	pdfStream, err := h.Service.ExportPdf(c.Request.Context(), params)
	if err != nil {
		h.handleError(c, err)
		return
	}
	defer pdfStream.Close()

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", `attachment; filename="usuarios.pdf"`)

	if _, err := io.Copy(c.Writer, pdfStream); err != nil {
		return
	}
}

func (h *UserHandler) handleError(c *gin.Context, err error) {
	handler.HandleError(c, err)
}
