package service

import (
	"context"
	"fmt"
	"log/slog"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/mockhub/mockhub/internal/model"
)

// SeedService idempotently inserts demo users, a demo project and its
// endpoints plus sample request logs on first boot.
type SeedService struct {
	db      *gorm.DB
	baseURL string
	logger  *slog.Logger
}

// NewSeedService builds a SeedService.
func NewSeedService(db *gorm.DB, baseURL string, logger *slog.Logger) *SeedService {
	return &SeedService{db: db, baseURL: baseURL, logger: logger}
}

// EnsureSeedData creates the demo dataset if the users table is empty.
func (s *SeedService) EnsureSeedData(ctx context.Context) error {
	var count int64
	if err := s.db.WithContext(ctx).Model(&model.User{}).Count(&count).Error; err != nil {
		return fmt.Errorf("seed: count users: %w", err)
	}
	if count > 0 {
		return nil
	}

	hash := func(pwd string) (string, error) {
		b, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
		if err != nil {
			return "", fmt.Errorf("hash seed password: %w", err)
		}
		return string(b), nil
	}

	leadHash, err := hash("lead123")
	if err != nil {
		return err
	}
	devHash, err := hash("dev123")
	if err != nil {
		return err
	}

	lead := &model.User{Username: "lead", PasswordHash: leadHash, Role: RoleLead}
	dev := &model.User{Username: "dev", PasswordHash: devHash, Role: RoleDev}
	if err := s.db.WithContext(ctx).Create(&lead).Error; err != nil {
		return fmt.Errorf("seed: create lead: %w", err)
	}
	if err := s.db.WithContext(ctx).Create(&dev).Error; err != nil {
		return fmt.Errorf("seed: create dev: %w", err)
	}

	project := &model.Project{
		Name:        "用户管理系统",
		Description: "用户 CRUD Mock 接口，包含动态数据与条件响应示例",
		UserID:      lead.ID,
	}
	if err := s.db.WithContext(ctx).Create(&project).Error; err != nil {
		return fmt.Errorf("seed: create project: %w", err)
	}
	project.BaseURL = fmt.Sprintf("%s/mock/%d", s.baseURL, project.ID)
	if err := s.db.WithContext(ctx).Save(project).Error; err != nil {
		return fmt.Errorf("seed: set base url: %w", err)
	}

	adminCondition := []model.ConditionRule{{
		Field:        "role",
		Operator:     "equals",
		Value:        "admin",
		StatusCode:   200,
		ResponseBody: `{"code":0,"message":"ok","data":[{"id":1,"name":"管理员","email":"admin@example.com","role":"admin"},{"id":2,"name":"运营","email":"ops@example.com","role":"admin"}]}`,
	}}

	endpoints := []*model.MockAPI{
		{
			ProjectID:    project.ID,
			Path:         "/api/users",
			Method:       "GET",
			StatusCode:   200,
			ResponseBody: `{"code":0,"message":"ok","data":[{{repeat 5}}{"id":{{number.int 1 999}},"name":"{{name.fullName}}","email":"{{internet.email}}","role":"user"}{{endrepeat}}]}`,
			Conditions:   adminCondition,
		},
		{
			ProjectID:    project.ID,
			Path:         "/api/users/:id",
			Method:       "GET",
			StatusCode:   200,
			ResponseBody: `{"code":0,"message":"ok","data":{"id":"{{path.id}}","name":"{{name.fullName}}","email":"{{internet.email}}","createdAt":"{{date.timestamp}}"}}`,
		},
		{
			ProjectID:    project.ID,
			Path:         "/api/users",
			Method:       "POST",
			StatusCode:   201,
			ResponseBody: `{"code":0,"message":"created","data":{"id":{{number.int 1000 9999}},"name":"{{name.fullName}}","email":"{{internet.email}}","createdAt":"{{date.timestamp}}"}}`,
		},
		{
			ProjectID:    project.ID,
			Path:         "/api/users/:id",
			Method:       "PUT",
			StatusCode:   200,
			ResponseBody: `{"code":0,"message":"updated","data":{"id":"{{path.id}}","updated":true,"updatedAt":"{{date.timestamp}}"}}`,
		},
		{
			ProjectID:    project.ID,
			Path:         "/api/users/:id",
			Method:       "DELETE",
			StatusCode:   204,
			ResponseBody: ``,
		},
	}
	for _, ep := range endpoints {
		ep.CurrentVersion = 1
		if err := s.db.WithContext(ctx).Create(ep).Error; err != nil {
			return fmt.Errorf("seed: create endpoint: %w", err)
		}
		if err := s.db.WithContext(ctx).Create(model.SnapshotFrom(ep, 1, lead.ID, lead.Username)).Error; err != nil {
			return fmt.Errorf("seed: create endpoint version: %w", err)
		}
	}

	// Seed 10 request log samples against the GET /api/users/:id endpoint (v1).
	usersByID := endpoints[1]
	for i := 1; i <= 10; i++ {
		logEntry := &model.RequestLog{
			ProjectID:      project.ID,
			APIID:          usersByID.ID,
			APIVersion:     usersByID.CurrentVersion,
			Method:         "GET",
			Path:           fmt.Sprintf("/api/users/%d", i),
			Headers:        map[string]string{"Accept": "application/json", "User-Agent": "seed/1.0"},
			Query:          map[string]string{"page": "1"},
			Body:           nil,
			ResponseStatus: 200,
			ResponseBody:   fmt.Sprintf(`{"code":0,"data":[{"id":%d,"name":"Seed User %d"}]}`, i, i),
		}
		if err := s.db.WithContext(ctx).Create(logEntry).Error; err != nil {
			return fmt.Errorf("seed: create request log: %w", err)
		}
	}

	s.logger.Info("seed data inserted", "project_id", project.ID, "users", 2, "endpoints", len(endpoints), "logs", 10)
	return nil
}
