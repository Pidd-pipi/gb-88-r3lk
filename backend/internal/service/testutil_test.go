package service

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/mockhub/mockhub/internal/config"
	"github.com/mockhub/mockhub/internal/model"
	"github.com/mockhub/mockhub/internal/repository"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Project{},
		&model.MockAPI{},
		&model.MockAPIVersion{},
		&model.ResponseTemplate{},
		&model.RequestLog{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

// newEndpointService wires an EndpointService with all of its repositories
// backed by the test database.
func newEndpointService(db *gorm.DB) *EndpointService {
	return NewEndpointService(
		repository.NewProjectRepository(db),
		repository.NewEndpointRepository(db),
		repository.NewEndpointVersionRepository(db),
		repository.NewUserRepository(db),
		discardLogger(),
	)
}

func newTestConfig() *config.Config {
	return &config.Config{
		Env:       "development",
		JWTSecret: "test-secret-that-is-long-enough-for-tests",
		JWTExpire: 1,
		BaseURL:   "http://localhost:3119",
	}
}

func createUser(t *testing.T, db *gorm.DB, username, role string) *model.User {
	t.Helper()
	u := &model.User{Username: username, PasswordHash: "not-a-real-hash", Role: role}
	if err := db.Create(u).Error; err != nil {
		t.Fatalf("create user %s: %v", username, err)
	}
	return u
}
