package service

import (
	"context"

	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
	"github.com/Hacieva/clinic-scheduler/backend/internal/repository"
)

// ServiceCatalogService manages the global services catalog.
// Services here have no owner; assignment to doctors is done via DoctorAssignmentService.
type ServiceCatalogService struct {
	repo repository.ServiceRepository
}

func NewServiceCatalogService(repo repository.ServiceRepository) *ServiceCatalogService {
	return &ServiceCatalogService{repo: repo}
}

// CatalogServiceInput carries fields for creating or updating a catalog service.
type CatalogServiceInput struct {
	DirectionID     *int64 // optional — specialisation grouping
	Category        *string
	Name            string
	Description     *string
	DurationMinutes int
	Price           *int64 // kopecks
}

func (s *ServiceCatalogService) ListAll(ctx context.Context, activeOnly bool) ([]model.Service, error) {
	return s.repo.ListAll(ctx, activeOnly)
}

func (s *ServiceCatalogService) GetByID(ctx context.Context, id int64) (*model.Service, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ServiceCatalogService) Create(ctx context.Context, input CatalogServiceInput) (*model.Service, error) {
	return s.repo.Create(ctx, repository.CreateServiceInput{
		DoctorID:        nil, // global catalog service — no doctor owner
		DirectionID:     input.DirectionID,
		Category:        input.Category,
		Name:            input.Name,
		Description:     input.Description,
		DurationMinutes: input.DurationMinutes,
		Price:           input.Price,
	})
}

func (s *ServiceCatalogService) Update(ctx context.Context, id int64, input CatalogServiceInput) (*model.Service, error) {
	return s.repo.Update(ctx, id, repository.UpdateServiceInput{
		DirectionID:     input.DirectionID,
		Category:        input.Category,
		Name:            input.Name,
		Description:     input.Description,
		DurationMinutes: input.DurationMinutes,
		Price:           input.Price,
	})
}

func (s *ServiceCatalogService) Delete(ctx context.Context, id int64) error {
	return s.repo.SoftDelete(ctx, id)
}
