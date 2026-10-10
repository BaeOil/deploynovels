package service

import (
	"novel-be/internal/dto"
	"novel-be/internal/models"
	"novel-be/internal/repository"
)

type socialService struct {
	repo repository.SocialRepository
}

func NewSocialService(repo repository.SocialRepository) SocialService {
	return &socialService{repo: repo}
}

func (s *socialService) AddLike(like models.Like) error {
	return s.repo.AddLike(like)
}

func (s *socialService) RemoveLike(userID, novelID int) error {
	return s.repo.RemoveLike(userID, novelID)
}

func (s *socialService) IsLikeExists(userID, novelID int) (bool, error) {
	return s.repo.IsLikeExists(userID, novelID)
}

func (s *socialService) AddToBookshelf(userID, novelID int) error {
	return s.repo.AddToBookshelf(userID, novelID)
}

func (s *socialService) RemoveFromBookshelf(userID, novelID int) error {
	return s.repo.RemoveFromBookshelf(userID, novelID)
}

func (s *socialService) GetBookshelfByUserID(userID int) ([]models.Novel, error) {
	return s.repo.GetBookshelfByUserID(userID)
}

func (s *socialService) GetBookshelfCountByNovelID(novelID int) (int, error) {
	return s.repo.GetBookshelfCountByNovelID(novelID)
}

func (s *socialService) GetBookshelfCountsByAuthorID(authorID int) ([]models.Novel, error) {
	return s.repo.GetBookshelfCountsByAuthorID(authorID)
}

func (s *socialService) AddComment(comment models.Comment) (int, error) {
	return s.repo.AddComment(comment)
}

func (s *socialService) RemoveComment(commentID, userID int) error {
	return s.repo.RemoveComment(commentID, userID)
}

func (s *socialService) AddFollow(follow models.Follow) error {
	return s.repo.AddFollow(follow)
}

func (s *socialService) RemoveFollow(userID, writerID int) error {
	return s.repo.RemoveFollow(userID, writerID)
}

func (s *socialService) GetFollowingWriters(userID int) ([]models.Writer, error) {
	return s.repo.GetFollowingWriters(userID)
}

func (s *socialService) GetCommentCountByNovelID(novelID int) (int, error) {
	return s.repo.GetCommentCountByNovelID(novelID)
}

func (s *socialService) GetCommentsByNovelID(novelID int) ([]dto.CommentDetailDTO, error) {
	comments, err := s.repo.GetCommentsByNovelID(novelID)
	if err != nil {
		return nil, err
	}

	result := make([]dto.CommentDetailDTO, len(comments))
	for i, c := range comments {
		result[i] = dto.CommentDetailDTO{
			CommentID: c.CommentID,
			UserID:    c.UserID,
			Username:  c.Username,
			NovelID:   c.NovelID,
			SceneID:   c.SceneID,
			Content:   c.Content,
			CreatedAt: c.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}
	return result, nil
}

func (s *socialService) GetCommentsBySceneID(sceneID int) ([]dto.CommentDetailDTO, error) {
	comments, err := s.repo.GetCommentsBySceneID(sceneID)
	if err != nil {
		return nil, err
	}

	result := make([]dto.CommentDetailDTO, len(comments))
	for i, c := range comments {
		result[i] = dto.CommentDetailDTO{
			CommentID: c.CommentID,
			UserID:    c.UserID,
			Username:  c.Username,
			NovelID:   c.NovelID,
			SceneID:   c.SceneID,
			Content:   c.Content,
			CreatedAt: c.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}
	return result, nil
}
