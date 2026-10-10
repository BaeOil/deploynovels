package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"novel-be/internal/dto"
	"novel-be/internal/models" // 👈 มั่นใจว่ามีอิมพอร์ตโมเดลตัวจริงเข้ามาใช้งานแล้ว
)

type sqlWriterRepository struct {
	db *sql.DB
}

// Sentinel errors ให้ handler ชั้นบนแยกแยะได้ว่าควรตอบ 404/409 ไม่ใช่ 500 เหมารวม
// และให้แอดมินเห็นสาเหตุจริงแทนข้อความกำกวมแบบเดิม
var (
	ErrWriterApplicationNotFound         = errors.New("ไม่พบคำขอนักเขียนนี้ในระบบ")
	ErrNotLatestWriterApplication        = errors.New("คำขอนี้ไม่ใช่คำขอล่าสุดของผู้ใช้แล้ว (มีการยื่นคำขอใหม่ทับไปแล้ว) กรุณารีเฟรชรายการ")
	ErrWriterApplicationAlreadyProcessed = errors.New("คำขอนี้ถูกดำเนินการไปแล้ว หรือไม่อยู่ในสถานะรอตรวจสอบ")
)

func NewWriterRepository(db *sql.DB) WriterRepository {
	return &sqlWriterRepository{db: db}
}

// 🟢 ปรับปรุงฟังก์ชันนี้ให้ตรงตามสัญญา (want GetWriterByID(int) (*models.Writer, error))
func (r *sqlWriterRepository) GetWriterByID(id int) (*models.Writer, error) {
	query := `
SELECT
	w.writer_id,
	w.user_id,
	COALESCE(u.username, '') AS username,
	w.name_lastname,
	w.pen_name,
	w.bio,
	w.email_writer,
	w.contact_info,
	w.avatar_url,
	COALESCE(
		(
			SELECT COUNT(*)
			FROM likes l
			JOIN novels n ON n.novel_id = l.novel_id
			WHERE n.author_id = w.writer_id AND n.is_published = TRUE
		),
		0
	) AS total_like_count,
	COALESCE(
		(
			SELECT SUM(n.views)
			FROM novels n
			WHERE n.author_id = w.writer_id AND n.is_published = TRUE
		),
		0
	) AS total_view_count,
	COALESCE(
		(
			SELECT COUNT(*)
			FROM follows f
			WHERE f.following_id = w.writer_id
		),
		0
	) AS follower_count,
	COALESCE(
		(
			SELECT COUNT(b.user_id)
			FROM bookshelves b
			JOIN novels n ON n.novel_id = b.novel_id
			WHERE n.author_id = w.writer_id AND n.is_published = TRUE
		),
		0
	) AS total_bookshelf_count,
	COALESCE(
		(
			SELECT COUNT(*)
			FROM novels n
			WHERE n.author_id = w.writer_id AND n.is_published = TRUE
		),
		0
	) AS novel_count,
	COALESCE(
		(
			SELECT json_agg(json_build_object('category_id', c.category_id, 'name', c.name))
			FROM writer_categories wc
			JOIN categories c ON c.category_id = wc.category_id
			WHERE wc.writer_id = w.writer_id
		)::text,
		'[]'
	) AS categories_json,
	w.applied_at,
	w.approved_at
FROM writers w
LEFT JOIN users u ON u.user_id = w.user_id
WHERE w.writer_id = $1`

	var w models.Writer
	var categoriesJSON string
	var appliedAt, approvedAt sql.NullTime
	err := r.db.QueryRow(query, id).Scan(
		&w.WriterID,
		&w.UserID,
		&w.Username,
		&w.NameLastname,
		&w.PenName,
		&w.Bio,
		&w.EmailWriter,
		&w.ContactInfo,
		&w.AvatarURL,
		&w.TotalLikeCount,
		&w.TotalViewCount,
		&w.FollowerCount,
		&w.TotalBookshelfCount,
		&w.NovelCount,
		&categoriesJSON,
		&appliedAt,
		&approvedAt,
	)
	if err != nil {
		return nil, err
	}

	if appliedAt.Valid {
		w.AppliedAt = &appliedAt.Time
	}
	if approvedAt.Valid {
		w.ApprovedAt = &approvedAt.Time
	}

	if len(categoriesJSON) > 0 && categoriesJSON != "[]" {
		_ = json.Unmarshal([]byte(categoriesJSON), &w.Categories)
	}

	return &w, nil
}

