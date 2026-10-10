package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"novel-be/internal/dto"
	"novel-be/internal/models"
)

var ErrUsernameTaken = errors.New("username already in use")
var ErrUserRoleNotReader = errors.New("user role must be reader to restore writer access")
var ErrUserHasNoPriorRevokedWriterHistory = errors.New("user has no prior revoked writer history")

type sqlAuthRepository struct {
	db *sql.DB
}

func NewAuthRepository(db *sql.DB) AuthRepository {
	return &sqlAuthRepository{db: db}
}

// CreateUser ยิง SQL บันทึกยูสเซอร์ใหม่ลงฐานข้อมูลจริง
func (r *sqlAuthRepository) CreateUser(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (username, email, password_hash, role, pic_profile, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, COALESCE(NULLIF($6, ''), 'active'), NOW(), NOW())
		RETURNING user_id, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query, user.Username, user.Email, user.PasswordHash, user.Role, user.PicProfile, user.Status).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	return err
}

// GetByUsername ดึงข้อมูลจากฐานข้อมูลมาตรวจสอบตอนล็อกอิน
func (r *sqlAuthRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `SELECT user_id, username, email, password_hash, pic_profile, role, status, suspended_reason, suspended_at, last_action_by_admin_id, created_at, updated_at FROM users WHERE username = $1`

	var user models.User
	var suspendedReason sql.NullString
	var suspendedAt sql.NullTime
	var lastActionByAdminID sql.NullInt64
	err := r.db.QueryRowContext(ctx, query, username).
		Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.PicProfile, &user.Role, &user.Status, &suspendedReason, &suspendedAt, &lastActionByAdminID, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if suspendedReason.Valid {
		user.SuspendedReason = suspendedReason.String
	}
	if suspendedAt.Valid {
		user.SuspendedAt = &suspendedAt.Time
	}
	if lastActionByAdminID.Valid {
		id := uint(lastActionByAdminID.Int64)
		user.LastActionByAdminID = &id
	}
	return &user, nil
}

func (r *sqlAuthRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `SELECT user_id, username, email, password_hash, pic_profile, role, status, suspended_reason, suspended_at, last_action_by_admin_id, created_at, updated_at FROM users WHERE email = $1`

	var user models.User
	var suspendedReason sql.NullString
	var suspendedAt sql.NullTime
	var lastActionByAdminID sql.NullInt64
	err := r.db.QueryRowContext(ctx, query, email).
		Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.PicProfile, &user.Role, &user.Status, &suspendedReason, &suspendedAt, &lastActionByAdminID, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if suspendedReason.Valid {
		user.SuspendedReason = suspendedReason.String
	}
	if suspendedAt.Valid {
		user.SuspendedAt = &suspendedAt.Time
	}
	if lastActionByAdminID.Valid {
		id := uint(lastActionByAdminID.Int64)
		user.LastActionByAdminID = &id
	}
	return &user, nil
}

// GetByID ดึงข้อมูลผู้ใช้จากไอดีของเขา (ใช้ตอนต้องการเรียกดูข้อมูลผู้ใช้ปัจจุบัน)
func (r *sqlAuthRepository) GetByID(ctx context.Context, userID uint) (*models.User, error) {
	query := `SELECT user_id, username, email, password_hash, pic_profile, role, status, suspended_reason, suspended_at, last_action_by_admin_id, created_at, updated_at FROM users WHERE user_id = $1`

	var user models.User
	var suspendedReason sql.NullString
	var suspendedAt sql.NullTime
	var lastActionByAdminID sql.NullInt64
	err := r.db.QueryRowContext(ctx, query, userID).
		Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.PicProfile, &user.Role, &user.Status, &suspendedReason, &suspendedAt, &lastActionByAdminID, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if suspendedReason.Valid {
		user.SuspendedReason = suspendedReason.String
	}
	if suspendedAt.Valid {
		user.SuspendedAt = &suspendedAt.Time
	}
	if lastActionByAdminID.Valid {
		id := uint(lastActionByAdminID.Int64)
		user.LastActionByAdminID = &id
	}
	return &user, nil
}

