package models

import "time"

// Report เป็นตัวแทนของข้อมูลตาราง reports ในฐานข้อมูล
type Report struct {
	ReportID   int       `json:"report_id"`
	UserID     int       `json:"user_id"`
	NovelID    int       `json:"novel_id"`
	Reason     string    `json:"reason"`
	ReportType string    `json:"report_type"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}
