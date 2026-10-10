package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"novel-be/internal/models"
)

var (
	ErrCategoryNotFound  = errors.New("category not found")
	ErrCategoryDuplicate = errors.New("category name already exists")
	ErrCategoryInUse     = errors.New("category is still in use")
)

type CategoryRepository interface {
	GetAllCategories() ([]models.Category, error)
	GetCategoryByID(ctx context.Context, id int) (*models.Category, error)
	GetCategoryByName(ctx context.Context, name string) (*models.Category, error)
	CreateCategory(ctx context.Context, name string) (*models.Category, error)
	UpdateCategory(ctx context.Context, id int, name string) (*models.Category, error)
	DeleteCategory(ctx context.Context, id int) error
	CountUsageByID(ctx context.Context, id int) (int, error)
}

type categoryRepository struct {
	db *sql.DB
}

func NewCategoryRepository(db *sql.DB) CategoryRepository {
	return &categoryRepository{db: db}
}

func (r *categoryRepository) GetAllCategories() ([]models.Category, error) {
	query := `SELECT category_id, name FROM categories ORDER BY category_id ASC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.CategoryID, &c.Name); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return categories, nil
}

func (r *categoryRepository) GetCategoryByID(ctx context.Context, id int) (*models.Category, error) {
	query := `SELECT category_id, name FROM categories WHERE category_id = $1`
	var c models.Category
	err := r.db.QueryRowContext(ctx, query, id).Scan(&c.CategoryID, &c.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *categoryRepository) GetCategoryByName(ctx context.Context, name string) (*models.Category, error) {
	query := `SELECT category_id, name FROM categories WHERE LOWER(TRIM(name)) = LOWER(TRIM($1))`
	var c models.Category
	err := r.db.QueryRowContext(ctx, query, name).Scan(&c.CategoryID, &c.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *categoryRepository) CreateCategory(ctx context.Context, name string) (*models.Category, error) {
	query := `INSERT INTO categories (name) VALUES ($1) RETURNING category_id, name`
	var c models.Category
	err := r.db.QueryRowContext(ctx, query, strings.TrimSpace(name)).Scan(&c.CategoryID, &c.Name)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *categoryRepository) UpdateCategory(ctx context.Context, id int, name string) (*models.Category, error) {
	query := `UPDATE categories SET name = $1 WHERE category_id = $2 RETURNING category_id, name`
	var c models.Category
	err := r.db.QueryRowContext(ctx, query, strings.TrimSpace(name), id).Scan(&c.CategoryID, &c.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *categoryRepository) DeleteCategory(ctx context.Context, id int) error {
	query := `DELETE FROM categories WHERE category_id = $1`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrCategoryNotFound
	}
	return nil
}

func (r *categoryRepository) CountUsageByID(ctx context.Context, id int) (int, error) {
	query := `
		SELECT 
			(SELECT COUNT(*) FROM novel_categories WHERE category_id = $1) +
			(SELECT COUNT(*) FROM writer_categories WHERE category_id = $1) AS total_count
	`
	var count int
	err := r.db.QueryRowContext(ctx, query, id).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
