package storage

import "context"

type StorageProvider interface {
	Upload(ctx context.Context, filename string, data []byte) (string, error)
	Delete(ctx context.Context, filename string) error
	GetURL(ctx context.Context, filename string) (string, error)
}
