package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	host := os.Getenv("TEST_POSTGRES_HOST")
	port := os.Getenv("TEST_POSTGRES_PORT")
	user := os.Getenv("TEST_POSTGRES_USER")
	password := os.Getenv("TEST_POSTGRES_PASSWORD")
	dbname := os.Getenv("TEST_POSTGRES_DB")
	if host == "" {
		host = os.Getenv("POSTGRES_HOST")
	}
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = os.Getenv("POSTGRES_PORT")
	}
	if port == "" {
		port = "5435"
	}
	if user == "" {
		t.Fatal("test database misconfigured: set TEST_POSTGRES_USER (no app fallback for credentials). " +
			"Example (PowerShell, values without secrets): $env:TEST_POSTGRES_HOST='localhost'; $env:TEST_POSTGRES_PORT='5435'; $env:TEST_POSTGRES_USER='<test_user>'; $env:TEST_POSTGRES_DB='database_test'")
	}
	if password == "" {
		t.Fatal("test database misconfigured: set TEST_POSTGRES_PASSWORD (no app fallback for credentials).")
	}
	if dbname == "" {
		t.Fatal("test database misconfigured: set TEST_POSTGRES_DB to the isolated test database " +
			"(e.g. 'database_test'). Refusing to fall back to the application database 'database'.")
	}
	if dbname == "database" {
		t.Fatalf("refusing to run integration tests against the application database %q; set TEST_POSTGRES_DB to the isolated test database", dbname)
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=UTC", host, port, user, password, dbname)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		t.Fatalf("ping test db: %v", err)
	}
	var connectedName string
	if err := db.QueryRow(`SELECT current_database()`).Scan(&connectedName); err != nil {
		_ = db.Close()
		t.Fatalf("verify test database name: %v", err)
	}
	if connectedName != dbname || connectedName == "database" {
		_ = db.Close()
		t.Fatalf("refusing to run integration tests: connected database %q does not match isolated test database %q", connectedName, dbname)
	}
	// Close only after all t.Cleanup callbacks registered later (e.g. row
	// cleanup) have run: t.Cleanup runs LIFO, so the close registered here
	// executes last.
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

// cleanupTestUserRows removes only the rows created for one test, scoped by the
// user primary key returned at insert time. It must run against the isolated
// test database connection (verified by openTestDB above) — never against the
// application database.
func cleanupTestUserRows(t *testing.T, db *sql.DB, userID int64) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := db.ExecContext(ctx, `DELETE FROM novels WHERE author_id IN (SELECT writer_id FROM writers WHERE user_id = $1)`, userID); err != nil {
			t.Logf("cleanup novels for test user %d: %v", userID, err)
		}
		if _, err := db.ExecContext(ctx, `DELETE FROM writers WHERE user_id = $1`, userID); err != nil {
			t.Logf("cleanup writers for test user %d: %v", userID, err)
		}
		if _, err := db.ExecContext(ctx, `DELETE FROM users WHERE user_id = $1`, userID); err != nil {
			t.Logf("cleanup user %d: %v", userID, err)
		}
	})
}

func insertTestUserWithRevokedWriters(t *testing.T, db *sql.DB, suffix string, revokedCount int) (int64, []int64) {
	t.Helper()
	userID := int64(0)
	username := fmt.Sprintf("restore_user_%s_%d", suffix, time.Now().UnixNano())
	email := fmt.Sprintf("restore_user_%s_%d@test.local", suffix, time.Now().UnixNano())
	if err := db.QueryRowContext(context.Background(), `
		INSERT INTO users (username, email, password_hash, role, status, created_at, updated_at)
		VALUES ($1, $2, 'hash', 'reader', 'active', NOW(), NOW())
		RETURNING user_id`, username, email).Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	writerIDs := make([]int64, 0, revokedCount)
	for i := 0; i < revokedCount; i++ {
		var writerID int64
		if err := db.QueryRowContext(context.Background(), `
			INSERT INTO writers (user_id, pen_name, status, applied_at, approved_at, acted_by_admin_id)
			VALUES ($1, $2, 'revoked', NOW() - ($3 * INTERVAL '1 day'), NOW() - ($3 * INTERVAL '1 day'), 1)
			RETURNING writer_id`, userID, fmt.Sprintf("pen_%s_%d", suffix, i), i+1).Scan(&writerID); err != nil {
			t.Fatalf("insert writer row %d: %v", i, err)
		}
		writerIDs = append(writerIDs, writerID)
	}
	return userID, writerIDs
}

