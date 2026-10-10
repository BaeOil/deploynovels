package models

// NovelOverviewStats คือ Analytics สรุปภาพรวมของนิยาย 1 เรื่อง
// สำหรับ Writer Dashboard – Novel Overview
type NovelOverviewStats struct {
	TotalViews       int64          `json:"total_views"`
	UniqueReaders    int64          `json:"unique_readers"`
	CompletedReaders int64          `json:"completed_readers"`
	CompletionRate   float64        `json:"completion_rate"` // หน่วย %
	EndingStats      []EndingStat   `json:"ending_stats"`
	TopDropOffScenes []DropOffScene `json:"top_drop_off_scenes"`
}

// EndingStat สรุปสถิติการปลดล็อก Ending แต่ละประเภท
type EndingStat struct {
	SceneID     int     `json:"scene_id,omitempty"`
	EndingTitle string  `json:"ending_title,omitempty"`
	EndingType  string  `json:"ending_type"`
	Count       int64   `json:"count"`      // จำนวน distinct users ที่ปลดล็อก
	Percentage  float64 `json:"percentage"` // สัดส่วนเทียบกับ unique_readers
}

// DropOffScene แสดงฉากที่ผู้อ่านมักหยุดอ่าน (Approximate)
//
// Drop-off คำนวณจาก:
//
//	visited_users - continued_users
//	─────────────────────────────── × 100
//	       visited_users
//
// หมายเหตุ: นี่คือ approximation เนื่องจากยังไม่มี scene exit event
// ไม่ควรแปลผลว่าเป็น exact session drop-off
type DropOffScene struct {
	SceneID       int     `json:"scene_id"`
	Title         string  `json:"title"`
	VisitCount    int64   `json:"visit_count"`    // ผลรวม visit_count ทุก user
	UniqueReaders int64   `json:"unique_readers"` // distinct users ที่เข้าฉากนี้
	DropOffRate   float64 `json:"drop_off_rate"`  // หน่วย %
}

// SceneAnalyticsStats คือ Analytics สรุปของฉาก 1 ฉากในนิยาย
type SceneAnalyticsStats struct {
	SceneID          int                 `json:"scene_id"`
	Title            string              `json:"title"`
	VisitCount       int64               `json:"visit_count"`        // SUM(user_scene_history.visit_count)
	UniqueReaders    int64               `json:"unique_readers"`     // COUNT(DISTINCT user_scene_history.user_id)
	RepeatVisitCount int64               `json:"repeat_visit_count"` // visit_count - unique_readers (>= 0)
	DropOffRate      float64             `json:"drop_off_rate"`      // Approximate drop-off rate (%)
	PreviousScenes   []PreviousSceneStat `json:"previous_scenes"`
	NextScenes       []NextSceneStat     `json:"next_scenes"`
}

// PreviousSceneStat แสดงสถิติฉากก่อนหน้าที่มีทางเลือกนำเข้าสู่ฉากนี้
type PreviousSceneStat struct {
	SceneID         int     `json:"scene_id"`
	Title           string  `json:"title"`
	TransitionCount int64   `json:"transition_count"` // จำนวนครั้งที่เลือก choice นำเข้าสู่ฉากนี้จาก user_choice_history
	Percentage      float64 `json:"percentage"`       // สัดส่วนเทียบกับ transition_count รวมของทุก previous scene
}

// NextSceneStat แสดงสถิติทางเลือกถัดไปออกจากฉากนี้
type NextSceneStat struct {
	SceneID         int     `json:"scene_id"`
	Title           string  `json:"title"`
	ChoiceLabel     string  `json:"choice_label"`
	TransitionCount int64   `json:"transition_count"` // จำนวนครั้งที่เลือก choice นี้จาก user_choice_history
	Percentage      float64 `json:"percentage"`       // สัดส่วนเทียบกับ transition_count รวมของทุก next scene
}

// SceneChoiceAnalyticsStats สรุปสถิติทางเลือก (Choices) ของฉาก 1 ฉาก
type SceneChoiceAnalyticsStats struct {
	SceneID   int            `json:"scene_id"`
	Choices   []ChoiceStat   `json:"choices"`
	TopChoice *TopChoiceStat `json:"top_choice"`
}

// ChoiceStat แสดงสถิติของแต่ละ Choice ที่ออกจากฉาก
type ChoiceStat struct {
	ChoiceID         int     `json:"choice_id"`
	Label            string  `json:"label"`
	ToSceneID        *int    `json:"to_scene_id"`
	TargetSceneTitle string  `json:"target_scene_title"`
	SelectionCount   int64   `json:"selection_count"` // COUNT(*) จาก user_choice_history
	Percentage       float64 `json:"percentage"`      // % สัดส่วนเทียบกับยอดเลือก choices ทั้งหมดในฉากนี้
}

// TopChoiceStat แสดง Choice ที่มีผู้อ่านเลือกมากที่สุด
type TopChoiceStat struct {
	ChoiceID       int     `json:"choice_id"`
	Label          string  `json:"label"`
	SelectionCount int64   `json:"selection_count"`
	Percentage     float64 `json:"percentage"`
}

// AllScenesAnalyticsStats คือ Analytics สรุปของทุกฉากในนิยาย
type AllScenesAnalyticsStats struct {
	SceneID       int     `json:"scene_id"`
	Title         string  `json:"title"`
	VisitCount    int64   `json:"visit_count"`
	UniqueReaders int64   `json:"unique_readers"`
	DropOffRate   float64 `json:"drop_off_rate"`
}

type EdgeAnalyticsStats struct {
	FromSceneID    int     `json:"from_scene_id"`
	ToSceneID      int     `json:"to_scene_id"`
	ChoiceID       int     `json:"choice_id"`
	ChoiceLabel    string  `json:"choice_label"`
	TargetTitle    string  `json:"target_scene_title"`
	SelectionCount int64   `json:"selection_count"`
	Percentage     float64 `json:"percentage"`
}