func (r *sqlAuthRepository) ListUsers(ctx context.Context, role, status, search string, page, limit int) ([]dto.AdminUserListItemDTO, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	query := `
		SELECT u.user_id, u.username, u.email, u.pic_profile, u.role, COALESCE(u.status, 'active'), u.created_at,
			u.suspended_reason, u.suspended_at,
			w.status AS writer_application_status,
			EXISTS(
				SELECT 1 FROM writers w3 WHERE w3.user_id = u.user_id AND w3.status = 'revoked'
			) AS had_writer_history,
			COALESCE((SELECT json_agg(c.name) FROM writer_categories wc JOIN categories c ON c.category_id = wc.category_id WHERE wc.writer_id = w.writer_id), '[]'::json) AS genres_json,
			w.name_lastname, w.pen_name, w.bio, w.contact_info, w.writer_id
		FROM users u
		LEFT JOIN writers w ON w.writer_id = (SELECT w2.writer_id FROM writers w2 WHERE w2.user_id = u.user_id ORDER BY w2.applied_at DESC LIMIT 1)
		WHERE 1=1`
	args := []interface{}{}
	argCount := 1

	if role != "" {
		query += fmt.Sprintf(" AND u.role = $%d", argCount)
		args = append(args, strings.ToLower(role))
		argCount++
	}
	if status != "" {
		query += fmt.Sprintf(" AND u.status = $%d", argCount)
		args = append(args, strings.ToLower(status))
		argCount++
	}
	if search != "" {
		query += fmt.Sprintf(" AND (LOWER(u.username) LIKE $%d OR LOWER(u.email) LIKE $%d)", argCount, argCount)
		args = append(args, "%"+strings.ToLower(search)+"%")
		argCount++
	}

	query += fmt.Sprintf(" ORDER BY u.created_at DESC LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []dto.AdminUserListItemDTO
	for rows.Next() {
		var item dto.AdminUserListItemDTO
		var picProfile sql.NullString
		var createdAt time.Time
		var suspendedReason sql.NullString
		var suspendedAt sql.NullTime
		var writerAppStatus sql.NullString
		var hadWriterHistory bool
		var nameLastname, penName, bio sql.NullString
		var contactInfo sql.NullString
		var genresJSON []byte
		var writerID sql.NullInt64
		err := rows.Scan(&item.ID, &item.Username, &item.Email, &picProfile, &item.Role, &item.Status, &createdAt, &suspendedReason, &suspendedAt, &writerAppStatus, &hadWriterHistory, &genresJSON, &nameLastname, &penName, &bio, &contactInfo, &writerID)
		if err != nil {
			return nil, err
		}
		item.CreatedAt = createdAt.Format(time.RFC3339)
		if picProfile.Valid {
			item.PicProfile = &picProfile.String
		}
		if suspendedReason.Valid {
			item.SuspendedReason = &suspendedReason.String
		}
		if suspendedAt.Valid {
			suspendedAtValue := suspendedAt.Time
			item.SuspendedAt = &suspendedAtValue
		}
		if writerAppStatus.Valid {
			status := writerAppStatus.String
			item.WriterApplicationStatus = &status
		} else {
			item.WriterApplicationStatus = nil
		}
		item.HasWriterHistory = hadWriterHistory
		if writerID.Valid {
			wID := uint(writerID.Int64)
			item.WriterID = &wID
		}
		if nameLastname.Valid || penName.Valid || bio.Valid || contactInfo.Valid {
			genres := []string{}
			if len(genresJSON) > 0 && string(genresJSON) != "null" {
				_ = json.Unmarshal(genresJSON, &genres)
			}
			primaryContact := ""
			secondaryContact := ""
			if contactInfo.Valid && strings.TrimSpace(contactInfo.String) != "" {
				var contactMap map[string]interface{}
				if err := json.Unmarshal([]byte(contactInfo.String), &contactMap); err == nil {
					if val, ok := contactMap["contact_required"]; ok {
						if str, ok := val.(string); ok {
							primaryContact = str
						}
					} else if val, ok := contactMap["primary_contact"]; ok {
						if str, ok := val.(string); ok {
							primaryContact = str
						}
					}
					if val, ok := contactMap["contact_optional"]; ok {
						if str, ok := val.(string); ok {
							secondaryContact = str
						}
					} else if val, ok := contactMap["secondary_contact"]; ok {
						if str, ok := val.(string); ok {
							secondaryContact = str
						}
					}
				}
			}
			var wID uint
			if writerID.Valid {
				wID = uint(writerID.Int64)
			}
			item.WriterDetails = &dto.AdminWriterDetailsDTO{
				ID:               wID,
				WriterID:         wID,
				NameLastname:     nameLastname.String,
				PenName:          penName.String,
				Bio:              bio.String,
				Genres:           genres,
				PrimaryContact:   primaryContact,
				SecondaryContact: secondaryContact,
			}
		}
		results = append(results, item)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func (r *sqlAuthRepository) GetUserForAdmin(ctx context.Context, userID uint) (*dto.AdminUserDetailDTO, error) {
	query := `
		SELECT u.user_id, u.username, u.email, u.pic_profile, u.role, COALESCE(u.status, 'active'), u.created_at,
			u.suspended_reason, u.suspended_at,
			w.status AS writer_application_status,
			EXISTS(
				SELECT 1 FROM writers w3 WHERE w3.user_id = u.user_id AND w3.status = 'revoked'
			) AS had_writer_history,
			COALESCE((SELECT json_agg(c.name) FROM writer_categories wc JOIN categories c ON c.category_id = wc.category_id WHERE wc.writer_id = w.writer_id), '[]'::json) AS genres_json,
			w.name_lastname, w.pen_name, w.bio, w.contact_info, w.writer_id
		FROM users u
		LEFT JOIN writers w ON w.writer_id = (SELECT w2.writer_id FROM writers w2 WHERE w2.user_id = u.user_id ORDER BY w2.applied_at DESC LIMIT 1)
		WHERE u.user_id = $1`

	var item dto.AdminUserDetailDTO
	var picProfile sql.NullString
	var createdAt time.Time
	var suspendedReason sql.NullString
	var suspendedAt sql.NullTime
	var writerAppStatus sql.NullString
	var hadWriterHistory bool
	var nameLastname, penName, bio sql.NullString
	var contactInfo sql.NullString
	var genresJSON []byte
	var writerID sql.NullInt64
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&item.ID, &item.Username, &item.Email, &picProfile, &item.Role, &item.Status, &createdAt, &suspendedReason, &suspendedAt, &writerAppStatus, &hadWriterHistory, &genresJSON, &nameLastname, &penName, &bio, &contactInfo, &writerID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	item.CreatedAt = createdAt.Format(time.RFC3339)
	if picProfile.Valid {
		item.PicProfile = &picProfile.String
	}
	if suspendedReason.Valid {
		item.SuspendedReason = &suspendedReason.String
	}
	if suspendedAt.Valid {
		suspendedAtValue := suspendedAt.Time
		item.SuspendedAt = &suspendedAtValue
	}
	if writerAppStatus.Valid {
		status := writerAppStatus.String
		item.WriterApplicationStatus = &status
	}
	item.HasWriterHistory = hadWriterHistory
	if writerID.Valid {
		wID := uint(writerID.Int64)
		item.WriterID = &wID
	}
	if nameLastname.Valid || penName.Valid || bio.Valid || contactInfo.Valid {
		genres := []string{}
		if len(genresJSON) > 0 && string(genresJSON) != "null" {
			_ = json.Unmarshal(genresJSON, &genres)
		}
		primaryContact := ""
		secondaryContact := ""
		if contactInfo.Valid && strings.TrimSpace(contactInfo.String) != "" {
			var contactMap map[string]interface{}
			if err := json.Unmarshal([]byte(contactInfo.String), &contactMap); err == nil {
				if val, ok := contactMap["contact_required"]; ok {
					if str, ok := val.(string); ok {
						primaryContact = str
					}
				} else if val, ok := contactMap["primary_contact"]; ok {
					if str, ok := val.(string); ok {
						primaryContact = str
					}
				}
				if val, ok := contactMap["contact_optional"]; ok {
					if str, ok := val.(string); ok {
						secondaryContact = str
					}
				} else if val, ok := contactMap["secondary_contact"]; ok {
					if str, ok := val.(string); ok {
						secondaryContact = str
					}
				}
			}
		}
		var wID uint
		if writerID.Valid {
			wID = uint(writerID.Int64)
		}
		item.WriterDetails = &dto.AdminWriterDetailsDTO{
			ID:               wID,
			WriterID:         wID,
			NameLastname:     nameLastname.String,
			PenName:          penName.String,
			Bio:              bio.String,
			Genres:           genres,
			PrimaryContact:   primaryContact,
			SecondaryContact: secondaryContact,
		}
	}
	return &item, nil
}

func (r *sqlAuthRepository) UpdateUserStatus(ctx context.Context, userID uint, status, reason string, suspendedAt *time.Time, adminID uint) error {
	query := `
		UPDATE users
		SET status = $1::text,
			suspended_reason = CASE WHEN $1::text = 'suspended' THEN NULLIF(TRIM($2::text), '') ELSE NULL END,
			suspended_at = CASE WHEN $1::text = 'suspended' THEN COALESCE($3::timestamptz, NOW()) ELSE NULL END,
			last_action_by_admin_id = $4,
			updated_at = NOW()
		WHERE user_id = $5`

	_, err := r.db.ExecContext(ctx, query, status, normalizeSuspendReason(reason), suspendAtValue(suspendedAt), adminID, userID)
	return err
}

// Invariant: any user demotion from writer->reader must be centralized here.
// Do not mutate users.role or writers.status in another admin flow; otherwise
// query-time visibility and writer authorization will drift over time.
func (r *sqlAuthRepository) DemoteUserToReader(ctx context.Context, userID uint, adminID uint) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		UPDATE users
		SET role = 'reader', last_action_by_admin_id = $1, updated_at = NOW()
		WHERE user_id = $2`, adminID, userID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE writers
		SET status = 'revoked', acted_by_admin_id = $1
		WHERE user_id = $2 AND status = 'approved'`, adminID, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *sqlAuthRepository) RestoreUserWriterAccess(ctx context.Context, userID uint, adminID uint) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var role string
	if err := tx.QueryRowContext(ctx, `SELECT role FROM users WHERE user_id = $1`, userID).Scan(&role); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sql.ErrNoRows
		}
		return err
	}
	if !strings.EqualFold(role, "reader") {
		return ErrUserRoleNotReader
	}

	var revokedCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM writers WHERE user_id = $1 AND status = 'revoked'`, userID).Scan(&revokedCount); err != nil {
		return err
	}
	if revokedCount == 0 {
		return ErrUserHasNoPriorRevokedWriterHistory
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE writers
		SET status = 'approved', approved_at = NOW(), acted_by_admin_id = $1, rejected_at = NULL, rejection_reason = NULL
		WHERE user_id = $2 AND status = 'revoked'`, adminID, userID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE users
		SET role = 'writer', last_action_by_admin_id = $1, updated_at = NOW()
		WHERE user_id = $2 AND role = 'reader'`, adminID, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *sqlAuthRepository) DeleteUser(ctx context.Context, userID uint) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE user_id = $1`, userID)
	return err
}

