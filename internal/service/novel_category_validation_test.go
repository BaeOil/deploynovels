package service

import (
	"context"
	"errors"
	"testing"

	"novel-be/internal/models"
	"novel-be/internal/repository"
)

// fakeCategoryRepo สำหรับตรวจ existence ของ category โดยไม่ต้องพึ่ง DB จริง
type fakeCategoryRepo struct {
	existing map[int]bool
}

func (f *fakeCategoryRepo) GetAllCategories() ([]models.Category, error) { return nil, nil }
func (f *fakeCategoryRepo) GetCategoryByID(_ context.Context, id int) (*models.Category, error) {
	if f.existing[id] {
		return &models.Category{CategoryID: id, Name: "category"}, nil
	}
	return nil, repository.ErrCategoryNotFound
}
func (f *fakeCategoryRepo) GetCategoryByName(context.Context, string) (*models.Category, error) {
	return nil, repository.ErrCategoryNotFound
}
func (f *fakeCategoryRepo) CreateCategory(context.Context, string) (*models.Category, error) {
	return nil, nil
}
func (f *fakeCategoryRepo) UpdateCategory(context.Context, int, string) (*models.Category, error) {
	return nil, nil
}
func (f *fakeCategoryRepo) DeleteCategory(context.Context, int) error { return nil }
func (f *fakeCategoryRepo) CountUsageByID(context.Context, int) (int, error) {
	return 0, nil
}

var _ repository.CategoryRepository = (*fakeCategoryRepo)(nil)

// VAL-06: category ID ต้องมีอยู่จริง / ไม่ส่งหมวดหมู่ก็ยังใช้ได้ตาม behavior เดิม
func TestValidateCategoryIDsExistence(t *testing.T) {
	svc := &novelService{
		categoryRepo: &fakeCategoryRepo{existing: map[int]bool{1: true, 2: true, 3: true}},
	}

	tests := []struct {
		name    string
		ids     []int
		wantErr bool
	}{
		{"0 categories", nil, false},
		{"existing categories", []int{1, 2, 3}, false},
		{"one missing category", []int{1, 99}, true},
		{"all missing", []int{98, 99}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.validateCategoryIDs(tt.ids)
			if tt.wantErr {
				if !errors.Is(err, repository.ErrCategoryNotFound) {
					t.Fatalf("expected repository.ErrCategoryNotFound, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