func (r *sqlWriterRepository) GetWriterByUserID(userID int) (*models.Writer, error) {
	query := `SELECT writer_id, user_id, pen_name, bio, email_writer, contact_info, avatar_url, status FROM writers WHERE user_id = $1 AND status = 'approved' LIMIT 1`

	var w models.Writer
	err := r.db.QueryRow(query, userID).Scan(&w.WriterID, &w.UserID, &w.PenName, &w.Bio, &w.EmailWriter, &w.ContactInfo, &w.AvatarURL, &w.Status)
	if err != nil {
		return nil, err
	}

	return &w, nil
}

func (r *sqlWriterRepository) GetLatestWriterApplicationByUserID(userID int) (*models.Writer, error) {
	query := `
		SELECT 
			w.writer_id, 
			w.user_id, 
			COALESCE(w.name_lastname, '') AS name_lastname, 
			w.pen_name, 
			COALESCE(w.bio, '') AS bio, 
			COALESCE(w.email_writer, '') AS email_writer, 
			COALESCE(w.contact_info::text, '{}') AS contact_info, 
			COALESCE(w.avatar_url, '') AS avatar_url, 
			w.status, 
			w.rejected_at, 
			w.rejection_reason,
			COALESCE(
				(SELECT json_agg(wc.category_id)
				 FROM writer_categories wc
				 WHERE wc.writer_id = w.writer_id),
				'[]'::json
			)::text AS category_ids_json
		FROM writers w 
		WHERE w.user_id = $1 
		ORDER BY w.applied_at DESC 
		LIMIT 1`

	var w models.Writer
	var bioStr, emailStr, contactInfoStr, categoryIDsJSON string
	var rejectedAt sql.NullTime
	var rejectionReason sql.NullString

	err := r.db.QueryRow(query, userID).Scan(
		&w.WriterID,
		&w.UserID,
		&w.NameLastname,
		&w.PenName,
		&bioStr,
		&emailStr,
		&contactInfoStr,
		&w.AvatarURL,
		&w.Status,
		&rejectedAt,
		&rejectionReason,
		&categoryIDsJSON,
	)
	if err != nil {
		return nil, err
	}

	w.Bio = &bioStr
	w.EmailWriter = &emailStr
	w.ContactInfo = contactInfoStr

	if rejectedAt.Valid {
		w.RejectedAt = &rejectedAt.Time
	}
	if rejectionReason.Valid {
		w.RejectionReason = &rejectionReason.String
	}

	if len(categoryIDsJSON) > 0 && categoryIDsJSON != "[]" && categoryIDsJSON != "null" {
		_ = json.Unmarshal([]byte(categoryIDsJSON), &w.CategoryIDs)
	}

	return &w, nil
}

func (r *sqlWriterRepository) GetUserRoleByUserID(userID int) (string, error) {
	query := `SELECT role FROM users WHERE user_id = $1`
	var role string
	err := r.db.QueryRow(query, userID).Scan(&role)
	if err != nil {
		return "", err
	}
	return role, nil
}

