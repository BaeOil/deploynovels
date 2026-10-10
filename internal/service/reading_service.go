package service

import (
	"novel-be/internal/models"
	"novel-be/internal/repository"
)

type readingService struct {
	repo repository.ReadingRepository
}

func NewReadingService(repo repository.ReadingRepository) ReadingService {
	return &readingService{repo: repo}
}

func (s *readingService) GetProgress(userID, novelID int) (*models.ReadingProgress, error) {
	return s.repo.GetReadingProgress(userID, novelID)
}

func (s *readingService) SaveProgress(progress models.ReadingProgress) error {
	return s.repo.SaveReadingProgress(progress.UserID, progress.NovelID, progress.CurrentSceneID)
}

func (s *readingService) ResetProgress(userID, novelID int) error {
	return s.repo.ResetReadingProgress(userID, novelID)
}

func (s *readingService) GetReadingHistory(userID int) ([]models.Novel, error) {
	return s.repo.GetReadingHistory(userID)
}

func (s *readingService) RecordChoiceHistory(history models.ChoiceHistory) error {
	return s.repo.InsertChoiceHistory(history)
}

func (s *readingService) RecordEnding(userID, novelID, sceneID int) error {
	return s.repo.InsertUserEnding(userID, novelID, sceneID)
}

func (s *readingService) DeleteReadingHistoryByNovel(userID, novelID int) (bool, error) {
	return s.repo.DeleteReadingHistoryByNovel(userID, novelID)
}

func (s *readingService) DeleteReadingHistoryByUser(userID int, novelIDs []int) (int, error) {
	return s.repo.DeleteReadingHistoryByUser(userID, novelIDs)
}
