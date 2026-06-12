package pdf

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemotePdfProvider_GeneratePdf(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v1/pdf/generate", r.URL.Path)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			var req PdfRequestDTO
			err := json.NewDecoder(r.Body).Decode(&req)
			assert.NoError(t, err)
			assert.Equal(t, "test-template", req.Template)

			w.Header().Set("Content-Type", "application/pdf")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("%PDF-1.4 test content"))
		}))
		defer server.Close()

		provider := NewRemotePdfProvider(server.URL)
		request := PdfRequestDTO{
			Template: "test-template",
			Data:     map[string]interface{}{"key": "value"},
		}

		reader, err := provider.GeneratePdf(request)
		assert.NoError(t, err)
		assert.NotNil(t, reader)

		content, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.Equal(t, "%PDF-1.4 test content", string(content))
		reader.Close()
	})

	t.Run("service_error_status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		provider := NewRemotePdfProvider(server.URL)
		request := PdfRequestDTO{Template: "test"}

		reader, err := provider.GeneratePdf(request)
		assert.Error(t, err)
		assert.Nil(t, reader)
		assert.Contains(t, err.Error(), "pdf service returned status: 500")
	})

	t.Run("connection_error", func(t *testing.T) {
		provider := NewRemotePdfProvider("http://invalid-url-that-does-not-exist")
		request := PdfRequestDTO{Template: "test"}

		reader, err := provider.GeneratePdf(request)
		assert.NoError(t, err)
		assert.NotNil(t, reader)

		content, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.Contains(t, string(content), "Mock PDF Content")
		reader.Close()
	})

	t.Run("marshal_error", func(t *testing.T) {
		provider := NewRemotePdfProvider("http://localhost")
		request := PdfRequestDTO{
			Template: "test",
			Data: map[string]interface{}{
				"invalid": make(chan int), // Channels cannot be marshaled to JSON
			},
		}

		reader, err := provider.GeneratePdf(request)
		assert.Error(t, err)
		assert.Nil(t, reader)
		assert.Contains(t, err.Error(), "failed to marshal request")
	})
}
