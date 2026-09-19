package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/mockhub/mockhub/internal/config"
	"github.com/mockhub/mockhub/internal/handler"
	"github.com/mockhub/mockhub/internal/logger"
	"github.com/mockhub/mockhub/internal/model"
	"github.com/mockhub/mockhub/internal/repository"
	"github.com/mockhub/mockhub/internal/service"
	"github.com/mockhub/mockhub/internal/util"
)

// newTestServer boots the full router on top of an in-memory database so the
// version workflow is exercised end to end through HTTP.
func newTestServer(t *testing.T) (*httptest.Server, *gorm.DB) {
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
		&model.APIVersion{},
		&model.ResponseTemplate{},
		&model.RequestLog{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	cfg := &config.Config{
		Env:                "development",
		JWTSecret:          "test-secret-that-is-long-enough-for-tests",
		JWTExpire:          1,
		BaseURL:            "http://localhost:3119",
		CORSAllowedOrigins: "http://localhost:8119",
		MockRateLimit:      100000,
		AuthRateLimit:      100000,
	}
	log := logger.New("test")

	userRepo := repository.NewUserRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	endpointRepo := repository.NewEndpointRepository(db)
	versionRepo := repository.NewAPIVersionRepository(db)
	logRepo := repository.NewRequestLogRepository(db)

	authSvc := service.NewAuthService(cfg, userRepo, log)
	projectSvc := service.NewProjectService(projectRepo, log)
	endpointSvc := service.NewEndpointService(projectRepo, endpointRepo, versionRepo, userRepo, log)
	logSvc := service.NewRequestLogService(projectRepo, logRepo, log)
	mockEngine := service.NewMockEngine(endpointRepo, logRepo, log)

	h := &Handlers{
		Health:     handler.NewHealthHandler(db, log),
		Auth:       handler.NewAuthHandler(authSvc, log),
		Project:    handler.NewProjectHandler(projectSvc, cfg, log),
		Endpoint:   handler.NewEndpointHandler(endpointSvc, log),
		Mock:       handler.NewMockHandler(mockEngine, log),
		RequestLog: handler.NewRequestLogHandler(logSvc, log),
	}

	srv := httptest.NewServer(New(cfg, log, h))
	t.Cleanup(srv.Close)
	return srv, db
}

func authToken(t *testing.T, userID uint, role string) string {
	t.Helper()
	token, err := util.GenerateToken("test-secret-that-is-long-enough-for-tests", userID, role, time.Hour)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	return token
}

func doJSON(t *testing.T, method, url, token string, body any) (int, map[string]any, http.Header) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var parsed map[string]any
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &parsed); err != nil {
			t.Fatalf("response is not JSON: %s", raw)
		}
	}
	return resp.StatusCode, parsed, resp.Header
}

func dataOf(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("missing data object in %v", body)
	}
	return data
}

