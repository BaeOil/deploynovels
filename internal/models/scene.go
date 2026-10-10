package models

import "time"

// ✅ ใช้กับ database
type Scene struct {
	SceneID           int       `json:"scene_id"`
	ChapterID         int       `json:"chapter_id"`
	NovelID           int       `json:"novel_id"`
	Title             string    `json:"title"`
	Content           string    `json:"content"`
	ImageURL          string    `json:"image_url"`
	Type              string    `json:"type"`
	Status            string    `json:"status,omitempty"`
	EndingTitle       *string   `json:"ending_title,omitempty"`
	EndingType        *string   `json:"ending_type,omitempty"`
	EndingDescription *string   `json:"ending_description,omitempty"`
	Choices           []Choice  `json:"choices,omitempty"`
	CreatedAt         time.Time `json:"created_at,omitempty"`
	UpdatedAt         time.Time `json:"updated_at,omitempty"`
	NovelTitle        string    `json:"novel_title,omitempty"`
	ChapterTitle      string    `json:"chapter_title,omitempty"`
	ChapterEpisode    int       `json:"chapter_episode,omitempty"`
}

type SceneResponse = Scene

type EndingScene struct {
	SceneID           int        `json:"scene_id"`
	Title             string     `json:"title"`
	Type              string     `json:"type"`
	EndingTitle       *string    `json:"ending_title,omitempty"`
	EndingType        *string    `json:"ending_type,omitempty"`
	EndingDescription *string    `json:"ending_description,omitempty"`
	IsUnlocked        bool       `json:"is_unlocked"`
	UnlockedAt        *time.Time `json:"unlocked_at,omitempty"`
}

type StoryTreeResponse struct {
	NovelTitle     string      `json:"novel_title"`      // 🎯 ส่งชื่อเรื่องนิยายมาในก้อนนี้ด้วย
	CurrentSceneID int         `json:"current_scene_id"` // ไอดีฉากปัจจุบันที่ผู้เล่นอ่านค้างอยู่
	Stats          TreeStats   `json:"stats"`
	Nodes          []SceneNode `json:"nodes"`
	Edges          []SceneEdge `json:"edges"`
}

type TreeStats struct {
	VisitedScenes     int `json:"visited_scenes"`      // จำนวนโหนดที่ is_unlocked = true
	TotalScenes       int `json:"total_scenes"`        // นับจำนวนโหนดทั้งหมดของนิยายเรื่องนี้
	DiscoveredChoices int `json:"discovered_choices"`  // จำนวนเส้นเชื่อม (Edges) ที่ผู้เล่นเคยเดินผ่าน
	TotalChoicePoints int `json:"total_choice_points"` // นับจำนวนเส้นเชื่อมทั้งหมดที่มีในเรื่อง
	UnlockedEndings   int `json:"unlocked_endings"`    // จำนวนโหนดประเภท ending ที่เป็น true
	TotalEndings      int `json:"total_endings"`       // จำนวนโหนดประเภท ending ทั้งหมด
}

type SceneNode struct {
	ID                   int      `json:"id"`
	Label                string   `json:"label"`
	Title                string   `json:"title"`   // 🎯 ชื่อตอน/หัวข้อฉากจริงๆ เช่น "พบหญิงสาวปริศนา"
	Content              string   `json:"content"` // 🎯 เรื่องย่อท่อนสั้นๆ ประจำฉาก
	Type                 string   `json:"type"`
	IsUnlocked           bool     `json:"is_unlocked"`
	IsCurrent            bool     `json:"is_current"` // 🎯 ฉากปัจจุบันที่ผู้เล่นอ่านค้างอยู่
	ChapterTitle         string   `json:"chapter_title,omitempty"`
	ChapterEpisode       int      `json:"chapter_episode,omitempty"`
	SceneNumberInChapter int      `json:"scene_number_in_chapter,omitempty"`
	Excerpt              string   `json:"excerpt,omitempty"`
	Status               string   `json:"status,omitempty"`
	NodeX                *float64 `json:"node_x"`
	NodeY                *float64 `json:"node_y"`
}

type SceneEdge struct {
	ChoiceID                 int    `json:"choice_id"`
	FromID                   int    `json:"from_id"`
	ToID                     int    `json:"to_id"`
	Label                    string `json:"label"`
	FromSceneTitle           string `json:"from_scene_title,omitempty"`
	ToSceneTitle             string `json:"to_scene_title,omitempty"`
	FromChapterTitle         string `json:"from_chapter_title,omitempty"`
	ToChapterTitle           string `json:"to_chapter_title,omitempty"`
	FromChapterEpisode       int    `json:"from_chapter_episode,omitempty"`
	ToChapterEpisode         int    `json:"to_chapter_episode,omitempty"`
	FromSceneNumberInChapter int    `json:"from_scene_number_in_chapter,omitempty"`
	ToSceneNumberInChapter   int    `json:"to_scene_number_in_chapter,omitempty"`
}
