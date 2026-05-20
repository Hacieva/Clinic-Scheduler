package service

import (
	"context"

	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
	"github.com/Hacieva/clinic-scheduler/backend/internal/repository"
)

type DirectionService struct {
	repo repository.DirectionRepository
}

func NewDirectionService(repo repository.DirectionRepository) *DirectionService {
	return &DirectionService{repo: repo}
}

func (s *DirectionService) List(ctx context.Context) ([]model.Direction, error) {
	return s.repo.List(ctx)
}

func (s *DirectionService) GetByID(ctx context.Context, id int64) (*model.Direction, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *DirectionService) Create(ctx context.Context, name string, description *string) (*model.Direction, error) {
	return s.repo.Create(ctx, name, description)
}

func (s *DirectionService) Update(ctx context.Context, id int64, name string, description *string) (*model.Direction, error) {
	return s.repo.Update(ctx, id, name, description)
}

func (s *DirectionService) Delete(ctx context.Context, id int64) error {
	return s.repo.SoftDelete(ctx, id)
}
