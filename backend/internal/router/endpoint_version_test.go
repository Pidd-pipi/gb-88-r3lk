package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/mockhub/mockhub/internal/config"
	"github.com/mockhub/mockhub/internal/handler"
	"github.com/mockhub/mockhub/internal/model"
	"github.com/mockhub/mockhub/internal/repository"
	"github.com/mockhub/mockhub/internal/service"
	"github.com/mockhub/mockhub/internal/util"
)

// testEnv boots the full Gin engine against an in-memory SQLite database so
// the versioning API can be exercised end to end over HTTP.
type testEnv struct {
	engine http.Handler
	token  string
	db     *gorm.DB
}

func newTestEnv(t *testing.T) *testEnv {
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

	cfg := &config.Config{
		Env:                "development",
		JWTSecret:          "router-test-secret-that-is-long-enough",
		BaseURL:            "http://localhost:3119",
		CORSAllowedOrigins: "http://localhost:8119",
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	userRepo := repository.NewUserRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	endpointRepo := repository.NewEndpointRepository(db)
	versionRepo := repository.NewEndpointVersionRepository(db)
	logRepo := repository.NewRequestLogRepository(db)

	endpointSvc := service.NewEndpointService(projectRepo, endpointRepo, versionRepo, userRepo, logger)
	handlers := &Handlers{
		Health:     handler.NewHealthHandler(db, logger),
		Endpoint:   handler.NewEndpointHandler(endpointSvc, logger),
		Mock:       handler.NewMockHandler(service.NewMockEngine(endpointRepo, logRepo, logger), logger),
		RequestLog: handler.NewRequestLogHandler(service.NewRequestLogService(projectRepo, logRepo, logger), logger),
	}
	engine := New(cfg, logger, handlers)

	user := &model.User{Username: "dev", PasswordHash: "x", Role: service.RoleDev}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	project := &model.Project{Name: "p", UserID: user.ID, BaseURL: "http://localhost:3119/mock/1"}
	if err := db.Create(project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	token, err := util.GenerateToken(cfg.JWTSecret, user.ID, user.Role, time.Hour)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	return &testEnv{engine: engine, token: token, db: db}
}

// do issues an authenticated JSON request and decodes the unified envelope.
func (e *testEnv) do(t *testing.T, method, path string, payload any) (int, map[string]any) {
	t.Helper()
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		body = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.token)
	rec := httptest.NewRecorder()
	e.engine.ServeHTTP(rec, req)

	var envelope map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("%s %s: decode envelope: %v (body: %s)", method, path, err, rec.Body.String())
	}
	return rec.Code, envelope
}

// envelopeData asserts code == 0 and returns the data object.
func envelopeData(t *testing.T, envelope map[string]any) map[string]any {
	t.Helper()
	if code := envelope["code"].(float64); code != 0 {
		t.Fatalf("envelope code = %v, message = %v", code, envelope["message"])
	}
	data, ok := envelope["data"].(map[string]any)
	if !ok {
		t.Fatalf("envelope data is not an object: %v", envelope["data"])
	}
	return data
}

