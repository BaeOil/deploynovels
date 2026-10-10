package dto

import "time"

type DashboardSummary struct {
	TotalUsers            int64                `json:"total_users"`
	TotalNovelsPublished  int64                `json:"total_novels_published"`
	TotalWriters          int64                `json:"total_writers"`
	ActiveWriters30d      int64                `json:"active_writers_30d"`
	PendingWriterRequests DashboardQueue       `json:"pending_writer_requests"`
	PendingReports        DashboardReportQueue `json:"pending_reports"`
	AppealPending         DashboardReportQueue `json:"appeal_pending"`
}

type DashboardQueue struct {
	Count  int64                    `json:"count"`
	Recent []DashboardWriterRequest `json:"recent"`
}

type DashboardWriterRequest struct {
	UserID    int       `json:"user_id"`
	Username  string    `json:"username"`
	PenName   string    `json:"pen_name"`
	AppliedAt time.Time `json:"applied_at"`
}

type DashboardReportQueue struct {
	Count  int64                 `json:"count"`
	Recent []DashboardReportItem `json:"recent"`
}

type DashboardReportItem struct {
	ReportID   int       `json:"report_id"`
	NovelID    int       `json:"novel_id"`
	NovelTitle string    `json:"novel_title"`
	CreatedAt  time.Time `json:"created_at"`
}

type DashboardTrend struct {
	Months []DashboardTrendMonth `json:"months"`
}

type DashboardTrendMonth struct {
	Month     string `json:"month"`
	NewUsers  int64  `json:"new_users"`
	NewNovels int64  `json:"new_novels"`
}