// ✍️ 1. ส่งคำขอเข้าตาราง writers (เริ่มต้นสถานะ 'pending')
func (r *sqlWriterRepository) Apply(ctx context.Context, userID uint, req dto.WriterApplyRequest, contactJSON string) error {
	// 1. เปิด Transaction เพราะเราจะบันทึกมากกว่า 1 ตารางพร้อมกัน
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 2. บันทึกข้อมูลใบสมัครเข้าตาราง writers ก่อน และเอา writer_id กลับออกมา
	var writerID int
	queryWriter := `
		INSERT INTO writers (user_id, name_lastname, pen_name, bio, email_writer, contact_info, avatar_url, status, applied_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending', NOW())
		RETURNING writer_id
	`
	err = tx.QueryRowContext(ctx, queryWriter, userID, req.NameLastname, req.PenName, req.Bio, req.EmailWriter, contactJSON, req.AvatarURL).Scan(&writerID)
	if err != nil {
		return err
	}

	// 3. วนลูปเอาหมวดหมู่ที่นักเขียนติ๊กเลือก บันทึกลงตาราง writer_categories
	if len(req.CategoryIDs) > 0 {
		queryCategory := `INSERT INTO writer_categories (writer_id, category_id) VALUES ($1, $2)`
		for _, catID := range req.CategoryIDs {
			_, err := tx.ExecContext(ctx, queryCategory, writerID, catID)
			if err != nil {
				return err
			}
		}
	}

	// 4. บันทึกทุกอย่างพร้อมกันลงฐานข้อมูล
	return tx.Commit()
}

