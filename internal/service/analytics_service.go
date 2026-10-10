package service

import (
	"errors"

	"novel-be/internal/models"
	"novel-be/internal/repository"
)

// ErrNovelNotFound คืนเมื่อ novelID ไม่มีในระบบ
var ErrNovelNotFound = errors.New("novel not found")

// ErrSceneNotFound คืนเมื่อ sceneID ไม่มีในระบบหรือไม่อยู่ใน novelID ที่ระบุ
var ErrSceneNotFound = errors.New("scene not found in novel")

// AnalyticsService รับผิดชอบ:
//  1. รับ raw stats จาก AnalyticsRepository
//  2. Guard divide-by-zero: ถ้า unique_readers = 0 → percentage ทุกค่า = 0
//  3. Fill completion_rate = completed_readers / unique_readers × 100
//  4. Fill ending.Percentage เทียบกับ unique_readers
//  5. ส่ง drop-off ต่อจาก repo โดยไม่แก้ไข (approximate metric)
//
// ไม่ทำ: ดึง DB โดยตรง, คำนวณ SQL, แก้ business rule ของ drop-off
type AnalyticsService interface {
	GetNovelOverview(novelID int) (*models.NovelOverviewStats, error)
	GetSceneAnalytics(novelID, sceneID int) (*models.SceneAnalyticsStats, error)
	GetSceneChoiceAnalytics(novelID, sceneID int) (*models.SceneChoiceAnalyticsStats, error)
	GetAllScenesAnalytics(novelID int) ([]models.AllScenesAnalyticsStats, error)
	GetEdgeAnalytics(novelID int) ([]models.EdgeAnalyticsStats, error)
}

type analyticsService struct {
	repo repository.AnalyticsRepository
}

// NewAnalyticsService สร้าง AnalyticsService ที่ใช้ AnalyticsRepository
func NewAnalyticsService(repo repository.AnalyticsRepository) AnalyticsService {
	return &analyticsService{repo: repo}
}

// GetNovelOverview คืน analytics สรุปภาพรวมนิยาย
// ─── Responsibilities ───────────────────────────────────────────────────────
//   - เรียก repo.GetNovelOverview เพื่อรับ raw stats
//   - ตรวจสอบ stats != nil (novel ต้องมีอยู่จริง)
//   - Fill completion_rate และ ending percentage ถ้ายังเป็น 0
//     (repo คำนวณให้แล้ว — service ตรวจซ้ำเพื่อป้องกัน regression)
//   - ส่ง drop-off ต่อจาก repo ไม่แก้ไข
func (s *analyticsService) GetNovelOverview(novelID int) (*models.NovelOverviewStats, error) {
	stats, err := s.repo.GetNovelOverview(novelID)
	if err != nil {
		return nil, err
	}
	if stats == nil {
		return nil, ErrNovelNotFound
	}

	// ── completion_rate ──────────────────────────────────────────────────────
	// คำนวณใหม่เสมอเพื่อให้ service เป็น single source of truth
	// ป้องกัน divide-by-zero: ถ้า unique_readers = 0 → rate = 0
	stats.CompletionRate = calcPercentage(stats.CompletedReaders, stats.UniqueReaders)

	// ── ending_stats: percentage ─────────────────────────────────────────────
	// คำนวณ percentage ของแต่ละประเภทเทียบกับ unique_readers (ไม่ใช่ completed)
	// เหตุผล: แสดงให้ writer เห็นว่าผู้อ่านทั้งหมดกี่ % ปลดล็อก ending แต่ละแบบ
	for i := range stats.EndingStats {
		stats.EndingStats[i].Percentage = calcPercentage(stats.EndingStats[i].Count, stats.UniqueReaders)
	}

	// ── drop-off ─────────────────────────────────────────────────────────────
	// ส่งต่อจาก repo ตรงๆ — ไม่แก้ไข
	// drop_off_rate คำนวณด้วย SQL ROUND ใน repository แล้ว
	// (approximate metric — ไม่มี exact session exit event)

	return stats, nil
}

