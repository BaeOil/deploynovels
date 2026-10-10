package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"novel-be/internal/dto"
	"novel-be/internal/repository"
)

type DashboardService interface {
	GetSummary(ctx context.Context) (dto.DashboardSummary, error)
	GetTrend(ctx context.Context, months int, timezone string) (dto.DashboardTrend, error)
}

type dashboardService struct {
	repo repository.DashboardRepository
}

func NewDashboardService(repo repository.DashboardRepository) DashboardService {
	return &dashboardService{repo: repo}
}

func (s *dashboardService) GetSummary(ctx context.Context) (dto.DashboardSummary, error) {
	return s.repo.GetSummary(ctx)
}

func (s *dashboardService) GetTrend(ctx context.Context, months int, timezone string) (dto.DashboardTrend, error) {
	if months < 1 || months > 24 {
		return dto.DashboardTrend{}, fmt.Errorf("months must be between 1 and 24")
	}
	timezone = strings.TrimSpace(timezone)
	if timezone == "" {
		timezone = "Asia/Bangkok"
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return dto.DashboardTrend{}, fmt.Errorf("invalid timezone")
	}
	return s.repo.GetTrend(ctx, months, timezone)
}
