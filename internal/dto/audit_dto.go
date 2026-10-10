package dto

import "time"

type AuditLogFilter struct {
	ActorUserID  *uint
	ActorKeyword string
	Action       string
	TargetType   string
	TargetID     *int64
	Status       string
	DateFrom     *time.Time
	DateTo       *time.Time
	Page         int
	Limit        int
}

type AuditLogListResponse struct {
	Items []AuditLogResponse `json:"items"`
	Page  int                `json:"page"`
	Limit int                `json:"limit"`
	Total int                `json:"total"`
}

type AuditLogResponse struct {
	LogID         int64       `json:"log_id"`
	ActorUserID   *uint       `json:"actor_user_id,omitempty"`
	ActorUsername string      `json:"actor_username,omitempty"`
	ActorRole     string      `json:"actor_role"`
	Action        string      `json:"action"`
	TargetType    string      `json:"target_type,omitempty"`
	TargetID      *int64      `json:"target_id,omitempty"`
	TargetName    string      `json:"target_name,omitempty"`
	Status        string      `json:"status"`
	IPAddress     string      `json:"ip_address,omitempty"`
	Metadata      interface{} `json:"metadata"`
	CreatedAt     time.Time   `json:"created_at"`
}

type AuditLogMetadataResponse struct {
	Actions     []string `json:"actions"`
	TargetTypes []string `json:"target_types"`
	Statuses    []string `json:"statuses"`
}