func TestVersionedEndpointHTTPFlow(t *testing.T) {
	srv, db := newTestServer(t)

	user := &model.User{Username: "dev", PasswordHash: "x", Role: "dev"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	token := authToken(t, user.ID, "dev")

	// Create a project.
	status, body, _ := doJSON(t, http.MethodPost, srv.URL+"/api/v1/projects", token,
		map[string]any{"name": "demo", "description": "d"})
	if status != http.StatusOK {
		t.Fatalf("create project status = %d, body = %v", status, body)
	}
	projectID := dataOf(t, body)["_id"].(string)

	// Create an endpoint: it must start at v1.
	status, body, _ = doJSON(t, http.MethodPost, srv.URL+"/api/v1/projects/"+projectID+"/apis", token,
		map[string]any{"path": "/api/users", "method": "GET", "statusCode": 200, "responseBody": `{"v":1}`})
	if status != http.StatusOK {
		t.Fatalf("create api status = %d, body = %v", status, body)
	}
	api := dataOf(t, body)
	apiID := api["_id"].(string)
	if api["currentVersion"].(float64) != 1 {
		t.Fatalf("currentVersion = %v, want 1", api["currentVersion"])
	}

	// Edit with the right baseVersion: v2 goes live.
	status, body, _ = doJSON(t, http.MethodPut, srv.URL+"/api/v1/projects/"+projectID+"/apis/"+apiID, token,
		map[string]any{"path": "/api/users", "method": "GET", "statusCode": 200, "responseBody": `{"v":2}`, "baseVersion": 1})
	if status != http.StatusOK || dataOf(t, body)["currentVersion"].(float64) != 2 {
		t.Fatalf("update status = %d, body = %v", status, body)
	}

	// Edit with a stale baseVersion: 409 and the live version is untouched.
	status, body, _ = doJSON(t, http.MethodPut, srv.URL+"/api/v1/projects/"+projectID+"/apis/"+apiID, token,
		map[string]any{"path": "/api/users", "method": "POST", "statusCode": 201, "responseBody": `{"stale":true}`, "baseVersion": 1})
	if status != http.StatusConflict {
		t.Fatalf("stale update status = %d, want 409; body = %v", status, body)
	}
	if body["code"].(float64) != 40900 {
		t.Fatalf("stale update code = %v, want 40900", body["code"])
	}
	status, body, _ = doJSON(t, http.MethodGet, srv.URL+"/api/v1/projects/"+projectID+"/apis/"+apiID, token, nil)
	if got := dataOf(t, body)["currentVersion"].(float64); got != 2 {
		t.Fatalf("currentVersion after failed update = %v, want 2", got)
	}

	// Version history: two immutable versions with the editor recorded.
	status, body, _ = doJSON(t, http.MethodGet, srv.URL+"/api/v1/projects/"+projectID+"/apis/"+apiID+"/versions", token, nil)
	if status != http.StatusOK {
		t.Fatalf("list versions status = %d", status)
	}
	versions, ok := body["data"].([]any)
	if !ok || len(versions) != 2 {
		t.Fatalf("versions = %v, want 2 entries", body["data"])
	}
	v2 := versions[0].(map[string]any)
	if v2["version"].(float64) != 2 || v2["editorName"].(string) != "dev" {
		t.Fatalf("v2 summary = %v", v2)
	}

	// Preview v1: the original body is preserved.
	status, body, _ = doJSON(t, http.MethodGet, srv.URL+"/api/v1/projects/"+projectID+"/apis/"+apiID+"/versions/1", token, nil)
	if status != http.StatusOK || dataOf(t, body)["responseBody"].(string) != `{"v":1}` {
		t.Fatalf("preview v1 status = %d, body = %v", status, body)
	}

	// Two publishes race from the same base: the first wins, the second gets
	// 409 and the live version stays where the winner put it.
	status, body, _ = doJSON(t, http.MethodPost, srv.URL+"/api/v1/projects/"+projectID+"/apis/"+apiID+"/versions/1/publish", token,
		map[string]any{"baseVersion": 2})
	if status != http.StatusOK || dataOf(t, body)["currentVersion"].(float64) != 1 {
		t.Fatalf("publish v1 status = %d, body = %v", status, body)
	}
	status, body, _ = doJSON(t, http.MethodPost, srv.URL+"/api/v1/projects/"+projectID+"/apis/"+apiID+"/versions/2/publish", token,
		map[string]any{"baseVersion": 2})
	if status != http.StatusConflict {
		t.Fatalf("losing publish status = %d, want 409; body = %v", status, body)
	}
	status, body, _ = doJSON(t, http.MethodGet, srv.URL+"/api/v1/projects/"+projectID+"/apis/"+apiID, token, nil)
	if got := dataOf(t, body)["currentVersion"].(float64); got != 1 {
		t.Fatalf("currentVersion after lost race = %v, want 1", got)
	}

	// The mock engine serves the rolled-back content and stamps the version.
	resp, err := http.Get(fmt.Sprintf("%s/mock/%s/api/users", srv.URL, projectID))
	if err != nil {
		t.Fatalf("mock request: %v", err)
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.Header.Get("X-Mock-Version") != "1" {
		t.Fatalf("X-Mock-Version = %q, want 1", resp.Header.Get("X-Mock-Version"))
	}
	if string(raw) != `{"v":1}` {
		t.Fatalf("mock body = %s, want {\"v\":1}", raw)
	}

	// The request log persists the effective version (re-read after restart of
	// the request path, i.e. a fresh query).
	status, body, _ = doJSON(t, http.MethodGet, srv.URL+"/api/v1/projects/"+projectID+"/logs", token, nil)
	if status != http.StatusOK {
		t.Fatalf("list logs status = %d", status)
	}
	logs, ok := dataOf(t, body)["logs"].([]any)
	if !ok || len(logs) != 1 {
		t.Fatalf("logs = %v, want 1 entry", dataOf(t, body)["logs"])
	}
	entry := logs[0].(map[string]any)
	if entry["apiVersion"].(float64) != 1 {
		t.Fatalf("logged apiVersion = %v, want 1", entry["apiVersion"])
	}
}
