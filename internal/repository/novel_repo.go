package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"

	"novel-be/internal/models"
)

func deriveNovelStateForResponse(dbStatus, latestBanReason string) (string, bool, string) {
	normalizedStatus := strings.TrimSpace(strings.ToLower(dbStatus))
	switch normalizedStatus {
	case "banned":
		return "suspended", true, strings.TrimSpace(latestBanReason)
	case "completed-published":
		return "completed-published", false, ""
	case "completed-draft":
		return "completed-draft", false, ""
	case "published":
		return "published", false, ""
	case "draft":
		return "draft", false, ""
	default:
		if normalizedStatus == "" {
			return "draft", false, ""
		}
		return normalizedStatus, false, strings.TrimSpace(latestBanReason)
	}
}

func publishedNovelScope(alias, writerAlias string) string {
	if alias == "" {
		alias = "n"
	}
	if writerAlias == "" {
		writerAlias = "w"
	}
	return "(" + alias + ".is_published = TRUE AND " + alias + ".status NOT IN ('suspended', 'banned') AND " + writerAlias + ".status = 'approved')"
}

func matchesPublishedScope(status string, isPublished bool, writerStatus string) bool {
	if !isPublished {
		return false
	}
	if strings.TrimSpace(strings.ToLower(writerStatus)) != "approved" {
		return false
	}
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "suspended", "banned":
		return false
	default:
		return true
	}
}

func matchesStatusFilter(status string, isPublished bool, writerStatus string, filter string) bool {
	normalizedStatus := strings.TrimSpace(strings.ToLower(status))
	switch strings.TrimSpace(strings.ToLower(filter)) {
	case "all":
		return matchesStatusFilter(status, isPublished, writerStatus, "published") || matchesStatusFilter(status, isPublished, writerStatus, "suspended")
	case "published":
		return matchesPublishedScope(status, isPublished, writerStatus)
	case "suspended":
		return normalizedStatus == "suspended" || normalizedStatus == "banned"
	case "banned":
		return normalizedStatus == "banned" || normalizedStatus == "suspended"
	default:
		return true
	}
}

func buildNovelWhereClause(filters []string) string {
	if len(filters) == 0 {
		return ""
	}
	return "WHERE " + strings.Join(filters, " AND ")
}

