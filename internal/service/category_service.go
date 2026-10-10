package service

import (
	"context"
	"errors"
	"strings"

	"novel-be/internal/models"
	"novel-be/internal/repository"
)

var (
	ErrInvalidCategoryName = errors.New("invalid category name")
	ErrCategoryNotFound    = repository.ErrCategoryNotFound
	ErrCategoryDuplicate   = repository.ErrCategoryDuplicate
	ErrCategoryInUse       = repository.ErrCategoryInUse
)

type CategoryService interface {
	GetCategories() ([]models.Category, error)
	GetCategory(ctx context.Context, id int) (*models.Category, error)
	CreateCategory(ctx context.Context, name string) (*models.Category, error)
	UpdateCategory(ctx context.Context, id int, name string) (*models.Category, error)
	DeleteCategory(ctx context.Context, id int) error
}

type categoryService struct {
	repo repository.CategoryRepository
}

func NewCategoryService(repo repository.CategoryRepository) CategoryService {
	return &categoryService{repo: repo}
}

func (s *categoryService) GetCategories() ([]models.Category, error) {
	// เรียกใช้งาน Repository เพื่อดึงข้อมูล
	return s.repo.GetAllCategories()
}

func (s *categoryService) GetCategory(ctx context.Context, id int) (*models.Category, error) {
	return s.repo.GetCategoryByID(ctx, id)
}

func (s *categoryService) CreateCategory(ctx context.Context, name string) (*models.Category, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil, ErrInvalidCategoryName
	}

	// Check case-insensitive duplicate
	existing, err := s.repo.GetCategoryByName(ctx, trimmedName)
	if err == nil && existing != nil {
		return nil, ErrCategoryDuplicate
	} else if err != nil && !errors.Is(err, repository.ErrCategoryNotFound) {
		return nil, err
	}

	return s.repo.CreateCategory(ctx, trimmedName)
}

func (s *categoryService) UpdateCategory(ctx context.Context, id int, name string) (*models.Category, error) {
	if id <= 0 {
		return nil, repository.ErrCategoryNotFound
	}

	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil, ErrInvalidCategoryName
	}

	// Check if category exists
	c, err := s.repo.GetCategoryByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check case-insensitive duplicate with another category
	existing, err := s.repo.GetCategoryByName(ctx, trimmedName)
	if err == nil && existing != nil && existing.CategoryID != c.CategoryID {
		return nil, ErrCategoryDuplicate
	} else if err != nil && !errors.Is(err, repository.ErrCategoryNotFound) {
		return nil, err
	}

	return s.repo.UpdateCategory(ctx, id, trimmedName)
}

func (s *categoryService) DeleteCategory(ctx context.Context, id int) error {
	if id <= 0 {
		return repository.ErrCategoryNotFound
	}

	// Check if category exists
	_, err := s.repo.GetCategoryByID(ctx, id)
	if err != nil {
		return err
	}

	// Check if category is referenced by novel_categories or writer_categories
	count, err := s.repo.CountUsageByID(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrCategoryInUse
	}

	return s.repo.DeleteCategory(ctx, id)
}
