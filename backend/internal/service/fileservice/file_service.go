package fileservice

import (
	"auto-scan/internal/data/models"
	"auto-scan/internal/data/repository"
	"context"
)

type FileService struct {
	repo repository.FileRepository
}

func NewFileService(repo repository.FileRepository) *FileService {
	return &FileService{repo: repo}
}

func (s *FileService) ListFiles(ctx context.Context, filter repository.FileFilter) ([]*models.ScanFile, int, error) {
	return s.repo.List(ctx, filter)
}

func (s *FileService) GetFile(ctx context.Context, id string) (*models.ScanFile, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *FileService) DeleteFile(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *FileService) BatchDeleteFiles(ctx context.Context, ids []string) error {
	return s.repo.BatchDelete(ctx, ids)
}
