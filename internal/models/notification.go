package models

import "time"

type Notification struct {
	ID             int       `json:"id"`
	UserID         int       `json:"user_id"`
	Type           string    `json:"type"`
	Title          *string   `json:"title,omitempty"`
	Message        string    `json:"message"`
	CoverImage     *string   `json:"cover_image,omitempty"`
	ReferenceID    *int      `json:"reference_id,omitempty"`
	ReferenceType  *string   `json:"reference_type,omitempty"`
	IsRead         bool      `json:"is_read"`
	CreatedAt      time.Time `json:"created_at"`
	ActorID        int       `json:"actor_id,omitempty"`
	ActorName      string    `json:"actor_name,omitempty"`
	ActorAvatar    string    `json:"actor_avatar,omitempty"`
	ActorColor     string    `json:"actor_color,omitempty"`
	ReferenceTitle string    `json:"reference_title,omitempty"`
}

type NotificationSettings struct {
	UserID             uint `json:"user_id" db:"user_id"`
	NovelUpdateEnabled bool `json:"novel_update_enabled" db:"novel_update_enabled"`
	FollowerEnabled    bool `json:"follower_enabled" db:"follower_enabled"`
	LikeEnabled        bool `json:"like_enabled" db:"like_enabled"`
	CommentEnabled     bool `json:"comment_enabled" db:"comment_enabled"`
	SystemEnabled      bool `json:"system_enabled" db:"system_enabled"`
}
