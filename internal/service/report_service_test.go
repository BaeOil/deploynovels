package service

import (
	"context"
	"errors"
	"testing"

	"novel-be/internal/dto"
	"novel-be/internal/models"
)

type stubReportRepo struct {
	pendingAppeal bool
	createCalled  bool
}

func (s *stubReportRepo) CreateReport(ctx context.Context, report models.Report) error {
	return nil
}

func (s *stubReportRepo) GetReports(ctx context.Context, statusFilter string, reportType, search string, page, limit int) ([]dto.ReportResponse, int, error) {
	return nil, 0, nil
}

func (s *stubReportRepo) GetStatus(ctx context.Context, reportID int) (string, error) {
	return "pending", nil
}

func (s *stubReportRepo) GetReportDetail(ctx context.Context, reportID int) (string, string, int, string, int, string, bool, error) {
	return "pending", "report", 1, "Test Novel", 7, "published", true, nil
}

func (s *stubReportRepo) UpdateReportStatus(ctx context.Context, reportID int, req dto.UpdateReportStatusRequest) error {
	return nil
}

func (s *stubReportRepo) CreateAppeal(ctx context.Context, authorUserID int, appeal dto.CreateAppealRequest) error {
	s.createCalled = true
	if s.pendingAppeal {
		return errors.New("appeal already pending")
	}
	return nil
}

func (s *stubReportRepo) HasPendingAppeal(ctx context.Context, authorUserID int, novelID int) (bool, error) {
	return s.pendingAppeal, nil
}

func TestReportService_UpdateReportStatusRequiresReason(t *testing.T) {
	svc := &reportService{repo: &stubReportRepo{}}

	err := svc.UpdateReportStatus(context.Background(), 10, dto.UpdateReportStatusRequest{Status: "resolved", Reason: "   "})
	if err == nil || err.Error() != "admin reason is required" {
		t.Fatalf("expected admin reason validation, got: %v", err)
	}
}

func TestReportService_CreateAppealRejectsDuplicatePendingAppeal(t *testing.T) {
	repo := &stubReportRepo{pendingAppeal: true}
	svc := &reportService{repo: repo}

	err := svc.CreateAppeal(context.Background(), 99, dto.CreateAppealRequest{NovelID: 2, Reason: "please unban"})
	if err == nil || err.Error() != "appeal already pending for this novel" {
		t.Fatalf("expected duplicate appeal rejection, got: %v", err)
	}
	if repo.createCalled {
		t.Fatal("repository CreateAppeal should not be called when an active appeal already exists")
	}
}

func TestReportService_RejectsEmptyAppealReason(t *testing.T) {
	svc := &reportService{repo: &stubReportRepo{}}

	err := svc.CreateAppeal(context.Background(), 88, dto.CreateAppealRequest{NovelID: 5, Reason: ""})
	if err == nil || err.Error() != "appeal reason is required" {
		t.Fatalf("expected appeal reason validation, got: %v", err)
	}
}

func TestReportService_GetReportDetailReturnsReportMetadata(t *testing.T) {
	repo := &stubReportRepo{}
	svc := &reportService{repo: repo}

	status, reportType, novelID, title, authorID, novelStatus, novelIsPublished, err := svc.GetReportDetail(context.Background(), 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "pending" || reportType != "report" || novelID != 1 || title != "Test Novel" || authorID != 7 || novelStatus != "published" || !novelIsPublished {
		t.Fatalf("unexpected detail payload: status=%s type=%s novelID=%d title=%s authorID=%d novelStatus=%s published=%v", status, reportType, novelID, title, authorID, novelStatus, novelIsPublished)
	}
}
