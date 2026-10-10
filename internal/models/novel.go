package models

import "time"

type Novel struct {
	ID                    int        `json:"novel_id"`
	Title                 string     `json:"title"`
	Captions              *string    `json:"captions"`
	Introduction          *string    `json:"introduction"`
	CategoryIDs           []int      `json:"category_ids,omitempty"`
	Categories            []Category `json:"categories"`
	CoverImage            *string    `json:"cover_image"`
	Status                string     `json:"status"`
	IsPublished           bool       `json:"is_published,omitempty"`
	IsCompleted           bool       `json:"is_completed,omitempty"`
	IsBanned              bool       `json:"is_banned"`
	BanReason             string     `json:"ban_reason"`
	AuthorID              int        `json:"author_id"`
	AuthorName            string     `json:"author_name"`
	PenName               string     `json:"pen_name"`
	Views                 int        `json:"views"`
	ReadingStatus         string     `json:"reading_status,omitempty"`
	ChapterCount          int        `json:"chapter_count,omitempty"`
	SceneCount            int        `json:"scene_count,omitempty"`
	LikeCount             int        `json:"like_count,omitempty"`
	BookshelfCount        int        `json:"bookshelf_count,omitempty"`
	CurrentSceneID        int        `json:"current_scene_id,omitempty"`
	VisitedCount          int        `json:"visited_count,omitempty"`
	EndingCount           int        `json:"ending_count,omitempty"`
	TotalScenes           int        `json:"total_scenes,omitempty"`
	LastReadSceneID       int        `json:"last_read_scene_id,omitempty"`
	LastReadSceneTitle    string     `json:"last_read_scene_title,omitempty"`
	LastReadChapterNumber int        `json:"last_read_chapter_number,omitempty"`
	LastReadChapterTitle  string     `json:"last_read_chapter_title,omitempty"`
	LastReadSceneNumber   int        `json:"last_read_scene_number,omitempty"`
	LastReadSceneName     string     `json:"last_read_scene_name,omitempty"`
	LastChoiceText        string     `json:"last_choice_text,omitempty"`
	LastReadAt            time.Time  `json:"last_read_at,omitempty"`
	IsLiked               bool       `json:"is_liked,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}
