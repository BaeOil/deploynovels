package models

type Choice struct {
	ChoiceID    int    `json:"choice_id"`
	FromSceneID int    `json:"from_scene_id,omitempty"`
	Label       string `json:"label"`
	ToSceneID   int    `json:"to_scene_id"`
	IsPublished *bool  `json:"is_published,omitempty"`
}

type ChoiceDiff struct {
	CreatedCount int `json:"created_count"`
	UpdatedCount int `json:"updated_count"`
	DeletedCount int `json:"deleted_count"`
}