func (r *sqlAuthRepository) HasWriterNovels(ctx context.Context, userID uint) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM writers w
			JOIN novels n ON n.author_id = w.writer_id
			WHERE w.user_id = $1
		)`
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&exists)
	return exists, err
}

func (r *sqlAuthRepository) displayRole(role string) string {
	return strings.TrimSpace(role)
}

func (r *sqlAuthRepository) displayStatus(status string) string {
	trimmed := strings.TrimSpace(status)
	if trimmed == "" {
		return "active"
	}
	return trimmed
}

func normalizeSuspendReason(reason string) string {
	return strings.TrimSpace(reason)
}

func suspendAtValue(suspendedAt *time.Time) interface{} {
	if suspendedAt == nil {
		return nil
	}
	return *suspendedAt
}

func (r *sqlAuthRepository) UpdateUsername(ctx context.Context, userID uint, username string) error {
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1 AND user_id != $2)`
	if err := r.db.QueryRowContext(ctx, checkQuery, username, userID).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return ErrUsernameTaken
	}

	query := `UPDATE users SET username = $1, updated_at = NOW() WHERE user_id = $2`
	res, err := r.db.ExecContext(ctx, query, username, userID)
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

func (r *sqlAuthRepository) UpdateEmail(ctx context.Context, userID uint, email string) error {
	query := `UPDATE users SET email = $1, updated_at = NOW() WHERE user_id = $2`
	res, err := r.db.ExecContext(ctx, query, email, userID)
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

func (r *sqlAuthRepository) UpdatePassword(ctx context.Context, userID uint, passwordHash string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE user_id = $2`
	res, err := r.db.ExecContext(ctx, query, passwordHash, userID)
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

func (r *sqlAuthRepository) UpdateProfilePicture(ctx context.Context, userID uint, picProfile string) error {
	query := `UPDATE users SET pic_profile = $1, updated_at = NOW() WHERE user_id = $2`
	res, err := r.db.ExecContext(ctx, query, picProfile, userID)
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

func (r *sqlAuthRepository) SuspendUser(ctx context.Context, userID uint, reason string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		UPDATE users
		SET status = 'suspended',
		    suspended_reason = NULLIF(TRIM($1), ''),
		    suspended_at = NOW(),
		    updated_at = NOW()
		WHERE user_id = $2`
	res, err := tx.ExecContext(ctx, query, normalizeSuspendReason(reason), userID)
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

	return tx.Commit()
}
