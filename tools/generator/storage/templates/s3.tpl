package storage

import (
	"context"
	"fmt"
	"log"
)

type {{.Name}}Provider struct {
	Bucket string
}

func New{{.Name}}Provider(bucket string) *{{.Name}}Provider {
	log.Printf("{{.Name}}Provider inicializado com bucket: %s\n", bucket)
	return &{{.Name}}Provider{Bucket: bucket}
}

func (p *{{.Name}}Provider) Upload(ctx context.Context, filename string, data []byte) (string, error) {
	log.Printf("Upload para {{.Name}} bucket %s: %s\n", p.Bucket, filename)
	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", p.Bucket, filename), nil
}

func (p *{{.Name}}Provider) Delete(ctx context.Context, filename string) error {
	log.Printf("Delete de {{.Name}} bucket %s: %s\n", p.Bucket, filename)
	return nil
}

func (p *{{.Name}}Provider) GetURL(ctx context.Context, filename string) (string, error) {
	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", p.Bucket, filename), nil
}
