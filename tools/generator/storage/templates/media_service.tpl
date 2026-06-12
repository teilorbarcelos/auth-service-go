package media

import (
	"context"
	"backend-go/pkg/storage"
)

type MediaService struct {
	Storage storage.StorageProvider
}

func NewMediaService(storage storage.StorageProvider) *MediaService {
	return &MediaService{Storage: storage}
}

func (s *MediaService) Upload(ctx context.Context, filename string, data []byte) (string, error) {
	return s.Storage.Upload(ctx, filename, data)
}