func GetNovels(db *sql.DB) ([]models.Novel, error) {
	rows, err := db.Query(`
		SELECT 
			n.novel_id, n.title, n.captions, n.introduction, n.cover_image,
			CASE
				WHEN n.status = 'suspended' THEN 'suspended'
				WHEN n.status = 'banned' THEN 'suspended'
				WHEN n.is_completed AND n.is_published THEN 'completed-published'
				WHEN n.is_completed THEN 'completed-draft'
				WHEN n.is_published THEN 'published'
				ELSE 'draft'
			END AS status,
			n.is_published, n.is_completed,
			n.author_id, n.views, n.created_at, n.updated_at,
			-- counts
			(SELECT COUNT(*) FROM chapters ch WHERE ch.novel_id = n.novel_id) AS chapter_count,
			(SELECT COUNT(*) FROM scenes s WHERE s.novel_id = n.novel_id) AS scene_count,
			w.name_lastname, w.pen_name,
			COALESCE(COUNT(DISTINCT l.id), 0) AS like_count,
			(SELECT COALESCE(COUNT(*), 0) FROM bookshelves b WHERE b.novel_id = n.novel_id) AS bookshelf_count,
			COALESCE((
				SELECT r.reason
				FROM reports r
				WHERE r.novel_id = n.novel_id
				  AND r.status = 'resolved'
				ORDER BY r.created_at DESC, r.report_id DESC
				LIMIT 1
			), '') AS ban_reason,
			-- รวมหมวดหมู่เป็น JSON Array
			COALESCE(
				json_agg(
					json_build_object('category_id', c.category_id, 'name', c.name)
				) FILTER (WHERE c.category_id IS NOT NULL), '[]'
			) AS categories_json

		FROM novels n
		LEFT JOIN writers w ON n.author_id = w.writer_id
		LEFT JOIN novel_categories nc ON n.novel_id = nc.novel_id
		LEFT JOIN categories c ON nc.category_id = c.category_id
		LEFT JOIN likes l ON n.novel_id = l.novel_id
		WHERE ` + publishedNovelScope("n", "w") + `
		GROUP BY n.novel_id, w.writer_id
		ORDER BY n.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	novels := []models.Novel{}
	for rows.Next() {
		var n models.Novel
		var authorName, penName *string
		var categoriesJSON []byte
		var banReason string

		err := rows.Scan(
			&n.ID, &n.Title, &n.Captions, &n.Introduction, &n.CoverImage, &n.Status, &n.IsPublished, &n.IsCompleted,
			&n.AuthorID, &n.Views, &n.CreatedAt, &n.UpdatedAt,
			&n.ChapterCount, &n.SceneCount,
			&authorName, &penName,
			&n.LikeCount, &n.BookshelfCount,
			&banReason,
			&categoriesJSON,
		)
		if err != nil {
			return nil, err
		}

		if authorName != nil {
			n.AuthorName = *authorName
		}
		if penName != nil {
			n.PenName = *penName
		}

		n.Status, n.IsBanned, n.BanReason = deriveNovelStateForResponse(n.Status, banReason)

		// แปลง JSON Byte ให้กลับกลายเป็น Array ใน Go
		if len(categoriesJSON) > 0 {
			err = json.Unmarshal(categoriesJSON, &n.Categories)
			if err != nil {
				return nil, err
			}
			for _, cat := range n.Categories {
				n.CategoryIDs = append(n.CategoryIDs, cat.CategoryID)
			}
		}

		novels = append(novels, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return novels, nil
}

func (r *postgresNovelRepository) ListAdminNovels(ctx context.Context, search, status string, categoryID, page, limit int) ([]models.Novel, int, error) {
	offset := (page - 1) * limit
	args := []interface{}{}
	addArg := func(value interface{}) string {
		args = append(args, value)
		return "$" + strconv.Itoa(len(args))
	}
	where := []string{}

	switch status {
	case "published":
		where = append(where, publishedNovelScope("n", "w"))
	case "suspended":
		where = append(where, "n.status = "+addArg("suspended"))
	case "all", "":
		where = append(where, "("+publishedNovelScope("n", "w")+" OR n.status = "+addArg("suspended")+")")
	}
	if search != "" {
		placeholder := addArg("%" + search + "%")
		where = append(where, "(n.title ILIKE "+placeholder+" OR w.pen_name ILIKE "+placeholder+" OR w.name_lastname ILIKE "+placeholder+")")
	}
	if categoryID > 0 {
		where = append(where, "nc.category_id = "+addArg(categoryID))
	}
	whereClause := buildNovelWhereClause(where)

	countQuery := `
		SELECT COUNT(DISTINCT n.novel_id)
		FROM novels n
		LEFT JOIN writers w ON n.author_id = w.writer_id
		LEFT JOIN novel_categories nc ON n.novel_id = nc.novel_id
		` + whereClause
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT
			n.novel_id, n.title, n.captions, n.introduction, n.cover_image,
			n.status, n.is_published, n.is_completed, n.author_id,
			n.views, n.created_at, n.updated_at,
			(SELECT COUNT(*) FROM chapters ch WHERE ch.novel_id = n.novel_id),
			(SELECT COUNT(*) FROM scenes s WHERE s.novel_id = n.novel_id),
			w.name_lastname, w.pen_name,
			(SELECT COUNT(*) FROM likes l WHERE l.novel_id = n.novel_id),
			(SELECT COUNT(*) FROM bookshelves b WHERE b.novel_id = n.novel_id),
			COALESCE(json_agg(json_build_object('category_id', c.category_id, 'name', c.name)) FILTER (WHERE c.category_id IS NOT NULL), '[]')
		FROM novels n
		LEFT JOIN writers w ON n.author_id = w.writer_id
		LEFT JOIN novel_categories nc ON n.novel_id = nc.novel_id
		LEFT JOIN categories c ON nc.category_id = c.category_id
		` + whereClause + `
		GROUP BY n.novel_id, w.writer_id
		ORDER BY n.created_at DESC, n.novel_id DESC
		LIMIT $` + strconv.Itoa(len(args)+1) + ` OFFSET $` + strconv.Itoa(len(args)+2)

	queryArgs := append(append([]interface{}{}, args...), limit, offset)
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	novels := make([]models.Novel, 0)
	for rows.Next() {
		var novel models.Novel
		var authorName, penName *string
		var categoriesJSON []byte
		if err := rows.Scan(
			&novel.ID, &novel.Title, &novel.Captions, &novel.Introduction, &novel.CoverImage,
			&novel.Status, &novel.IsPublished, &novel.IsCompleted, &novel.AuthorID,
			&novel.Views, &novel.CreatedAt, &novel.UpdatedAt,
			&novel.ChapterCount, &novel.SceneCount, &authorName, &penName,
			&novel.LikeCount, &novel.BookshelfCount, &categoriesJSON,
		); err != nil {
			return nil, 0, err
		}
		if authorName != nil {
			novel.AuthorName = *authorName
		}
		if penName != nil {
			novel.PenName = *penName
		}
		if err := json.Unmarshal(categoriesJSON, &novel.Categories); err != nil {
			return nil, 0, err
		}
		for _, category := range novel.Categories {
			novel.CategoryIDs = append(novel.CategoryIDs, category.CategoryID)
		}
		novels = append(novels, novel)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return novels, total, nil
}

func GetNovelByID(db *sql.DB, id int) (*models.Novel, error) {
	row := db.QueryRow(`
		SELECT 
			n.novel_id, n.title, n.captions, n.introduction, n.cover_image,
			CASE
				WHEN n.status = 'suspended' THEN 'suspended'
				WHEN n.status = 'banned' THEN 'suspended'
				WHEN n.is_completed AND n.is_published THEN 'completed-published'
				WHEN n.is_completed THEN 'completed-draft'
				WHEN n.is_published THEN 'published'
				ELSE 'draft'
			END AS status,
			n.is_published, n.is_completed,
			n.author_id, n.views, n.created_at, n.updated_at,
			-- counts
			(SELECT COUNT(*) FROM chapters ch WHERE ch.novel_id = n.novel_id) AS chapter_count,
			(SELECT COUNT(*) FROM scenes s WHERE s.novel_id = n.novel_id) AS scene_count,
			w.name_lastname, w.pen_name,
			COALESCE(COUNT(DISTINCT l.id), 0) AS like_count,
			(SELECT COALESCE(COUNT(*), 0) FROM bookshelves b WHERE b.novel_id = n.novel_id) AS bookshelf_count,
			COALESCE((
				SELECT r.reason
				FROM reports r
				WHERE r.novel_id = n.novel_id
				  AND r.status = 'resolved'
				ORDER BY r.created_at DESC, r.report_id DESC
				LIMIT 1
			), '') AS ban_reason,
			-- รวมหมวดหมู่เป็น JSON Array
			COALESCE(
				json_agg(
					json_build_object('category_id', c.category_id, 'name', c.name)
				) FILTER (WHERE c.category_id IS NOT NULL), '[]'
			) AS categories_json

		FROM novels n
		LEFT JOIN writers w ON n.author_id = w.writer_id
		LEFT JOIN novel_categories nc ON n.novel_id = nc.novel_id
		LEFT JOIN categories c ON nc.category_id = c.category_id
		LEFT JOIN likes l ON n.novel_id = l.novel_id
		WHERE n.novel_id = $1
		GROUP BY n.novel_id, w.writer_id
	`, id)

	var n models.Novel
	var authorName, penName *string
	var categoriesJSON []byte

	var banReason string
	err := row.Scan(
		&n.ID, &n.Title, &n.Captions, &n.Introduction, &n.CoverImage, &n.Status, &n.IsPublished, &n.IsCompleted,
		&n.AuthorID, &n.Views, &n.CreatedAt, &n.UpdatedAt,
		&n.ChapterCount, &n.SceneCount,
		&authorName, &penName,
		&n.LikeCount, &n.BookshelfCount,
		&banReason,
		&categoriesJSON,
	)
	if err != nil {
		return nil, err
	}
	if authorName != nil {
		n.AuthorName = *authorName
	}
	if penName != nil {
		n.PenName = *penName
	}

	n.Status, n.IsBanned, n.BanReason = deriveNovelStateForResponse(n.Status, banReason)

	// แปลง JSON
	if len(categoriesJSON) > 0 {
		err = json.Unmarshal(categoriesJSON, &n.Categories)
		if err != nil {
			return nil, err
		}
		for _, cat := range n.Categories {
			n.CategoryIDs = append(n.CategoryIDs, cat.CategoryID)
		}
	}

	return &n, nil
}

func IncrementNovelViews(db *sql.DB, novelID int) error {
	_, err := db.Exec(`
		UPDATE novels
		SET views = views + 1
		WHERE novel_id = $1
	`, novelID)
	return err
}

func CreateNovel(db *sql.DB, novel models.Novel) (int, error) {
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var id int
	// 1. สร้างนิยาย และขอ ID ของนิยายที่เพิ่งสร้างคืนมา
	err = tx.QueryRowContext(ctx, `
		INSERT INTO novels (title, captions, introduction, cover_image, status, is_published, is_completed, author_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING novel_id
	`, novel.Title, novel.Captions, novel.Introduction, novel.CoverImage, novel.Status, novel.IsPublished, novel.IsCompleted, novel.AuthorID).Scan(&id)
	if err != nil {
		return 0, err
	}

	// 2. ถ้ามีการส่งหมวดหมู่ (CategoryIDs) มาด้วย ให้บันทึกลงตารางกลาง
	if len(novel.CategoryIDs) > 0 {
		for _, catID := range novel.CategoryIDs {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO novel_categories (novel_id, category_id) 
				VALUES ($1, $2)
			`, id, catID)

			if err != nil {
				return id, err // ถ้า Error ตอน insert หมวดหมู่ ให้ส่งกลับไปเลย และ rollback
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return id, nil
}

func UpdateNovel(db *sql.DB, novel models.Novel) error {
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Update core novel fields, include cover_image
	if _, err = tx.ExecContext(ctx, `
		UPDATE novels
		SET title = $1,
		    captions = $2,
		    introduction = $3,
		    cover_image = $4,
		    status = $5,
		    is_published = $6,
		    is_completed = $7,
		    updated_at = NOW()
		WHERE novel_id = $8
	`, novel.Title, novel.Captions, novel.Introduction, novel.CoverImage, novel.Status, novel.IsPublished, novel.IsCompleted, novel.ID); err != nil {
		return err
	}

	// Replace category mappings: delete existing, then insert new if provided
	if _, err = tx.ExecContext(ctx, `DELETE FROM novel_categories WHERE novel_id = $1`, novel.ID); err != nil {
		return err
	}
	if len(novel.CategoryIDs) > 0 {
		// Deduplicate category IDs to avoid unique constraint violations
		seen := make(map[int]bool)
		for _, catID := range novel.CategoryIDs {
			if catID == 0 {
				continue
			}
			if seen[catID] {
				continue
			}
			seen[catID] = true
			// Use ON CONFLICT DO NOTHING to be idempotent in case row already exists
			if _, err = tx.ExecContext(ctx, `INSERT INTO novel_categories (novel_id, category_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, novel.ID, catID); err != nil {
				return err
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func DeleteNovel(db *sql.DB, id int) error {
	_, err := db.Exec(`
		DELETE FROM novels
		WHERE novel_id = $1
	`, id)
	return err
}

func GetNovelsByAuthorID(db *sql.DB, authorID int) ([]models.Novel, error) {
	rows, err := db.Query(`
		SELECT 
			n.novel_id, n.title, n.captions, n.introduction, n.cover_image,
			CASE
				WHEN n.status = 'suspended' THEN 'suspended'
				WHEN n.status = 'banned' THEN 'suspended'
				WHEN n.is_completed AND n.is_published THEN 'completed-published'
				WHEN n.is_completed THEN 'completed-draft'
				WHEN n.is_published THEN 'published'
				ELSE 'draft'
			END AS status,
			n.is_published, n.is_completed,
			n.author_id, n.views, n.created_at, n.updated_at,
			(SELECT COUNT(*) FROM chapters ch WHERE ch.novel_id = n.novel_id) AS chapter_count,
			(SELECT COUNT(*) FROM scenes s WHERE s.novel_id = n.novel_id) AS scene_count,
			w.name_lastname, w.pen_name,
			(SELECT COALESCE(COUNT(*), 0) FROM likes l WHERE l.novel_id = n.novel_id) AS like_count,
			(SELECT COALESCE(COUNT(*), 0) FROM bookshelves b WHERE b.novel_id = n.novel_id) AS bookshelf_count,
			COALESCE((
				SELECT r.reason
				FROM reports r
				WHERE r.novel_id = n.novel_id
				  AND r.status = 'resolved'
				ORDER BY r.created_at DESC, r.report_id DESC
				LIMIT 1
			), '') AS ban_reason,
			-- รวมหมวดหมู่เป็น JSON Array
			COALESCE(
				json_agg(
					json_build_object('category_id', c.category_id, 'name', c.name)
				) FILTER (WHERE c.category_id IS NOT NULL), '[]'
			) AS categories_json

		FROM novels n
		LEFT JOIN writers w ON n.author_id = w.writer_id
		LEFT JOIN novel_categories nc ON n.novel_id = nc.novel_id
		LEFT JOIN categories c ON nc.category_id = c.category_id
		WHERE n.author_id = $1
		GROUP BY n.novel_id, w.writer_id
		ORDER BY n.created_at DESC
	`, authorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var novels []models.Novel
	for rows.Next() {
		var n models.Novel
		var authorName, penName *string
		var categoriesJSON []byte
		var banReason string

		err := rows.Scan(
			&n.ID, &n.Title, &n.Captions, &n.Introduction, &n.CoverImage, &n.Status, &n.IsPublished, &n.IsCompleted,
			&n.AuthorID, &n.Views, &n.CreatedAt, &n.UpdatedAt,
			&n.ChapterCount, &n.SceneCount,
			&authorName, &penName,
			&n.LikeCount, &n.BookshelfCount,
			&banReason,
			&categoriesJSON,
		)
		if err != nil {
			return nil, err
		}

		if authorName != nil {
			n.AuthorName = *authorName
		}
		if penName != nil {
			n.PenName = *penName
		}

		n.Status, n.IsBanned, n.BanReason = deriveNovelStateForResponse(n.Status, banReason)

		if len(categoriesJSON) > 0 {
			err = json.Unmarshal(categoriesJSON, &n.Categories)
			if err != nil {
				return nil, err
			}
		}

		novels = append(novels, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return novels, nil
}
