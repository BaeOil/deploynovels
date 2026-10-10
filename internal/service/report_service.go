package service

import (
	"context"
	"errors"
	"novel-be/internal/dto"
	"novel-be/internal/models"
	"novel-be/internal/repository"
	"strings"
)

var ErrInvalidReportTransition = errors.New("invalid report transition")

type ReportService interface {
	CreateReport(ctx context.Context, userID int, req dto.CreateReportRequest) error
	GetReports(ctx context.Context, statusFilter string, reportType, search string, page, limit int) ([]dto.ReportResponse, int, error)
	GetReportStatus(ctx context.Context, reportID int) (string, error)
	GetReportDetail(ctx context.Context, reportID int) (status string, reportType string, novelID int, novelTitle string, authorID int, novelStatus string, novelIsPublished bool, err error)
	UpdateReportStatus(ctx context.Context, reportID int, req dto.UpdateReportStatusRequest) error
	CreateAppeal(ctx context.Context, userID int, req dto.CreateAppealRequest) error
}

type reportService struct {
	repo repository.ReportRepository
}

func NewReportService(repo repository.ReportRepository) ReportService {
	return &reportService{repo: repo}
}

// 📌 1. สร้างรีพอร์ต (ฝั่งคนอ่าน)
func (s *reportService) CreateReport(ctx context.Context, userID int, req dto.CreateReportRequest) error {
	if req.NovelID <= 0 {
		return errors.New("invalid novel ID")
	}
	if req.Reason == "" {
		return errors.New("reason is required")
	}

	report := models.Report{
		UserID:  userID,
		NovelID: req.NovelID,
		Reason:  req.Reason,
	}

	return s.repo.CreateReport(ctx, report)
}

// 📌 2. ดึงรีพอร์ตให้แอดมินดูแบบ dynamic filter
func (s *reportService) GetReports(ctx context.Context, statusFilter string, reportType, search string, page, limit int) ([]dto.ReportResponse, int, error) {
	return s.repo.GetReports(ctx, statusFilter, reportType, search, page, limit)
}

// 📌 3. อัปเดตสถานะรีพอร์ต (ฝั่งแอดมิน)
func (s *reportService) UpdateReportStatus(ctx context.Context, reportID int, req dto.UpdateReportStatusRequest) error {
	if req.Status != "resolved" && req.Status != "rejected" {
		return errors.New("invalid status, must be resolved or rejected")
	}
	if strings.TrimSpace(req.Reason) == "" {
		return errors.New("admin reason is required")
	}
	currentStatus, reportType, _, _, _, _, _, err := s.repo.GetReportDetail(ctx, reportID)
	if err != nil {
		return err
	}
	if (reportType == "report" && currentStatus != "pending") || (reportType == "appeal" && currentStatus != "appeal_pending") {
		return ErrInvalidReportTransition
	}

	return s.repo.UpdateReportStatus(ctx, reportID, req)
}

func (s *reportService) CreateAppeal(ctx context.Context, userID int, req dto.CreateAppealRequest) error {
	if req.NovelID <= 0 {
		return errors.New("invalid novel ID")
	}
	if strings.TrimSpace(req.Reason) == "" {
		return errors.New("appeal reason is required")
	}

	hasPending, err := s.repo.HasPendingAppeal(ctx, userID, req.NovelID)
	if err != nil {
		return err
	}
	if hasPending {
		return errors.New("appeal already pending for this novel")
	}

	return s.repo.CreateAppeal(ctx, userID, req)
}

func (s *reportService) GetReportStatus(ctx context.Context, reportID int) (string, error) {
	return s.repo.GetStatus(ctx, reportID)
}

func (s *reportService) GetReportDetail(ctx context.Context, reportID int) (string, string, int, string, int, string, bool, error) {
	return s.repo.GetReportDetail(ctx, reportID)
}
