package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"novel-be/internal/dto"
	"novel-be/internal/middleware"
	"novel-be/internal/models"
	"novel-be/internal/service"
)

type analyticsTestNovelService struct {
	novel models.Novel
}

func (s *analyticsTestNovelService) ListNovels() ([]models.Novel, error) { return nil, nil }
func (s *analyticsTestNovelService) ListAdminNovels(context.Context, string, string, int, int, int) ([]models.Novel, int, error) {
	return nil, 0, nil
}
func (s *analyticsTestNovelService) GetNovelDetail(int) (interface{}, error) { return &s.novel, nil }
func (s *analyticsTestNovelService) IncrementViews(int) error                { return nil }
func (s *analyticsTestNovelService) GetNovelsByAuthorID(int) ([]models.Novel, error) {
	return nil, nil
}
func (s *analyticsTestNovelService) CreateNovel(models.Novel) (int, error)   { return 0, nil }
func (s *analyticsTestNovelService) UpdateNovel(models.Novel) error          { return nil }
func (s *analyticsTestNovelService) UpdateNovelCover(int, string) error      { return nil }
func (s *analyticsTestNovelService) DeleteNovel(int) error                   { return nil }
func (s *analyticsTestNovelService) SuspendNovel(context.Context, int) error { return nil }
func (s *analyticsTestNovelService) UnbanNovel(context.Context, int) error   { return nil }

type analyticsTestWriterService struct {
	writer *models.Writer
}

func (s *analyticsTestWriterService) GetWriterByID(int) (*models.Writer, error) { return s.writer, nil }
func (s *analyticsTestWriterService) GetWriterByUserID(int) (*models.Writer, error) {
	if s.writer == nil {
		return nil, errors.New("writer not found")
	}
	return s.writer, nil
}
func (s *analyticsTestWriterService) GetLatestWriterApplicationByUserID(int) (*models.Writer, error) {
	return nil, nil
}
func (s *analyticsTestWriterService) ApplyForWriter(context.Context, uint, dto.WriterApplyRequest) error {
	return nil
}
func (s *analyticsTestWriterService) GetPendingRequests(context.Context, string, int, int) ([]dto.WriterRequestResponse, error) {
	return nil, nil
}
func (s *analyticsTestWriterService) ApproveWriter(context.Context, uint, uint) error { return nil }
func (s *analyticsTestWriterService) RejectWriter(context.Context, uint, uint, string) error {
	return nil
}
func (s *analyticsTestWriterService) UpdateWriterProfile(context.Context, int, dto.UpdateWriterProfileRequest) error {
	return nil
}

type analyticsTestService struct{}

func (s *analyticsTestService) GetNovelOverview(int) (*models.NovelOverviewStats, error) {
	return &models.NovelOverviewStats{}, nil
}
func (s *analyticsTestService) GetSceneAnalytics(int, int) (*models.SceneAnalyticsStats, error) {
	return &models.SceneAnalyticsStats{}, nil
}
func (s *analyticsTestService) GetSceneChoiceAnalytics(int, int) (*models.SceneChoiceAnalyticsStats, error) {
	return &models.SceneChoiceAnalyticsStats{}, nil
}
func (s *analyticsTestService) GetAllScenesAnalytics(int) ([]models.AllScenesAnalyticsStats, error) {
	return []models.AllScenesAnalyticsStats{}, nil
}
func (s *analyticsTestService) GetEdgeAnalytics(int) ([]models.EdgeAnalyticsStats, error) {
	return []models.EdgeAnalyticsStats{}, nil
}

var (
	_ service.NovelService     = (*analyticsTestNovelService)(nil)
	_ service.WriterService    = (*analyticsTestWriterService)(nil)
	_ service.AnalyticsService = (*analyticsTestService)(nil)
)

func analyticsRequest(role string, userID uint) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/writer/novels/7/analytics", nil)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, middleware.RoleKey, role)
	return req.WithContext(ctx)
}

func TestNovelAnalyticsAuthorization(t *testing.T) {
	tests := []struct {
		name       string
		role       string
		userID     uint
		writer     *models.Writer
		expectCode int
	}{
		{
			name:       "approved writer owner",
			role:       "writer",
			userID:     10,
			writer:     &models.Writer{WriterID: 7, UserID: 10, Status: "approved"},
			expectCode: http.StatusOK,
		},
		{
			name:       "reader",
			role:       "reader",
			userID:     20,
			expectCode: http.StatusForbidden,
		},
		{
			name:       "revoked writer",
			role:       "writer",
			userID:     30,
			expectCode: http.StatusForbidden,
		},
		{
			name:       "approved writer not owner",
			role:       "writer",
			userID:     40,
			writer:     &models.Writer{WriterID: 8, UserID: 40, Status: "approved"},
			expectCode: http.StatusForbidden,
		},
		{
			name:       "admin",
			role:       "admin",
			userID:     1,
			expectCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			novelService := &analyticsTestNovelService{novel: models.Novel{ID: 7, AuthorID: 7, IsPublished: true}}
			handler := NovelAnalyticsHandler(
				&analyticsTestService{},
				novelService,
				&analyticsTestWriterService{writer: tt.writer},
			)
			response := httptest.NewRecorder()

			handler(response, analyticsRequest(tt.role, tt.userID))

			if response.Code != tt.expectCode {
				t.Fatalf("expected HTTP %d, got %d", tt.expectCode, response.Code)
			}
		})
	}
}
