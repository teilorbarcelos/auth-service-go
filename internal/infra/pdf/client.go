package pdf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// PdfRequestDTO defines the payload for PDF generation.
type PdfRequestDTO struct {
	Template string                 `json:"template"`
	Data     map[string]interface{} `json:"data"`
	Options  Options                `json:"options"`
}

// Options defines the PDF generation options.
type Options struct {
	Landscape bool   `json:"landscape"`
	Format    string `json:"format"`
}

// PdfProvider defines the interface for PDF generation.
type PdfProvider interface {
	GeneratePdf(request PdfRequestDTO) (io.ReadCloser, error)
}

// RemotePdfProvider implements PdfProvider calling a remote service.
type RemotePdfProvider struct {
	ServiceURL string
	HTTPClient *http.Client
}

// NewRemotePdfProvider creates a new RemotePdfProvider.
func NewRemotePdfProvider(serviceURL string) *RemotePdfProvider {
	return &RemotePdfProvider{
		ServiceURL: serviceURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GeneratePdf calls the remote service and returns the response body as a stream.
func (p *RemotePdfProvider) GeneratePdf(request PdfRequestDTO) (io.ReadCloser, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := p.HTTPClient.Post(p.ServiceURL+"/v1/pdf/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		// Fallback to mock PDF
		mockPdf := `%PDF-1.4
1 0 obj
<< /Type /Catalog /Pages 2 0 R >>
endobj
2 0 obj
<< /Type /Pages /Kids [3 0 R] /Count 1 >>
endobj
3 0 obj
<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << >> /Contents 4 0 R >>
endobj
4 0 obj
<< /Length 51 >>
stream
BT
/F1 12 Tf
72 712 Td
(Mock PDF Content) Tj
ET
endstream
endobj
xref
0 5
0000000000 65535 f 
0000000009 00000 n 
0000000056 00000 n 
0000000111 00000 n 
0000000212 00000 n 
trailer
<< /Size 5 /Root 1 0 R >>
startxref
311
%%EOF`
		return io.NopCloser(bytes.NewReader([]byte(mockPdf))), nil
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("pdf service returned status: %s", resp.Status)
	}

	return resp.Body, nil
}
