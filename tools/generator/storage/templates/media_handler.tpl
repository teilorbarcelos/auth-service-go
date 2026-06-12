package media

import (
	"io"
	"mime/multipart"
	"net/http"
	"github.com/gin-gonic/gin"
)

type MediaHandler struct {
	Service    *MediaService
	FileReader func(file *multipart.FileHeader) ([]byte, error)
}

func NewMediaHandler(service *MediaService) *MediaHandler {
	return &MediaHandler{
		Service: service,
		FileReader: func(file *multipart.FileHeader) ([]byte, error) {
			src, err := file.Open()
			if err != nil {
				return nil, err
			}
			defer src.Close()
			return io.ReadAll(src)
		},
	}
}

// Upload realiza o upload de um arquivo
// @Summary Upload de Arquivo
// @Description Recebe um arquivo via multipart/form-data e salva no storage configurado.
// @Tags Media
// @Accept multipart/form-data
// @Produce json
// @Security Bearer
// @Param file formData file true "Arquivo para upload"
// @Success 200 {object} map[string]string "URL do arquivo"
// @Failure 400 {object} map[string]string "Erro no arquivo"
// @Failure 500 {object} map[string]string "Erro interno"
// @Router /media/upload [post]
func (h *MediaHandler) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "arquivo não fornecido"})
		return
	}

	data, err := h.FileReader(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao processar arquivo"})
		return
	}

	url, err := h.Service.Upload(c.Request.Context(), file.Filename, data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":      url,
		"filename": file.Filename,
	})
}