// 🔍 2. แอดมินดึงรายการคำขอทั้งหมด (default คืนทุกสถานะ) และเพิ่ม support status/page/limit
func (r *sqlWriterRepository) GetPendingRequests(ctx context.Context, status string, page, limit int) ([]dto.WriterRequestResponse, error) {
	status = strings.TrimSpace(status)
	page = max(page, 0)
	limit = max(limit, 0)

	query := `
		SELECT w.writer_id, w.user_id, COALESCE(u.username, '') AS username, COALESCE(u.pic_profile, '') AS pic_profile,
			COALESCE(w.name_lastname, '') AS name_lastname, COALESCE(w.pen_name, '') AS pen_name,
			COALESCE(w.bio, '') AS bio, COALESCE(w.avatar_url, '') AS avatar_url,
			COALESCE(w.email_writer, '') AS email_writer, COALESCE(w.contact_info::text, '') AS contact_info,
			COALESCE(
				(SELECT json_agg(c.name)
				 FROM writer_categories wc
				 JOIN categories c ON c.category_id = wc.category_id
				 WHERE wc.writer_id = w.writer_id),
				'[]'::json
			) AS genres_json,
			w.status, w.applied_at, w.approved_at, w.rejected_at, w.acted_by_admin_id,
			admin_u.username AS acted_by_admin_username, w.rejection_reason,
			COALESCE(
				(SELECT COUNT(*) FROM writers w2 WHERE w2.user_id = w.user_id AND w2.status = 'rejected' AND w2.applied_at < w.applied_at),
				0
			) AS previous_attempt_count,
			(
				SELECT w2.rejection_reason
				FROM writers w2
				WHERE w2.user_id = w.user_id
					AND w2.status = 'rejected'
					AND w2.applied_at < w.applied_at
				ORDER BY w2.rejected_at DESC, w2.applied_at DESC
				LIMIT 1
			) AS previous_rejection_reason,
			prev.name_lastname AS previous_name_lastname,
			prev.pen_name AS previous_pen_name,
			prev.bio AS previous_bio,
			prev.email_writer AS previous_email_writer,
			COALESCE(prev.contact_info::text, '') AS previous_contact_info,
			prev.writer_id AS previous_writer_id,
			COALESCE((SELECT json_agg(c2.name)
				FROM writer_categories wc2
				JOIN categories c2 ON c2.category_id = wc2.category_id
				WHERE wc2.writer_id = prev.writer_id), '[]'::json) AS previous_genres_json
		FROM writers w
		LEFT JOIN users u ON u.user_id = w.user_id
		LEFT JOIN users admin_u ON admin_u.user_id = w.acted_by_admin_id
		LEFT JOIN LATERAL (
			SELECT w2.*
			FROM writers w2
			WHERE w2.user_id = w.user_id
				AND w2.status = 'rejected'
				AND w2.applied_at < w.applied_at
			ORDER BY w2.rejected_at DESC NULLS LAST, w2.applied_at DESC, w2.writer_id DESC
			LIMIT 1
		) prev ON true
		WHERE 1=1`
	args := []interface{}{}

	if status != "" {
		query += " AND w.status = $1"
		args = append(args, status)
	}

	query += " ORDER BY COALESCE(w.approved_at, w.rejected_at, w.applied_at) DESC"
	if page > 0 || limit > 0 {
		if page < 1 {
			page = 1
		}
		if limit < 1 {
			limit = 20
		}
		offset := (page - 1) * limit
		query += " LIMIT $%d OFFSET $%d"
		query = fmt.Sprintf(query, len(args)+1, len(args)+2)
		args = append(args, limit, offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []dto.WriterRequestResponse
	for rows.Next() {
		var resp dto.WriterRequestResponse
		var genresJSON []byte
		var approvedAt, rejectedAt sql.NullTime
		var actedByAdminID sql.NullInt64
		var actedByAdminUsername sql.NullString
		var rejectionReason sql.NullString
		var previousRejectionReason sql.NullString
		var previousNameLastname, previousPenName, previousBio, previousEmailWriter, previousContactInfo sql.NullString
		var previousWriterID sql.NullInt64
		var previousGenresJSON []byte
		err := rows.Scan(
			&resp.WriterID,
			&resp.UserID,
			&resp.Username,
			&resp.PicProfile,
			&resp.NameLastname,
			&resp.PenName,
			&resp.Bio,
			&resp.AvatarURL,
			&resp.EmailWriter,
			&resp.ContactInfo,
			&genresJSON,
			&resp.Status,
			&resp.AppliedAt,
			&approvedAt,
			&rejectedAt,
			&actedByAdminID,
			&actedByAdminUsername,
			&rejectionReason,
			&resp.PreviousAttemptCount,
			&previousRejectionReason,
			&previousNameLastname,
			&previousPenName,
			&previousBio,
			&previousEmailWriter,
			&previousContactInfo,
			&previousWriterID,
			&previousGenresJSON,
		)
		if err != nil {
			return nil, err
		}
		if approvedAt.Valid {
			t := approvedAt.Time
			resp.ApprovedAt = &t
		}
		if rejectedAt.Valid {
			t := rejectedAt.Time
			resp.RejectedAt = &t
		}
		if actedByAdminID.Valid {
			aid := uint(actedByAdminID.Int64)
			resp.ActedByAdminID = &aid
		}
		if actedByAdminUsername.Valid {
			username := actedByAdminUsername.String
			resp.ActedByAdminUsername = &username
		}
		if rejectionReason.Valid {
			rr := rejectionReason.String
			resp.RejectionReason = &rr
		}
		if previousRejectionReason.Valid {
			prr := previousRejectionReason.String
			resp.PreviousRejectionReason = &prr
		}
		if previousWriterID.Valid {
			previousApplication := &dto.PreviousWriterApplication{
				NameLastname: previousNameLastname.String,
				PenName:      previousPenName.String,
				Bio:          previousBio.String,
				EmailWriter:  previousEmailWriter.String,
				ContactInfo:  previousContactInfo.String,
				Genres:       []string{},
			}
			if len(previousGenresJSON) > 0 && string(previousGenresJSON) != "null" {
				if err := json.Unmarshal(previousGenresJSON, &previousApplication.Genres); err != nil {
					return nil, err
				}
			}
			resp.PreviousApplication = previousApplication
		}
		if len(genresJSON) > 0 && string(genresJSON) != "null" {
			if err := json.Unmarshal(genresJSON, &resp.Genres); err != nil {
				return nil, err
			}
		}
		requests = append(requests, resp)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return requests, nil
}

// 🎯 3. แอดมินกดอนุมัติ (อัปเดตตาราง writers และปรับ role ตาราง users เป็น 'writer' พร้อมกัน)
func (r *sqlWriterRepository) ApproveWriter(ctx context.Context, writerID uint, adminID uint) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// กันเคสแอดมินถือ writer_id เก่าค้างจาก tab/cache ที่เปิดไว้นาน แล้ว user
	// ยื่นคำขอใหม่ทับไปแล้ว (เช่น หลังถูก reject แล้วแก้ไขส่งใหม่) — ต้อง approve
	// เฉพาะคำขอที่เป็น "ล่าสุด" ของ user คนนั้นเท่านั้น ไม่งั้นจะไป approve คำขอเก่าที่ตายไปแล้ว
	var isLatest bool
	checkQuery := `
		SELECT w.writer_id = (
			SELECT w2.writer_id FROM writers w2 WHERE w2.user_id = w.user_id ORDER BY w2.applied_at DESC LIMIT 1
		)
		FROM writers w WHERE w.writer_id = $1`
	err = tx.QueryRowContext(ctx, checkQuery, writerID).Scan(&isLatest)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrWriterApplicationNotFound
		}
		return err
	}
	if !isLatest {
		return ErrNotLatestWriterApplication
	}

	var userID uint
	queryUpdateWriter := `
		UPDATE writers 
		SET status = 'approved', approved_at = NOW(), acted_by_admin_id = $2, rejected_at = NULL, rejection_reason = NULL
		WHERE writer_id = $1 AND status = 'pending'
		RETURNING user_id
	`
	err = tx.QueryRowContext(ctx, queryUpdateWriter, writerID, adminID).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrWriterApplicationAlreadyProcessed
		}
		return err
	}

	queryUpdateUser := `UPDATE users SET role = 'writer', updated_at = NOW() WHERE user_id = $1`
	_, err = tx.ExecContext(ctx, queryUpdateUser, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// ❌ แอดมินกดปฏิเสธคำขอ (อัปเดตสถานะในตาราง writers เป็น 'rejected')
func (r *sqlWriterRepository) RejectWriter(ctx context.Context, writerID uint, adminID uint, rejectionReason string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var currentStatus string
	var isLatest bool
	checkQuery := `
		SELECT w.status, w.writer_id = (
			SELECT w2.writer_id FROM writers w2 WHERE w2.user_id = w.user_id ORDER BY w2.applied_at DESC LIMIT 1
		)
		FROM writers w WHERE w.writer_id = $1`
	err = tx.QueryRowContext(ctx, checkQuery, writerID).Scan(&currentStatus, &isLatest)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrWriterApplicationNotFound
		}
		return err
	}
	if !isLatest {
		return ErrNotLatestWriterApplication
	}
	if currentStatus != "pending" {
		return ErrWriterApplicationAlreadyProcessed
	}

	query := `
		UPDATE writers 
		SET status = 'rejected', rejected_at = NOW(), acted_by_admin_id = $2, rejection_reason = NULLIF(TRIM($3), '' )
		WHERE writer_id = $1 AND status = 'pending'
	`
	result, err := tx.ExecContext(ctx, query, writerID, adminID, rejectionReason)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrWriterApplicationAlreadyProcessed
	}

	return tx.Commit()
}

// ✏️ อัปเดตข้อมูลโปรไฟล์นักเขียน (pen_name, bio, avatar_url, contact_info และ writer_categories)
func (r *sqlWriterRepository) UpdateWriterProfile(ctx context.Context, writerID int, req dto.UpdateWriterProfileRequest, contactJSON string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. อัปเดตตาราง writers
	queryWriter := `
		UPDATE writers
		SET pen_name = $1, bio = $2, avatar_url = $3, contact_info = $4, email_writer = COALESCE(NULLIF($5, ''), email_writer)
		WHERE writer_id = $6
	`
	_, err = tx.ExecContext(ctx, queryWriter, req.PenName, req.Bio, req.AvatarURL, contactJSON, req.EmailWriter, writerID)
	if err != nil {
		return err
	}

	// 2. ลบหมวดหมู่เดิมและลงใหม่หากส่ง category_ids มา
	if len(req.CategoryIDs) > 0 {
		_, err = tx.ExecContext(ctx, `DELETE FROM writer_categories WHERE writer_id = $1`, writerID)
		if err != nil {
			return err
		}

		for _, catID := range req.CategoryIDs {
			_, err = tx.ExecContext(ctx, `INSERT INTO writer_categories (writer_id, category_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, writerID, catID)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}
