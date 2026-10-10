package repository

import (
	"database/sql"
	"errors"
	"novel-be/internal/models"
)

var (
	ErrChapterHasStartScene      = errors.New("ไม่สามารถลบตอนที่มีฉากเริ่มต้นของเรื่องได้ กรุณาเปลี่ยนฉากเริ่มต้นก่อน")
	ErrChapterHasIncomingChoices = errors.New("ไม่สามารถลบตอนนี้ได้ เนื่องจากมีทางเลือกจากตอนอื่นเชื่อมมายังฉากภายในตอนนี้ กรุณาแก้ไขหรือลบทางเลือกที่เกี่ยวข้องก่อน")
)

func GetChaptersByNovelID(db *sql.DB, novelID int) ([]models.Chapter, error) {
	rows, err := db.Query(`
        SELECT chapter_id, novel_id, episode, title, status, created_at, updated_at
        FROM chapters
        WHERE novel_id = $1
        ORDER BY episode ASC
    `, novelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	chapters := []models.Chapter{}
	for rows.Next() {
		var c models.Chapter
		err := rows.Scan(&c.ChapterID, &c.NovelID, &c.Episode, &c.Title, &c.Status, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, err
		}

		scenes, err := GetScenesByChapterID(db, c.ChapterID)
		if err != nil {
			return nil, err
		}
		c.Scenes = scenes

		chapters = append(chapters, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return chapters, nil
}

func GetChapterByID(db *sql.DB, id int) (*models.Chapter, error) {
	row := db.QueryRow(`
        SELECT chapter_id, novel_id, episode, title, status, created_at, updated_at
        FROM chapters
        WHERE chapter_id = $1
    `, id)

	var c models.Chapter
	err := row.Scan(&c.ChapterID, &c.NovelID, &c.Episode, &c.Title, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func CreateChapter(db *sql.DB, chapter models.Chapter) (int, error) {
	var id int
	err := db.QueryRow(`
        INSERT INTO chapters (novel_id, episode, title, status)
        VALUES ($1, $2, $3, $4)
        RETURNING chapter_id
    `, chapter.NovelID, chapter.Episode, chapter.Title, chapter.Status).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func UpdateChapter(db *sql.DB, chapter models.Chapter) error {
	_, err := db.Exec(`
        UPDATE chapters
        SET title = $1,
            status = $2,
            updated_at = NOW()
        WHERE chapter_id = $3
    `, chapter.Title, chapter.Status, chapter.ChapterID)
	return err
}

func DeleteChapter(db *sql.DB, chapterID int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var novelID, deletedEpisode int
	if err = tx.QueryRow(`
		SELECT novel_id, episode
		FROM chapters
		WHERE chapter_id = $1
	`, chapterID).Scan(&novelID, &deletedEpisode); err != nil {
		return err
	}

	// 1. ตรวจสอบว่า Chapter นี้มี Scene ที่เป็น Start Scene ของ Novel หรือไม่
	//    Chapter 1 (episode = 1) ที่มี Start Scene ยังลบได้ เพราะเป็นการลบทั้ง Chapter
	//    ไม่ใช่การลบ Start Scene โดยตรง (ข้อยกเว้น FR-STRUCT-08) โดยฉากแรกของ Chapter
	//    ถัดไปจะถูกเลื่อนเป็น Start Scene อัตโนมัติ (FR-STRUCT-09)
	//    Chapter อื่น (episode != 1) ที่มี Start Scene ยังห้ามลบ ตาม FR-STRUCT-08
	var hasStartScene bool
	if err = tx.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM scenes
			WHERE chapter_id = $1 AND type = 'start'
		)
	`, chapterID).Scan(&hasStartScene); err != nil {
		return err
	}
	if hasStartScene && deletedEpisode != 1 {
		return ErrChapterHasStartScene
	}

	// 2. ตรวจสอบว่า Scene ภายใน Chapter นี้มี Choice จาก Scene ภายนอก Chapter ชี้เข้ามาหรือไม่
	var hasExternalIncomingChoices bool
	if err = tx.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM choices c
			JOIN scenes from_s ON c.from_scene_id = from_s.scene_id
			JOIN scenes to_s ON c.to_scene_id = to_s.scene_id
			WHERE from_s.chapter_id != $1 AND to_s.chapter_id = $1
		)
	`, chapterID).Scan(&hasExternalIncomingChoices); err != nil {
		return err
	}
	if hasExternalIncomingChoices {
		return ErrChapterHasIncomingChoices
	}

	if _, err = tx.Exec(`DELETE FROM chapters WHERE chapter_id = $1`, chapterID); err != nil {
		return err
	}

	if _, err = tx.Exec(`
		UPDATE chapters
		SET episode = episode + 100000, updated_at = NOW()
		WHERE novel_id = $1 AND episode > $2
	`, novelID, deletedEpisode); err != nil {
		return err
	}
	if _, err = tx.Exec(`
		UPDATE chapters
		SET episode = episode - 100001, updated_at = NOW()
		WHERE novel_id = $1 AND episode > $2
	`, novelID, deletedEpisode+100000); err != nil {
		return err
	}

	// 4. หากลบ Chapter 1 ให้เลื่อนฉากแรกของ Chapter ถัดไป (ที่กลายเป็น Chapter 1 ใหม่)
	//    ขึ้นมาเป็น Start Scene อัตโนมัติ เฉพาะเมื่อนิยายไม่มี Start Scene หลงเหลือ
	//    หาก Chapter ที่เลื่อนขึ้นมาไม่มีฉาก จะไม่มีแถวถูกอัปเดต (ไม่สร้างฉากหรือ Start Scene ปลอม)
	//    การลบ Chapter ที่ไม่ใช่ Chapter 1 ไม่สร้าง Start Scene ใหม่ (business rule)
	if deletedEpisode == 1 {
		if _, err = tx.Exec(`
			UPDATE scenes
			SET type = 'start', updated_at = NOW()
			WHERE scene_id = (
				SELECT s.scene_id
				FROM scenes s
				JOIN chapters c ON c.chapter_id = s.chapter_id
				WHERE c.novel_id = $1
				  AND NOT EXISTS (
					  SELECT 1 FROM scenes existing_start
					  WHERE existing_start.novel_id = $1 AND existing_start.type = 'start'
				  )
				ORDER BY c.episode ASC, s.scene_id ASC
				LIMIT 1
			)
		`, novelID); err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

// ReorderChapters updates the episode/order of existing chapters according to
// the provided slice of chapter IDs (ordered from top to bottom). This will
// perform UPDATEs per ID and will not delete or recreate rows.
func ReorderChapters(db *sql.DB, orderedIDs []int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	for idx, id := range orderedIDs {
		// set episode starting from 1
		if _, err = tx.Exec(`UPDATE chapters SET episode = $1, updated_at = NOW() WHERE chapter_id = $2`, idx+1, id); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}