// GetSceneAnalytics คืน analytics สรุปเฉพาะฉาก
// ─── Responsibilities ───────────────────────────────────────────────────────
//   - เรียก repo.GetSceneAnalytics เพื่อรับ raw stats
//   - ตรวจสอบ stats != nil (scene ต้องสังกัด novelID) -> หากเป็น nil คืน ErrSceneNotFound
//   - Normalize repeat_visit_count = visit_count - unique_readers (กั้นไม่ให้ติดลบ)
//   - Calculate/normalize percentages ของ previous_scenes และ next_scenes
func (s *analyticsService) GetSceneAnalytics(novelID, sceneID int) (*models.SceneAnalyticsStats, error) {
	stats, err := s.repo.GetSceneAnalytics(novelID, sceneID)
	if err != nil {
		return nil, err
	}
	if stats == nil {
		return nil, ErrSceneNotFound
	}

	// ── repeat_visit_count ────────────────────────────────────────────────────
	if stats.VisitCount > stats.UniqueReaders {
		stats.RepeatVisitCount = stats.VisitCount - stats.UniqueReaders
	} else {
		stats.RepeatVisitCount = 0
	}

	// ── previous_scenes percentages ──────────────────────────────────────────
	var totalPrev int64
	for _, ps := range stats.PreviousScenes {
		totalPrev += ps.TransitionCount
	}
	for i := range stats.PreviousScenes {
		stats.PreviousScenes[i].Percentage = calcPercentage(stats.PreviousScenes[i].TransitionCount, totalPrev)
	}

	// ── next_scenes percentages ──────────────────────────────────────────────
	var totalNext int64
	for _, ns := range stats.NextScenes {
		totalNext += ns.TransitionCount
	}
	for i := range stats.NextScenes {
		stats.NextScenes[i].Percentage = calcPercentage(stats.NextScenes[i].TransitionCount, totalNext)
	}

	return stats, nil
}

// GetSceneChoiceAnalytics คืน analytics สรุปสถิติทางเลือก (Choices) ของฉาก
// ─── Responsibilities ───────────────────────────────────────────────────────
//   - เรียก repo.GetSceneChoiceAnalytics เพื่อรับ raw stats
//   - ตรวจสอบ stats != nil (scene ต้องสังกัด novelID) -> หากเป็น nil คืน ErrSceneNotFound
//   - คำนวณ percentage ของแต่ละ choice: selection_count / totalSelections * 100
//   - หา top_choice (choice ที่มี selection_count สูงสุด):
//   - ถ้าทุก choice selection_count == 0 -> top_choice = nil
//   - ถ้ามี selection_count เสมอกัน -> ใช้ choice_id ที่น้อยกว่าเป็น deterministic tie-break
func (s *analyticsService) GetSceneChoiceAnalytics(novelID, sceneID int) (*models.SceneChoiceAnalyticsStats, error) {
	stats, err := s.repo.GetSceneChoiceAnalytics(novelID, sceneID)
	if err != nil {
		return nil, err
	}
	if stats == nil {
		return nil, ErrSceneNotFound
	}

	var totalSelections int64
	for _, c := range stats.Choices {
		totalSelections += c.SelectionCount
	}

	var topIdx = -1
	var maxCount int64 = 0

	for i := range stats.Choices {
		stats.Choices[i].Percentage = calcPercentage(stats.Choices[i].SelectionCount, totalSelections)

		count := stats.Choices[i].SelectionCount
		if count > 0 {
			if topIdx == -1 || count > maxCount || (count == maxCount && stats.Choices[i].ChoiceID < stats.Choices[topIdx].ChoiceID) {
				maxCount = count
				topIdx = i
			}
		}
	}

	if topIdx != -1 {
		top := stats.Choices[topIdx]
		stats.TopChoice = &models.TopChoiceStat{
			ChoiceID:       top.ChoiceID,
			Label:          top.Label,
			SelectionCount: top.SelectionCount,
			Percentage:     top.Percentage,
		}
	} else {
		stats.TopChoice = nil
	}

	return stats, nil
}

func (s *analyticsService) GetAllScenesAnalytics(novelID int) ([]models.AllScenesAnalyticsStats, error) {
	return s.repo.GetAllScenesAnalytics(novelID)
}

func (s *analyticsService) GetEdgeAnalytics(novelID int) ([]models.EdgeAnalyticsStats, error) {
	return s.repo.GetEdgeAnalytics(novelID)
}

// calcPercentage คำนวณ part/total × 100 ปัดทศนิยม 2 ตำแหน่ง
// คืน 0 เมื่อ total = 0 เพื่อป้องกัน divide-by-zero
func calcPercentage(part, total int64) float64 {
	if total == 0 {
		return 0
	}
	return roundTo2(float64(part) * 100.0 / float64(total))
}

// roundTo2 ปัดทศนิยม 2 ตำแหน่ง
func roundTo2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}
