package models

import "time"

type Like struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	NovelID   int       `json:"novel_id"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

type Comment struct {
	CommentID int       `json:"comment_id"`
	UserID    int       `json:"user_id"`
	NovelID   int       `json:"novel_id"`
	SceneID   *int      `json:"scene_id,omitempty"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	Username  string    `json:"username,omitempty"`
}

type Follow struct {
	ID          int       `json:"id"`
	FollowerID  int       `json:"follower_id"`
	FollowingID int       `json:"following_id"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
}
