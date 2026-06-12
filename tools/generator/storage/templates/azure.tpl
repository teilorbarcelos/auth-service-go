package storage

import (
	"context"
	"fmt"
	"log"
)

type {{.Name}}Provider struct {
	Container string
}

func New{{.Name}}Provider(container string) *{{.Name}}Provider {
	log.Printf("{{.Name}}Provider inicializado com container: %s\n", container)
	return &{{.Name}}Provider{Container: container}
}

func (p *{{.Name}}Provider) Upload(ctx context.Context, filename string, data []byte) (string, error) {
	log.Printf("Upload para {{.Name}} container %s: %s\n", p.Container, filename)
	return fmt.Sprintf("https://account.blob.core.windows.net/%s/%s", p.Container, filename), nil
}

func (p *{{.Name}}Provider) Delete(ctx context.Context, filename string) error {
	log.Printf("Delete de {{.Name}} container %s: %s\n", p.Container, filename)
	return nil
}

func (p *{{.Name}}Provider) GetURL(ctx context.Context, filename string) (string, error) {
	return fmt.Sprintf("https://account.blob.core.windows.net/%s/%s", p.Container, filename), nil
}
