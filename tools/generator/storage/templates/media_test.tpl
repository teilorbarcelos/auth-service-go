package media

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// MockStorage para testes de integração sem dependência de drivers reais
type MockStorage struct{}
func (s *MockStorage) Upload(ctx context.Context, filename string, data []byte) (string, error) {
	return fmt.Sprintf("https://storage.mock/%s", filename), nil
}
func (s *MockStorage) Delete(ctx context.Context, filename string) error { return nil }
func (s *MockStorage) GetURL(ctx context.Context, filename string) (string, error) { 
	return fmt.Sprintf("https://storage.mock/%s", filename), nil 
}

// ErroneousStorage para testar falhas
type ErroneousStorage struct{}
func (s *ErroneousStorage) Upload(ctx context.Context, filename string, data []byte) (string, error) {
	return "", fmt.Errorf("erro forçado")
}
func (s *ErroneousStorage) Delete(ctx context.Context, filename string) error { return nil }
func (s *ErroneousStorage) GetURL(ctx context.Context, filename string) (string, error) { return "", nil }

func TestMediaModule(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("RegisterRoutes", func(t *testing.T) {
		r := gin.Default()
		group := r.Group("/v1")
		RegisterRoutes(group)
		
		routes := r.Routes()
		found := false
		for _, route := range routes {
			if route.Path == "/v1/media/upload" {
				found = true
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("Upload Success", func(t *testing.T) {
		service := NewMediaService(&MockStorage{})
		handler := NewMediaHandler(service)
		r := gin.Default()
		r.POST("/upload", handler.Upload)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "test.txt")
		part.Write([]byte("hello world"))
		writer.Close()

		req, _ := http.NewRequest("POST", "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "https://storage.mock/test.txt")
	})

	t.Run("Default FileReader Error", func(t *testing.T) {
		handler := NewMediaHandler(nil)
		_, err := handler.FileReader(&multipart.FileHeader{Filename: "invalid"})
		assert.Error(t, err)
	})

	t.Run("Upload No File", func(t *testing.T) {
		handler := NewMediaHandler(NewMediaService(&MockStorage{}))
		r := gin.Default()
		r.POST("/upload", handler.Upload)

		req, _ := http.NewRequest("POST", "/upload", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Upload Storage Error", func(t *testing.T) {
		service := NewMediaService(&ErroneousStorage{})
		handler := NewMediaHandler(service)
		r := gin.Default()
		r.POST("/upload", handler.Upload)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "test.txt")
		part.Write([]byte("test"))
		writer.Close()

		req, _ := http.NewRequest("POST", "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Upload Custom Reader Error", func(t *testing.T) {
		service := NewMediaService(&MockStorage{})
		handler := NewMediaHandler(service)
		handler.FileReader = func(file *multipart.FileHeader) ([]byte, error) {
			return nil, fmt.Errorf("erro de leitura")
		}
		
		r := gin.Default()
		r.POST("/upload", handler.Upload)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "test.txt")
		part.Write([]byte("test"))
		writer.Close()

		req, _ := http.NewRequest("POST", "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
