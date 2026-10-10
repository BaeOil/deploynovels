package service

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"novel-be/internal/dto"
	"novel-be/internal/models"
	"novel-be/internal/repository"
)

type NotificationService interface {
	CreateNotification(userID int, actorID int, typ, title, message string, cover *string, referenceID *int, referenceType *string) (int, error)
	ListNotifications(userID int, page, limit int) ([]models.Notification, error)
	GetUnreadCount(userID int) (int, error)
	MarkRead(notificationID, userID int) error
	MarkAllRead(userID int) error
	Delete(notificationID, userID int) error
	DeleteAll(userID int) error
	GetNotificationSettings(userID int) (*dto.NotificationSettingsDTO, error)
	UpdateNotificationSettings(userID int, settings dto.NotificationSettingsDTO) error
	NotifyNewNovelPublished(novelID int) error
	NotifyNovelChapterPublished(novelID, chapterID int) error
	NotifySceneUpdated(novelID, sceneID int) error
	NotifyFollow(targetWriterID, actorID int) error
	NotifyLike(novelID, actorID int) error
	NotifyComment(novelID, actorID int, commentText string) error
	DeleteLikeNotification(novelID, actorID int) error
	NotifyWriterApproved(writerID int) error
	NotifyWriterRejected(writerID int, reason string) error
}

type notificationService struct {
	db *sql.DB
}

type notificationSubscription struct {
	userID int
	ch     chan string
}

var (
	notificationSubscribersMu sync.RWMutex
	notificationSubscribers   = make(map[int][]chan string)
)

func NewNotificationService(db *sql.DB) NotificationService {
	return &notificationService{db: db}
}

func SubscribeNotifications(userID int) (<-chan string, func()) {
	ch := make(chan string, 16)
	notificationSubscribersMu.Lock()
	notificationSubscribers[userID] = append(notificationSubscribers[userID], ch)
	notificationSubscribersMu.Unlock()

	cleanup := func() {
		notificationSubscribersMu.Lock()
		defer notificationSubscribersMu.Unlock()
		channels := notificationSubscribers[userID]
		updated := channels[:0]
		for _, existing := range channels {
			if existing != ch {
				updated = append(updated, existing)
			}
		}
		notificationSubscribers[userID] = updated
		if len(notificationSubscribers[userID]) == 0 {
			delete(notificationSubscribers, userID)
		}
		close(ch)
	}

	return ch, cleanup
}

func publishNotification(userID int, payload map[string]any) {
	notificationSubscribersMu.RLock()
	channels := append([]chan string(nil), notificationSubscribers[userID]...)
	notificationSubscribersMu.RUnlock()

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	for _, ch := range channels {
		select {
		case ch <- string(data):
		default:
		}
	}
}

func (s *notificationService) CreateNotification(userID int, actorID int, typ, title, message string, cover *string, referenceID *int, referenceType *string) (int, error) {
	log.Println("CreateNotification called", "userID:", userID, "actorID:", actorID, "type:", typ, "title:", title, "message:", message, "cover:", cover, "referenceID:", referenceID, "referenceType:", referenceType)
	if userID == actorID {
		log.Println("CreateNotification skipped because userID == actorID")
		return 0, nil
	}

	settings, err := repository.GetNotificationSettings(s.db, userID)
	if err != nil {
		log.Printf("CreateNotification error fetching settings for user %d: %v", userID, err)
		return 0, err
	}

	switch typ {
	case "novel_update":
		if !settings.NovelUpdateEnabled {
			log.Printf("CreateNotification skipped for user %d: novel_update_enabled disabled", userID)
			return 0, nil
		}
	case "follower":
		if !settings.FollowerEnabled {
			log.Printf("CreateNotification skipped for user %d: follower_enabled disabled", userID)
			return 0, nil
		}
	case "like":
		if !settings.LikeEnabled {
			log.Printf("CreateNotification skipped for user %d: like_enabled disabled", userID)
			return 0, nil
		}
	case "comment":
		if !settings.CommentEnabled {
			log.Printf("CreateNotification skipped for user %d: comment_enabled disabled", userID)
			return 0, nil
		}
	case "system":
		if !settings.SystemEnabled {
			log.Printf("CreateNotification skipped for user %d: system_enabled disabled", userID)
			return 0, nil
		}
	}

	id, err := repository.CreateNotificationForUser(s.db, userID, typ, title, message, cover, referenceID, referenceType)
	if err != nil {
		log.Printf("CreateNotification error: %v", err)
		return 0, err
	}

	payload := s.buildNotificationPayload(id, actorID, typ, title, message, cover, referenceID, referenceType)
	payload["user_id"] = userID
	payload["is_read"] = false
	payload["created_at"] = time.Now().UTC().Format(time.RFC3339)
	publishNotification(userID, payload)
	return id, nil
}

