package models

import (
	"encoding/json"
	"time"
)

type AuditLog struct {
	LogID         int64           `json:"log_id"`
	ActorUserID   *uint           `json:"actor_user_id,omitempty"`
	ActorUsername string          `json:"actor_username,omitempty"`
	ActorRole     string          `json:"actor_role"`
	Action        string          `json:"action"`
	TargetType    string          `json:"target_type,omitempty"`
	TargetID      *int64          `json:"target_id,omitempty"`
	TargetName    string          `json:"target_name,omitempty"`
	Status        string          `json:"status"`
	IPAddress     string          `json:"ip_address,omitempty"`
	Metadata      json.RawMessage `json:"metadata"`
	CreatedAt     time.Time       `json:"created_at"`
}
