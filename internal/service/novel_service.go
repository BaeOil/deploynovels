package service

import (
	"context"
	"strings"

	"novel-be/internal/models"
	"novel-be/internal/repository"
)

type novelService struct {
	repo         repository.NovelRepository
	mediaService MediaService
	categoryRepo repository.CategoryRepository
}

func NewNovelService(repo repository.NovelRepository, mediaService MediaService, categoryRepo repository.CategoryRepository) NovelService {
	return &novelService{
		repo:         repo,
		mediaService: mediaService,
		categoryRepo: categoryRepo,
	}
}

// validateCategoryIDs ตรวจว่า category ID ที่ส่งมา มีอยู่จริงในระบบ (VAL-06)
// ถ้าไม่มี repository.ErrCategoryNotFound จะถูกส่งขึ้นไปให้ handler map เป็น 400
func (s *novelService) validateCategoryIDs(categoryIDs []int) error {
	if len(categoryIDs) == 0 || s.categoryRepo == nil {
		return nil
	}
	for _, id := range categoryIDs {
		if _, err := s.categoryRepo.GetCategoryByID(context.Background(), id); err != nil {
			return err
		}
	}
	return nil
}

// 🟢 ปรับเพื่อให้หน้า Home ดึงรูปไปโชว์ได้
func (s *novelService) ListNovels() ([]models.Novel, error) {
	novels, err := s.repo.ListNovels()
	if err != nil {
		return nil, err
	}
	// ไม่ต้องทำอะไรเพิ่มที่นี่ เพราะเราไปจัดการ URL ที่ Frontend (HomePage.jsx) แล้ว
	return novels, nil
}

func (s *novelService) ListAdminNovels(ctx context.Context, search, status string, categoryID, page, limit int) ([]models.Novel, int, error) {
	return s.repo.ListAdminNovels(ctx, strings.TrimSpace(search), strings.ToLower(strings.TrimSpace(status)), categoryID, page, limit)
}

// 🟢 ปรับเพื่อให้หน้ารายละเอียดโชว์รูปได้
func (s *novelService) GetNovelDetail(id int) (interface{}, error) {
	return s.repo.GetNovelByID(id)
}

func (s *novelService) IncrementViews(novelID int) error {
	return s.repo.IncrementViews(novelID)
}

func (s *novelService) GetNovelsByAuthorID(authorID int) ([]models.Novel, error) {
	return s.repo.GetNovelsByAuthorID(authorID)
}

func (s *novelService) CreateNovel(novel models.Novel) (int, error) {
	if err := s.validateCategoryIDs(novel.CategoryIDs); err != nil {
		return 0, err
	}
	return s.repo.CreateNovel(novel)
}

func (s *novelService) UpdateNovel(novel models.Novel) error {
	if err := s.validateCategoryIDs(novel.CategoryIDs); err != nil {
		return err
	}
	return s.repo.UpdateNovel(novel)
}

func (s *novelService) UpdateNovelCover(id int, url string) error {
	return s.repo.UpdateCoverImage(id, url)
}

func (s *novelService) DeleteNovel(id int) error {
	return s.repo.DeleteNovel(id)
}

func (s *novelService) SuspendNovel(ctx context.Context, novelID int) error {
	return s.repo.SuspendNovel(ctx, novelID)
}

// UnbanNovel allows admin to unban a novel, setting status to 'draft' and clearing ban flags.
func (s *novelService) UnbanNovel(ctx context.Context, novelID int) error {
	return s.repo.UnbanNovel(ctx, novelID)
}

// 🟢 ฟังก์ชันช่วยเช็ค (ถ้าไฟล์อื่นเรียกใช้)
func containsHTTP(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}