func (s *notificationService) buildNotificationPayload(id, actorID int, typ, title, message string, cover *string, referenceID *int, referenceType *string) map[string]any {
	actorName := "StoryVerse"
	actorAvatar := ""
	actorColor := "#E91E8C"

	if actorID != 0 {
		if name, err := repository.ResolveDisplayNameByUserID(s.db, actorID); err == nil && strings.TrimSpace(name) != "" {
			actorName = name
		}
		if avatar, err := repository.ResolveUserAvatarByUserID(s.db, actorID); err == nil && strings.TrimSpace(avatar) != "" {
			actorAvatar = avatar
		}
	}

	payload := map[string]any{
		"id":           id,
		"actor_id":     actorID,
		"type":         typ,
		"actor_name":   actorName,
		"actor_avatar": actorAvatar,
		"actor_color":  actorColor,
		"actor": map[string]any{
			"id":     actorID,
			"name":   actorName,
			"avatar": actorAvatar,
			"color":  actorColor,
		},
		"title":   title,
		"message": message,
	}

	if cover != nil {
		payload["cover_image"] = *cover
		payload["cover"] = *cover
	}
	if referenceID != nil {
		payload["reference_id"] = *referenceID
	}
	if referenceType != nil {
		payload["reference_type"] = *referenceType
	}
	if referenceType != nil && *referenceType == "novel" && referenceID != nil {
		if refTitle, err := repository.ResolveNovelTitle(s.db, *referenceID); err == nil && refTitle != "" {
			payload["reference_title"] = refTitle
		}
	}

	return payload
}

func (s *notificationService) ListNotifications(userID int, page, limit int) ([]models.Notification, error) {
	return repository.GetNotificationsByUserID(s.db, userID, page, limit)
}

func (s *notificationService) GetUnreadCount(userID int) (int, error) {
	return repository.GetUnreadNotificationCount(s.db, userID)
}

func (s *notificationService) MarkRead(notificationID, userID int) error {
	return repository.MarkNotificationRead(s.db, notificationID, userID)
}

func (s *notificationService) MarkAllRead(userID int) error {
	return repository.MarkAllNotificationsRead(s.db, userID)
}

func (s *notificationService) Delete(notificationID, userID int) error {
	return repository.DeleteNotification(s.db, notificationID, userID)
}

func (s *notificationService) DeleteAll(userID int) error {
	return repository.DeleteAllNotifications(s.db, userID)
}

func (s *notificationService) GetNotificationSettings(userID int) (*dto.NotificationSettingsDTO, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user id")
	}
	settings, err := repository.GetNotificationSettings(s.db, userID)
	if err != nil {
		return nil, err
	}
	return &dto.NotificationSettingsDTO{
		InAppNotifications: settings.SystemEnabled,
		NovelUpdates:       settings.NovelUpdateEnabled,
		Comments:           settings.CommentEnabled,
		Likes:              settings.LikeEnabled,
		Follows:            settings.FollowerEnabled,
	}, nil
}

func (s *notificationService) UpdateNotificationSettings(userID int, settings dto.NotificationSettingsDTO) error {
	if userID <= 0 {
		return errors.New("invalid user id")
	}
	modelSettings := models.NotificationSettings{
		UserID:             uint(userID),
		NovelUpdateEnabled: settings.NovelUpdates,
		FollowerEnabled:    settings.Follows,
		LikeEnabled:        settings.Likes,
		CommentEnabled:     settings.Comments,
		SystemEnabled:      settings.InAppNotifications,
	}
	return repository.UpsertNotificationSettings(s.db, userID, modelSettings)
}

