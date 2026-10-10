package service

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strings"

	"novel-be/internal/dto"
	"novel-be/internal/middleware"
	"novel-be/internal/models"
	"novel-be/internal/repository"
)

type AuditEvent struct {
	Action     string
	TargetType string
	TargetID   *int64
	Status     string
	IPAddress  string
	Metadata   map[string]interface{}
}

type AuditService interface {
	Record(ctx context.Context, event AuditEvent) error
	List(ctx context.Context, filter dto.AuditLogFilter) ([]models.AuditLog, int, error)
	GetByID(ctx context.Context, id int64) (*models.AuditLog, error)
	GetMetadata(ctx context.Context) (dto.AuditLogMetadataResponse, error)
}

// BackendActorAuditService is an optional extension for events whose actor was
// verified by a backend workflow before a JWT context exists.
type BackendActorAuditService interface {
	RecordWithActor(ctx context.Context, actorUserID *uint, actorRole string, event AuditEvent) error
}

type auditService struct{ repo repository.AuditRepository }

func NewAuditService(repo repository.AuditRepository) AuditService { return &auditService{repo: repo} }

func (s *auditService) Record(ctx context.Context, event AuditEvent) error {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok || userID == 0 {
		return errors.New("authenticated actor is required")
	}
	role, ok := middleware.GetRoleFromContext(ctx)
	if !ok || strings.TrimSpace(role) == "" {
		return errors.New("authenticated actor role is required")
	}
	return s.record(ctx, &userID, role, event)
}

func (s *auditService) RecordWithActor(ctx context.Context, actorUserID *uint, actorRole string, event AuditEvent) error {
	if actorUserID != nil && *actorUserID == 0 {
		return errors.New("backend actor is invalid")
	}
	if actorUserID == nil && strings.TrimSpace(actorRole) != "" {
		return errors.New("backend actor user is required")
	}
	return s.record(ctx, actorUserID, actorRole, event)
}

func RecordWithBackendActor(audit AuditService, ctx context.Context, actorUserID *uint, actorRole string, event AuditEvent) error {
	recorder, ok := audit.(BackendActorAuditService)
	if !ok {
		return errors.New("audit service does not support backend actors")
	}
	return recorder.RecordWithActor(ctx, actorUserID, actorRole, event)
}

func (s *auditService) record(ctx context.Context, actorUserID *uint, actorRole string, event AuditEvent) error {
	if strings.TrimSpace(event.Action) == "" || strings.TrimSpace(event.Status) == "" {
		return errors.New("audit action and status are required")
	}
	metadata, err := sanitizeAuditMetadata(event.Metadata)
	if err != nil {
		return err
	}
	return s.repo.Insert(ctx, models.AuditLog{ActorUserID: actorUserID, ActorRole: actorRole, Action: event.Action, TargetType: event.TargetType, TargetID: event.TargetID, Status: event.Status, IPAddress: normalizeIP(event.IPAddress), Metadata: metadata})
}

func (s *auditService) List(ctx context.Context, filter dto.AuditLogFilter) ([]models.AuditLog, int, error) {
	logs, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	count, err := s.repo.Count(ctx, filter)
	return logs, count, err
}
func (s *auditService) GetByID(ctx context.Context, id int64) (*models.AuditLog, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *auditService) GetMetadata(ctx context.Context) (dto.AuditLogMetadataResponse, error) {
	return s.repo.GetMetadata(ctx)
}

func normalizeIP(value string) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(value))
	if err == nil {
		return host
	}
	return strings.TrimSpace(value)
}

var sensitiveAuditKeys = map[string]bool{"password": true, "password_hash": true, "token": true, "access_token": true, "refresh_token": true, "jwt": true, "secret": true}

func sanitizeAuditMetadata(input map[string]interface{}) ([]byte, error) {
	cleaned, err := sanitizeAuditValue(input)
	if err != nil {
		return nil, err
	}
	return json.Marshal(cleaned)
}
func sanitizeAuditValue(value interface{}) (interface{}, error) {
	switch typed := value.(type) {
	case map[string]interface{}:
		cleaned := make(map[string]interface{}, len(typed))
		for key, item := range typed {
			if sensitiveAuditKeys[strings.ToLower(strings.TrimSpace(key))] {
				continue
			}
			next, err := sanitizeAuditValue(item)
			if err != nil {
				return nil, err
			}
			cleaned[key] = next
		}
		return cleaned, nil
	case []interface{}:
		cleaned := make([]interface{}, len(typed))
		for index, item := range typed {
			next, err := sanitizeAuditValue(item)
			if err != nil {
				return nil, err
			}
			cleaned[index] = next
		}
		return cleaned, nil
	default:
		return value, nil
	}
}