func TestEndpointVersioningOverHTTP(t *testing.T) {
	env := newTestEnv(t)

	// Create: the endpoint starts at v1.
	_, env1 := env.do(t, http.MethodPost, "/api/v1/projects/1/apis", map[string]any{
		"path": "/api/users", "method": "GET", "statusCode": 200, "responseBody": `{"v":1}`,
	})
	created := envelopeData(t, env1)
	if got := created["currentVersion"].(float64); got != 1 {
		t.Fatalf("currentVersion = %v, want 1", got)
	}

	// Edit with a matching baseVersion: generates v2.
	_, env2 := env.do(t, http.MethodPut, "/api/v1/projects/1/apis/1", map[string]any{
		"path": "/api/users", "method": "GET", "statusCode": 200, "responseBody": `{"v":2}`, "baseVersion": 1,
	})
	if got := envelopeData(t, env2)["currentVersion"].(float64); got != 2 {
		t.Fatalf("currentVersion after edit = %v, want 2", got)
	}

	// A stale editor is rejected with the conflict code and changes nothing.
	_, envStale := env.do(t, http.MethodPut, "/api/v1/projects/1/apis/1", map[string]any{
		"path": "/api/users", "method": "GET", "statusCode": 200, "responseBody": `{"v":99}`, "baseVersion": 1,
	})
	if got := envStale["code"].(float64); got != 40900 {
		t.Fatalf("stale edit code = %v, want 40900", got)
	}
	_, envGet := env.do(t, http.MethodGet, "/api/v1/projects/1/apis/1", nil)
	current := envelopeData(t, envGet)
	if got := current["currentVersion"].(float64); got != 2 {
		t.Fatalf("currentVersion after stale edit = %v, want 2", got)
	}
	if got := current["responseBody"].(string); got != `{"v":2}` {
		t.Fatalf("responseBody after stale edit = %q", got)
	}

	// Version history: two immutable snapshots with operator stamps.
	_, envVersions := env.do(t, http.MethodGet, "/api/v1/projects/1/apis/1/versions", nil)
	list, ok := envVersions["data"].([]any)
	if !ok || len(list) != 2 {
		t.Fatalf("versions = %v, want 2 entries", envVersions["data"])
	}
	v2 := list[0].(map[string]any)
	v1 := list[1].(map[string]any)
	if v2["version"].(float64) != 2 || v1["version"].(float64) != 1 {
		t.Fatalf("version order = %v/%v, want 2/1", v2["version"], v1["version"])
	}
	if v2["createdByName"].(string) != "dev" || v2["createdAt"].(string) == "" {
		t.Fatalf("v2 operator stamp = %v %v", v2["createdByName"], v2["createdAt"])
	}

	// Concurrent publish: both publishers saw current=2, only the first wins.
	_, envPub1 := env.do(t, http.MethodPost, "/api/v1/projects/1/apis/1/versions/1/publish", map[string]any{
		"expectedVersion": 2,
	})
	published := envelopeData(t, envPub1)
	if got := published["currentVersion"].(float64); got != 1 {
		t.Fatalf("currentVersion after publish = %v, want 1", got)
	}
	if got := published["responseBody"].(string); got != `{"v":1}` {
		t.Fatalf("responseBody after rollback = %q", got)
	}
	_, envPub2 := env.do(t, http.MethodPost, "/api/v1/projects/1/apis/1/versions/2/publish", map[string]any{
		"expectedVersion": 2,
	})
	if got := envPub2["code"].(float64); got != 40900 {
		t.Fatalf("losing publish code = %v, want 40900", got)
	}
	// The failed publish must not have moved the current version.
	_, envGet2 := env.do(t, http.MethodGet, "/api/v1/projects/1/apis/1", nil)
	if got := envelopeData(t, envGet2)["currentVersion"].(float64); got != 1 {
		t.Fatalf("currentVersion after failed publish = %v, want 1", got)
	}

	// Mock requests report the effective version and logs persist it.
	req := httptest.NewRequest(http.MethodGet, "/mock/1/api/users", nil)
	rec := httptest.NewRecorder()
	env.engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("mock status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Mock-API-Version"); got != "1" {
		t.Fatalf("X-Mock-API-Version = %q, want 1", got)
	}
	if got := rec.Body.String(); got != `{"v":1}` {
		t.Fatalf("mock body = %q, want rolled-back content", got)
	}
	_, envLogs := env.do(t, http.MethodGet, "/api/v1/projects/1/logs", nil)
	logsData, ok := envLogs["data"].(map[string]any)
	if !ok {
		t.Fatalf("logs data = %v", envLogs["data"])
	}
	logs := logsData["logs"].([]any)
	if len(logs) != 1 {
		t.Fatalf("log count = %d, want 1", len(logs))
	}
	if got := logs[0].(map[string]any)["apiVersion"].(float64); got != 1 {
		t.Fatalf("logged apiVersion = %v, want 1", got)
	}
}

func TestLegacyEndpointOverHTTP(t *testing.T) {
	env := newTestEnv(t)

	// Simulate a pre-versioning endpoint row: no snapshots, current_version=0.
	legacy := &model.MockAPI{ProjectID: 1, Path: "/legacy", Method: "GET", StatusCode: 200, ResponseBody: `{"old":true}`}
	if err := env.db.Create(legacy).Error; err != nil {
		t.Fatalf("create legacy endpoint: %v", err)
	}

	// The list endpoint shows version 0 for legacy rows and they stay editable.
	_, envList := env.do(t, http.MethodGet, "/api/v1/projects/1/apis", nil)
	apis := envList["data"].([]any)
	if len(apis) != 1 || apis[0].(map[string]any)["currentVersion"].(float64) != 0 {
		t.Fatalf("legacy list entry = %v", apis)
	}

	_, envEdit := env.do(t, http.MethodPut, fmt.Sprintf("/api/v1/projects/1/apis/%d", legacy.ID), map[string]any{
		"path": "/legacy", "method": "GET", "statusCode": 200, "responseBody": `{"new":true}`,
	})
	if got := envelopeData(t, envEdit)["currentVersion"].(float64); got != 2 {
		t.Fatalf("legacy currentVersion after edit = %v, want 2", got)
	}

	// The pre-edit content was preserved as v1.
	_, envVersions := env.do(t, http.MethodGet, fmt.Sprintf("/api/v1/projects/1/apis/%d/versions", legacy.ID), nil)
	versions := envVersions["data"].([]any)
	if len(versions) != 2 {
		t.Fatalf("legacy versions = %v, want 2", versions)
	}
	if got := versions[1].(map[string]any)["responseBody"].(string); got != `{"old":true}` {
		t.Fatalf("backfilled v1 body = %q", got)
	}

	// Mock requests against the edited legacy endpoint log the new version.
	req := httptest.NewRequest(http.MethodGet, "/mock/1/legacy", nil)
	rec := httptest.NewRecorder()
	env.engine.ServeHTTP(rec, req)
	if rec.Body.String() != `{"new":true}` {
		t.Fatalf("legacy mock body = %q", rec.Body.String())
	}
	if got := rec.Header().Get("X-Mock-API-Version"); got != "2" {
		t.Fatalf("legacy X-Mock-API-Version = %q, want 2", got)
	}
}
