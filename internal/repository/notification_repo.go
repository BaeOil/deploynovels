package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"novel-be/internal/models"
)

func CreateNotification(db *sql.DB, notification models.Notification) (int, error) {
	var id int
	var referenceID any
	var referenceType any
	if notification.ReferenceID != nil {
		referenceID = *notification.ReferenceID
	}
	if notification.ReferenceType != nil {
		referenceType = *notification.ReferenceType
	}
	err := db.QueryRow(`
		INSERT INTO notifications (
			user_id, type, message, reference_id, reference_type, is_read, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW())
		RETURNING notification_id
	`, notification.UserID, notification.Type, notification.Message, referenceID, referenceType, notification.IsRead).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func GetNotificationsByUserID(db *sql.DB, userID int, page, limit int) ([]models.Notification, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	rows, err := db.Query(`
		SELECT notification_id, user_id, type, message, reference_id, reference_type, is_read, created_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC, notification_id DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.Notification
	for rows.Next() {
		var n models.Notification
		var refID sql.NullInt64
		var refType sql.NullString
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Message, &refID, &refType, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, err
		}
		if refID.Valid {
			v := int(refID.Int64)
			n.ReferenceID = &v
		}
		if refType.Valid {
			n.ReferenceType = &refType.String
		}
		populateNotificationDisplayFields(db, &n)
		items = append(items, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func populateNotificationDisplayFields(db *sql.DB, n *models.Notification) {
	n.ActorColor = "#E91E8C"
	n.ActorAvatar = ""
	n.ActorName = "StoryVerse"

	if n.Type == "system" {
		n.ActorName = "StoryVerse"
		// Older notification rows predate the title column and only retain the
		// message. Restore the known system titles when reading those rows.
		if strings.Contains(n.Message, "ตอนนี้คุณสามารถสร้างนิยาย") {
			title := "คุณได้รับการอนุมัติเป็นนักเขียนแล้ว"
			n.Title = &title
		} else if strings.Contains(n.Message, "ไม่ได้รับการอนุมัติ") {
			title := "คำขอสมัครนักเขียนถูกปฏิเสธ"
			n.Title = &title
		}
		return
	}

	if n.ReferenceID != nil && n.ReferenceType != nil {
		if *n.ReferenceType == "novel" {
			title, err := ResolveNovelTitle(db, *n.ReferenceID)
			if err == nil {
				n.Title = &title
				n.ReferenceTitle = title
			}
			cover, err := ResolveNovelCover(db, *n.ReferenceID)
			if err == nil && cover != "" {
				n.CoverImage = &cover
			}
			if n.Type == "novel_update" {
				authorUserID, err := ResolveNovelAuthorUserID(db, *n.ReferenceID)
				if err == nil && authorUserID != 0 {
					actorName, err := ResolveDisplayNameByUserID(db, authorUserID)
					if err == nil {
						n.ActorName = actorName
					}
				}
			}
		}
	}

	if n.Type == "follower" {
		actorName := extractActorNameFromMessage(n.Message, n.Type)
		if actorName != "" {
			n.ActorName = actorName
			if actorID, err := ResolveUserIDByUsername(db, actorName); err == nil && actorID != 0 {
				n.ActorID = actorID
			}
			title := actorName
			n.Title = &title
		}
	}

	if n.Type == "like" || n.Type == "comment" {
		actorName := extractActorNameFromMessage(n.Message, n.Type)
		if actorName != "" {
			n.ActorName = actorName
			if actorID, err := ResolveUserIDByUsername(db, actorName); err == nil && actorID != 0 {
				n.ActorID = actorID
			}
		}
	}
}

func extractActorNameFromMessage(message, typ string) string {
	switch typ {
	case "follower":
		if idx := strings.Index(message, " "); idx > 0 {
			return message[:idx]
		}
		return message
	case "like", "comment":
		if parts := strings.SplitN(message, "|", 2); len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
		return message
	default:
		if idx := strings.Index(message, " "); idx > 0 {
			return message[:idx]
		}
		return message
	}
}

func ResolveUserIDByUsername(db *sql.DB, username string) (int, error) {
	var userID int
	err := db.QueryRow(`
		SELECT user_id
		FROM users
		WHERE username = $1
		LIMIT 1
	`, strings.TrimSpace(username)).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return userID, nil
}

func ResolveUserAvatarByUserID(db *sql.DB, userID int) (string, error) {
	var avatar sql.NullString
	// เลือก writer row ที่ approved ก่อน (status='approved' = 0), ไม่งั้นเรียงตาม applied_at ล่าสุด
	// เพื่อให้ได้ผลลัพธ์ที่แน่นอนเมื่อ user มีหลาย writer rows (rejected history)
	err := db.QueryRow(`
		SELECT COALESCE(NULLIF(w.avatar_url, ''), NULLIF(u.pic_profile, ''))
		FROM users u
		LEFT JOIN writers w ON w.writer_id = (
			SELECT w2.writer_id
			FROM writers w2
			WHERE w2.user_id = u.user_id
			ORDER BY (CASE WHEN w2.status = 'approved' THEN 0 ELSE 1 END), w2.applied_at DESC
			LIMIT 1
		)
		WHERE u.user_id = $1
	`, userID).Scan(&avatar)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	if avatar.Valid {
		return avatar.String, nil
	}
	return "", nil
}

func GetUnreadNotificationCount(db *sql.DB, userID int) (int, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(1)
		FROM notifications
		WHERE user_id = $1 AND is_read = FALSE
	`, userID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func MarkNotificationRead(db *sql.DB, notificationID, userID int) error {
	res, err := db.Exec(`
		UPDATE notifications
		SET is_read = TRUE
		WHERE notification_id = $1 AND user_id = $2
	`, notificationID, userID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func MarkAllNotificationsRead(db *sql.DB, userID int) error {
	_, err := db.Exec(`
		UPDATE notifications
		SET is_read = TRUE
		WHERE user_id = $1
	`, userID)
	return err
}

func DeleteNotification(db *sql.DB, notificationID, userID int) error {
	res, err := db.Exec(`
		DELETE FROM notifications
		WHERE notification_id = $1 AND user_id = $2
	`, notificationID, userID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func DeleteAllNotifications(db *sql.DB, userID int) error {
	_, err := db.Exec(`
		DELETE FROM notifications
		WHERE user_id = $1
	`, userID)
	return err
}

func DeleteNotificationsByUserReference(db *sql.DB, userID int, typ string, referenceID int, referenceType string) error {
	_, err := db.Exec(`
		DELETE FROM notifications
		WHERE user_id = $1 AND type = $2 AND reference_id = $3 AND reference_type = $4
	`, userID, typ, referenceID, referenceType)
	return err
}

func HasLikeNotificationForActor(db *sql.DB, userID, referenceID int, actorName string) (bool, error) {
	var count int
	if err := db.QueryRow(`
		SELECT COUNT(1)
		FROM notifications
		WHERE user_id = $1
		  AND type = 'like'
		  AND reference_id = $2
		  AND reference_type = 'novel'
		  AND message LIKE $3
	`, userID, referenceID, actorName+"|%").Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func DeleteLikeNotificationByActor(db *sql.DB, userID, referenceID int, actorName string) error {
	_, err := db.Exec(`
		DELETE FROM notifications
		WHERE user_id = $1
		  AND type = 'like'
		  AND reference_id = $2
		  AND reference_type = 'novel'
		  AND message LIKE $3
	`, userID, referenceID, actorName+"|%")
	return err
}

func HasRecentSceneUpdateNotification(db *sql.DB, userID, novelID int, since time.Time) (bool, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(1)
		FROM notifications
		WHERE user_id = $1
		  AND type = 'novel_update'
		  AND reference_id = $2
		  AND reference_type = 'novel'
		  AND created_at >= $3
		  AND message LIKE '% มีอัปเดตใหม่ใน %'
	`, userID, novelID, since).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func HasExistingFollowNotification(db *sql.DB, recipientUserID, targetWriterID int) (bool, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(1)
		FROM notifications
		WHERE user_id = $1
		  AND type = 'follower'
		  AND reference_id = $2
		  AND reference_type = 'user'
	`, recipientUserID, targetWriterID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func GetBookshelfUserIDsByNovelID(db *sql.DB, novelID int) ([]int, error) {
	rows, err := db.Query(`
		SELECT DISTINCT user_id
		FROM bookshelves
		WHERE novel_id = $1
	`, novelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

func GetFollowersByNovelAuthor(db *sql.DB, authorID int) ([]int, error) {
	rows, err := db.Query(`
		SELECT follower_id
		FROM follows
		WHERE following_id = $1
	`, authorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

// จงใจไม่ filter soft-deleted rows เพราะ query นี้ใช้ข้อเท็จจริงการอ่านจริง
// ไม่ใช่การแสดงผล UI
func GetReadersReachedChapterScene(db *sql.DB, novelID int, chapterID int) ([]int, error) {
	rows, err := db.Query(`
		SELECT rp.user_id
		FROM reading_progress rp
		JOIN scenes s ON s.scene_id = rp.current_scene_id
		WHERE rp.novel_id = $1 AND s.chapter_id = $2
	`, novelID, chapterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

func GetChapterSceneCount(db *sql.DB, chapterID int) (int, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(1)
		FROM scenes
		WHERE chapter_id = $1
	`, chapterID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func GetChapterLatestSceneID(db *sql.DB, chapterID int) (int, error) {
	var sceneID int
	err := db.QueryRow(`
		SELECT scene_id
		FROM scenes
		WHERE chapter_id = $1
		ORDER BY scene_id DESC
		LIMIT 1
	`, chapterID).Scan(&sceneID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return sceneID, nil
}

// จงใจไม่ filter soft-deleted rows เพราะ query นี้ใช้ข้อเท็จจริงการอ่านจริง
// ไม่ใช่การแสดงผล UI
func GetReaderLastSceneInChapter(db *sql.DB, userID, novelID, chapterID int) (int, error) {
	var sceneID int
	err := db.QueryRow(`
		SELECT rp.current_scene_id
		FROM reading_progress rp
		WHERE rp.user_id = $1 AND rp.novel_id = $2
	`, userID, novelID).Scan(&sceneID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}

	var currentChapterID int
	err = db.QueryRow(`
		SELECT chapter_id
		FROM scenes
		WHERE scene_id = $1
	`, sceneID).Scan(&currentChapterID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	if currentChapterID != chapterID {
		return 0, nil
	}
	return sceneID, nil
}

func ResolveWriterUserID(db *sql.DB, writerID int) (int, error) {
	var userID int
	err := db.QueryRow(`
		SELECT user_id
		FROM writers
		WHERE writer_id = $1
	`, writerID).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return userID, nil
}

func ResolveNovelAuthorUserID(db *sql.DB, novelID int) (int, error) {
	var userID int
	err := db.QueryRow(`
		SELECT w.user_id
		FROM novels n
		JOIN writers w ON n.author_id = w.writer_id
		WHERE n.novel_id = $1
		LIMIT 1
	`, novelID).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return userID, nil
}

func ResolveNovelAuthorWriterID(db *sql.DB, novelID int) (int, error) {
	var writerID int
	err := db.QueryRow(`
		SELECT author_id
		FROM novels
		WHERE novel_id = $1
		LIMIT 1
	`, novelID).Scan(&writerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return writerID, nil
}

func ResolveDisplayNameByUserID(db *sql.DB, userID int) (string, error) {
	var displayName string
	err := db.QueryRow(`
		SELECT COALESCE(NULLIF(u.username, ''), 'ผู้ใช้')
		FROM users u
		WHERE u.user_id = $1
		LIMIT 1
	`, userID).Scan(&displayName)
	if err != nil {
		if err == sql.ErrNoRows {
			return "ผู้ใช้", nil
		}
		return "", err
	}
	if strings.TrimSpace(displayName) == "" {
		return "ผู้ใช้", nil
	}
	return displayName, nil
}

func ResolveNovelTitle(db *sql.DB, novelID int) (string, error) {
	var title string
	err := db.QueryRow(`
		SELECT title
		FROM novels
		WHERE novel_id = $1
	`, novelID).Scan(&title)
	if err != nil {
		return "", err
	}
	return title, nil
}

func ResolveNovelPublishedState(db *sql.DB, novelID int) (bool, string, error) {
	var isPublished bool
	var status string
	err := db.QueryRow(`
		SELECT is_published, status
		FROM novels
		WHERE novel_id = $1
	`, novelID).Scan(&isPublished, &status)
	if err != nil {
		return false, "", err
	}
	return isPublished, status, nil
}

func ResolveSceneTitle(db *sql.DB, sceneID int) (string, error) {
	var title string
	err := db.QueryRow(`
		SELECT title
		FROM scenes
		WHERE scene_id = $1
	`, sceneID).Scan(&title)
	if err != nil {
		return "", err
	}
	return title, nil
}

func ResolveSceneStatus(db *sql.DB, sceneID int) (string, error) {
	var status string
	err := db.QueryRow(`
		SELECT status
		FROM scenes
		WHERE scene_id = $1
	`, sceneID).Scan(&status)
	if err != nil {
		return "", err
	}
	return status, nil
}

func ResolveNovelCover(db *sql.DB, novelID int) (string, error) {
	var cover sql.NullString
	err := db.QueryRow(`
		SELECT cover_image
		FROM novels
		WHERE novel_id = $1
	`, novelID).Scan(&cover)
	if err != nil {
		return "", err
	}
	if cover.Valid {
		return cover.String, nil
	}
	return "", nil
}

func ResolveWriterName(db *sql.DB, writerID int) (string, error) {
	var displayName string
	err := db.QueryRow(`
		SELECT pen_name
		FROM writers
		WHERE pen_name IS NOT NULL AND pen_name <> ''
		LIMIT 1
	`).Scan(&displayName)
	if err != nil {
		if err == sql.ErrNoRows {
			return "StoryVerse", nil
		}
		return "", err
	}
	if strings.TrimSpace(displayName) != "" {
		return displayName, nil
	}
	return "StoryVerse", nil
}

func ResolveWriterDisplayName(db *sql.DB, writerID int) (string, error) {
	var name string
	err := db.QueryRow(`
		SELECT pen_name
		FROM writers
		WHERE writer_id = $1
	`, writerID).Scan(&name)
	if err != nil {
		if err == sql.ErrNoRows {
			return "StoryVerse", nil
		}
		return "", err
	}
	return name, nil
}

func BuildNotification(db *sql.DB, userID int, typ, title, message string, cover *string, referenceID *int, referenceType *string) (models.Notification, error) {
	actorName, err := ResolveWriterDisplayName(db, userID)
	if err != nil {
		actorName = "StoryVerse"
	}
	return models.Notification{
		UserID:        userID,
		Type:          typ,
		Title:         &title,
		Message:       message,
		CoverImage:    cover,
		ReferenceID:   referenceID,
		ReferenceType: referenceType,
		IsRead:        false,
		ActorName:     actorName,
		ActorColor:    "#E91E8C",
	}, nil
}

func CreateNotificationForUser(db *sql.DB, userID int, typ, title, message string, cover *string, referenceID *int, referenceType *string) (int, error) {
	notif, err := BuildNotification(db, userID, typ, title, message, cover, referenceID, referenceType)
	if err != nil {
		return 0, fmt.Errorf("build notification: %w", err)
	}
	return CreateNotification(db, notif)
}

func GetNotificationSettings(db *sql.DB, userID int) (*models.NotificationSettings, error) {
	query := `
		SELECT
			user_id, novel_update_enabled, follower_enabled,
			like_enabled, comment_enabled, system_enabled
		FROM user_notification_settings
		WHERE user_id = $1
	`
	var s models.NotificationSettings
	err := db.QueryRow(query, userID).Scan(
		&s.UserID, &s.NovelUpdateEnabled, &s.FollowerEnabled,
		&s.LikeEnabled, &s.CommentEnabled, &s.SystemEnabled,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return &models.NotificationSettings{
				UserID:             uint(userID),
				NovelUpdateEnabled: true,
				FollowerEnabled:    true,
				LikeEnabled:        true,
				CommentEnabled:     true,
				SystemEnabled:      true,
			}, nil
		}
		return nil, err
	}
	return &s, nil
}

func UpsertNotificationSettings(db *sql.DB, userID int, settings models.NotificationSettings) error {
	query := `
		INSERT INTO user_notification_settings (
			user_id, novel_update_enabled, follower_enabled,
			like_enabled, comment_enabled, system_enabled
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id)
		DO UPDATE SET
			novel_update_enabled = EXCLUDED.novel_update_enabled,
			follower_enabled = EXCLUDED.follower_enabled,
			like_enabled = EXCLUDED.like_enabled,
			comment_enabled = EXCLUDED.comment_enabled,
			system_enabled = EXCLUDED.system_enabled
	`
	_, err := db.Exec(query, userID, settings.NovelUpdateEnabled, settings.FollowerEnabled,
		settings.LikeEnabled, settings.CommentEnabled, settings.SystemEnabled)
	return err
}