func TestRestoreUserWriterAccessSingleRevokedRow(t *testing.T) {
	db := openTestDB(t)

	ctx := context.Background()
	userID, _ := insertTestUserWithRevokedWriters(t, db, "single", 1)
	cleanupTestUserRows(t, db, userID)
	adminID := uint(1)

	repo := NewAuthRepository(db)
	if err := repo.RestoreUserWriterAccess(ctx, uint(userID), adminID); err != nil {
		t.Fatalf("restore should succeed: %v", err)
	}

	var role string
	if err := db.QueryRowContext(ctx, `SELECT role FROM users WHERE user_id = $1`, userID).Scan(&role); err != nil {
		t.Fatalf("read role: %v", err)
	}
	if role != "writer" {
		t.Fatalf("expected role writer after restore, got %s", role)
	}

	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM writers WHERE user_id = $1 AND status = 'approved'`, userID).Scan(&count); err != nil {
		t.Fatalf("read approved row count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one approved row after restore, got %d", count)
	}
}

func TestRestoreUserWriterAccessMultipleRevokedRows(t *testing.T) {
	db := openTestDB(t)

	ctx := context.Background()
	userID, _ := insertTestUserWithRevokedWriters(t, db, "multi", 3)
	cleanupTestUserRows(t, db, userID)
	adminID := uint(1)

	repo := NewAuthRepository(db)
	if err := repo.RestoreUserWriterAccess(ctx, uint(userID), adminID); err != nil {
		t.Fatalf("restore should succeed: %v", err)
	}

	var approvedCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM writers WHERE user_id = $1 AND status = 'approved'`, userID).Scan(&approvedCount); err != nil {
		t.Fatalf("read approved rows: %v", err)
	}
	if approvedCount != 3 {
		t.Fatalf("expected all revoked rows restored, got %d approved rows", approvedCount)
	}

	var revokedCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM writers WHERE user_id = $1 AND status = 'revoked'`, userID).Scan(&revokedCount); err != nil {
		t.Fatalf("read revoked rows: %v", err)
	}
	if revokedCount != 0 {
		t.Fatalf("expected no revoked rows remain, got %d", revokedCount)
	}
}

func TestRestoreUserWriterAccessRejectsNoHistory(t *testing.T) {
	db := openTestDB(t)

	ctx := context.Background()
	username := fmt.Sprintf("restore_no_history_%d", time.Now().UnixNano())
	email := fmt.Sprintf("restore_no_history_%d@test.local", time.Now().UnixNano())
	var userID int64
	if err := db.QueryRowContext(ctx, `
		INSERT INTO users (username, email, password_hash, role, status, created_at, updated_at)
		VALUES ($1, $2, 'hash', 'reader', 'active', NOW(), NOW())
		RETURNING user_id`, username, email).Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	cleanupTestUserRows(t, db, userID)

	repo := NewAuthRepository(db)
	err := repo.RestoreUserWriterAccess(ctx, uint(userID), 1)
	if err == nil {
		t.Fatal("expected restore to reject when no revoked writer history exists")
	}
	if err.Error() != "user has no prior revoked writer history" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRestoreUserWriterAccessRestoresPublishedNovelScope(t *testing.T) {
	db := openTestDB(t)

	ctx := context.Background()
	userID, writerIDs := insertTestUserWithRevokedWriters(t, db, "published", 2)
	cleanupTestUserRows(t, db, userID)
	for i, writerID := range writerIDs {
		_, err := db.ExecContext(ctx, `
			INSERT INTO novels (title, status, is_published, author_id, created_at, updated_at)
			VALUES ($1, 'published', TRUE, $2, NOW(), NOW())`, fmt.Sprintf("restore_novel_%d_%d", userID, i), writerID)
		if err != nil {
			t.Fatalf("insert novel %d: %v", i, err)
		}
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO novels (title, status, is_published, author_id, created_at, updated_at)
		VALUES ('restore_suspended_novel', 'suspended', TRUE, $1, NOW(), NOW())`, writerIDs[0])
	if err != nil {
		t.Fatalf("insert suspended novel: %v", err)
	}

	var beforeCount int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM novels n
		JOIN writers w ON w.writer_id = n.author_id
		WHERE w.user_id = $1
		  AND n.is_published = TRUE
		  AND n.status NOT IN ('suspended', 'banned')
		  AND w.status = 'approved'`, userID).Scan(&beforeCount); err != nil {
		t.Fatalf("count before restore: %v", err)
	}
	if beforeCount != 0 {
		t.Fatalf("expected zero visible published novels before restore, got %d", beforeCount)
	}

	repo := NewAuthRepository(db)
	if err := repo.RestoreUserWriterAccess(ctx, uint(userID), 1); err != nil {
		t.Fatalf("restore should succeed: %v", err)
	}

	var afterCount int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM novels n
		JOIN writers w ON w.writer_id = n.author_id
		WHERE w.user_id = $1
		  AND n.is_published = TRUE
		  AND n.status NOT IN ('suspended', 'banned')
		  AND w.status = 'approved'`, userID).Scan(&afterCount); err != nil {
		t.Fatalf("count after restore: %v", err)
	}
	if afterCount != 2 {
		t.Fatalf("expected 2 visible novels after restore, got %d", afterCount)
	}

	var suspendedVisible int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM novels n
		JOIN writers w ON w.writer_id = n.author_id
		WHERE w.user_id = $1
		  AND n.title = 'restore_suspended_novel'
		  AND n.is_published = TRUE
		  AND n.status NOT IN ('suspended', 'banned')
		  AND w.status = 'approved'`, userID).Scan(&suspendedVisible); err != nil {
		t.Fatalf("count suspended-visible: %v", err)
	}
	if suspendedVisible != 0 {
		t.Fatalf("expected suspended novel to remain hidden after restore, got %d", suspendedVisible)
	}
}