func (s *notificationService) NotifyNewNovelPublished(novelID int) error {
	authorUserID, err := repository.ResolveNovelAuthorUserID(s.db, novelID)
	if err != nil {
		return fmt.Errorf("resolve novel author user: %w", err)
	}
	if authorUserID == 0 {
		return nil
	}

	authorWriterID, err := repository.ResolveNovelAuthorWriterID(s.db, novelID)
	if err != nil {
		return fmt.Errorf("resolve novel author writer: %w", err)
	}
	if authorWriterID == 0 {
		return nil
	}

	novelTitle, err := repository.ResolveNovelTitle(s.db, novelID)
	if err != nil {
		return fmt.Errorf("resolve novel title: %w", err)
	}
	cover, err := repository.ResolveNovelCover(s.db, novelID)
	if err != nil {
		cover = ""
	}

	followers, err := repository.GetFollowersByNovelAuthor(s.db, authorWriterID)
	if err != nil {
		return fmt.Errorf("resolve followers: %w", err)
	}
	for _, followerID := range followers {
		message := fmt.Sprintf("นิยาย %s เปิดตัวใหม่แล้ว คุณสามารถอ่านได้ทันที", novelTitle)
		_, err = s.CreateNotification(followerID, authorUserID, "novel_update", novelTitle, message, &cover, &novelID, strPtr("novel"))
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *notificationService) NotifyNovelChapterPublished(novelID, chapterID int) error {
	log.Println("NotifyNovelChapterPublished called")
	authorUserID, err := repository.ResolveNovelAuthorUserID(s.db, novelID)
	if err != nil {
		return fmt.Errorf("resolve novel author user: %w", err)
	}
	if authorUserID == 0 {
		return nil
	}

	novelTitle, err := repository.ResolveNovelTitle(s.db, novelID)
	if err != nil {
		return fmt.Errorf("resolve novel title: %w", err)
	}
	cover, err := repository.ResolveNovelCover(s.db, novelID)
	if err != nil {
		cover = ""
	}
	readerIDs, err := repository.GetBookshelfUserIDsByNovelID(s.db, novelID)
	if err != nil {
		return fmt.Errorf("resolve bookshelf readers: %w", err)
	}
	chapterCount, err := repository.GetChapterSceneCount(s.db, chapterID)
	if err != nil {
		return fmt.Errorf("resolve chapter scenes: %w", err)
	}
	for _, readerID := range readerIDs {
		message := fmt.Sprintf("%s\nเพิ่มตอนที่ %d แล้ว", novelTitle, chapterCount)
		_, err = s.CreateNotification(readerID, authorUserID, "novel_update", novelTitle, message, &cover, &novelID, strPtr("novel"))
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *notificationService) NotifySceneUpdated(novelID, sceneID int) error {
	authorUserID, err := repository.ResolveNovelAuthorUserID(s.db, novelID)
	if err != nil {
		return fmt.Errorf("resolve novel author user: %w", err)
	}
	if authorUserID == 0 {
		return nil
	}

	novelPublished, novelStatus, err := repository.ResolveNovelPublishedState(s.db, novelID)
	if err != nil {
		return fmt.Errorf("resolve novel visibility: %w", err)
	}
	if !novelPublished || strings.EqualFold(novelStatus, "draft") || strings.EqualFold(novelStatus, "completed-draft") {
		return nil
	}

	sceneStatus, err := repository.ResolveSceneStatus(s.db, sceneID)
	if err != nil {
		return fmt.Errorf("resolve scene status: %w", err)
	}
	if !strings.EqualFold(sceneStatus, "published") {
		return nil
	}

	novelTitle, err := repository.ResolveNovelTitle(s.db, novelID)
	if err != nil {
		return fmt.Errorf("resolve novel title: %w", err)
	}
	cover, err := repository.ResolveNovelCover(s.db, novelID)
	if err != nil {
		cover = ""
	}
	sceneTitle, err := repository.ResolveSceneTitle(s.db, sceneID)
	if err != nil {
		sceneTitle = "ฉาก"
	}
	readerIDs, err := repository.GetBookshelfUserIDsByNovelID(s.db, novelID)
	if err != nil {
		return fmt.Errorf("resolve bookshelf readers: %w", err)
	}
	since := time.Now().Add(-1 * time.Hour)
	for _, readerID := range readerIDs {
		hasRecent, err := repository.HasRecentSceneUpdateNotification(s.db, readerID, novelID, since)
		if err != nil {
			return fmt.Errorf("check recent scene update notification: %w", err)
		}
		if hasRecent {
			continue
		}
		message := fmt.Sprintf("%s มีอัปเดตใหม่ใน %s", novelTitle, sceneTitle)
		_, err = s.CreateNotification(readerID, authorUserID, "novel_update", novelTitle, message, &cover, &novelID, strPtr("novel"))
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *notificationService) NotifyFollow(targetWriterID, actorID int) error {
	recipientUserID, err := repository.ResolveWriterUserID(s.db, targetWriterID)
	if err != nil {
		return err
	}
	if recipientUserID == 0 {
		return nil
	}

	hasExisting, err := repository.HasExistingFollowNotification(s.db, recipientUserID, targetWriterID)
	if err != nil {
		return err
	}
	if hasExisting {
		return nil
	}

	actorName, err := repository.ResolveDisplayNameByUserID(s.db, actorID)
	if err != nil {
		actorName = "คนอ่าน"
	}
	_, err = s.CreateNotification(recipientUserID, actorID, "follower", actorName, fmt.Sprintf("%s เริ่มติดตามคุณแล้ว", actorName), nil, &targetWriterID, strPtr("user"))
	return err
}

func (s *notificationService) NotifyLike(novelID, actorID int) error {
	authorUserID, err := repository.ResolveNovelAuthorUserID(s.db, novelID)
	if err != nil {
		return err
	}
	if authorUserID == 0 {
		return nil
	}
	novelTitle, err := repository.ResolveNovelTitle(s.db, novelID)
	if err != nil {
		return err
	}
	actorName, err := repository.ResolveDisplayNameByUserID(s.db, actorID)
	if err != nil {
		actorName = "ผู้ใช้"
	}

	hasExisting, err := repository.HasLikeNotificationForActor(s.db, authorUserID, novelID, actorName)
	if err != nil {
		return err
	}
	if hasExisting {
		return nil
	}

	messagePayload := fmt.Sprintf("%s|", actorName)
	_, err = s.CreateNotification(authorUserID, actorID, "like", novelTitle, messagePayload, nil, &novelID, strPtr("novel"))
	return err
}

func (s *notificationService) NotifyComment(novelID, actorID int, commentText string) error {
	authorUserID, err := repository.ResolveNovelAuthorUserID(s.db, novelID)
	if err != nil {
		return err
	}
	if authorUserID == 0 {
		return nil
	}
	novelTitle, err := repository.ResolveNovelTitle(s.db, novelID)
	if err != nil {
		return err
	}
	actorName, err := repository.ResolveDisplayNameByUserID(s.db, actorID)
	if err != nil {
		actorName = "ผู้ใช้"
	}
	// store actor name and the actual comment separated by a delimiter so list API can extract actor and body
	messagePayload := fmt.Sprintf("%s|%s", actorName, commentText)
	_, err = s.CreateNotification(authorUserID, actorID, "comment", novelTitle, messagePayload, nil, &novelID, strPtr("novel"))
	return err
}

func (s *notificationService) DeleteLikeNotification(novelID, actorID int) error {
	authorUserID, err := repository.ResolveNovelAuthorUserID(s.db, novelID)
	if err != nil {
		return err
	}
	if authorUserID == 0 {
		return nil
	}
	actorName, err := repository.ResolveDisplayNameByUserID(s.db, actorID)
	if err != nil {
		actorName = "ผู้ใช้"
	}
	return repository.DeleteLikeNotificationByActor(s.db, authorUserID, novelID, actorName)
}

func (s *notificationService) NotifyWriterApproved(writerID int) error {
	recipientUserID, err := repository.ResolveWriterUserID(s.db, writerID)
	if err != nil {
		return err
	}
	if recipientUserID == 0 {
		return nil
	}

	_, err = s.CreateNotification(recipientUserID, 0, "system", "คุณได้รับการอนุมัติเป็นนักเขียนแล้ว", "ตอนนี้คุณสามารถสร้างนิยายและเผยแพร่ผลงานได้แล้ว", nil, nil, nil)
	return err
}

func (s *notificationService) NotifyWriterRejected(writerID int, reason string) error {
	recipientUserID, err := repository.ResolveWriterUserID(s.db, writerID)
	if err != nil {
		return err
	}
	if recipientUserID == 0 {
		return nil
	}

	message := "คำขอสมัครเป็นนักเขียนของคุณไม่ได้รับการอนุมัติในครั้งนี้ ลองแก้ไขข้อมูลแล้วสมัครใหม่ได้อีกครั้งนะคะ"
	if strings.TrimSpace(reason) != "" {
		message = fmt.Sprintf("คำขอสมัครเป็นนักเขียนของคุณไม่ได้รับการอนุมัติ เหตุผล: %s", strings.TrimSpace(reason))
	}

	_, err = s.CreateNotification(recipientUserID, 0, "system", "คำขอสมัครนักเขียนถูกปฏิเสธ", message, nil, nil, nil)
	return err
}

func strPtr(v string) *string {
	return &v
}
