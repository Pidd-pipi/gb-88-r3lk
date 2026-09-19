package service

import (
	"errors"
	"testing"

	"github.com/mockhub/mockhub/internal/constants"
	"github.com/mockhub/mockhub/internal/dto"
	"github.com/mockhub/mockhub/internal/model"
	"github.com/mockhub/mockhub/internal/repository"
)

func intPtr(i int) *int { return &i }

// requireConflict asserts an error is the user-facing version conflict.
func requireConflict(t *testing.T, err error) {
	t.Helper()
	var appErr *constants.AppError
	if !errors.As(err, &appErr) || appErr.Code != constants.CodeConflict {
		t.Fatalf("err = %v, want AppError with CodeConflict", err)
	}
}

func TestEndpointVersioningLifecycle(t *testing.T) {
	db := newTestDB(t)
	projects := repository.NewProjectRepository(db)
	svc := newEndpointService(db)
	dev := createUser(t, db, "dev", RoleDev)
	project, err := NewProjectService(projects, discardLogger()).Create("p", "d", dev.ID, "http://localhost:3119")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	// Create generates v1 stamped with the operator.
	created, err := svc.Create(project.ID, dev.ID, RoleDev, dto.EndpointRequest{
		Path: "/api/users/:id", Method: "GET", StatusCode: 200, ResponseBody: `{"v":1}`,
		ResponseHeaders: map[string]string{"X-Demo": "1"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.CurrentVersion != 1 {
		t.Fatalf("CurrentVersion = %d, want 1", created.CurrentVersion)
	}

	// Every edit generates a new version. Fields left untouched by the edit
	// request (here: response headers) keep their previous content.
	updated, err := svc.Update(project.ID, created.ID, dev.ID, RoleDev, dto.EndpointRequest{
		Path: "/api/users/:id", Method: "GET", StatusCode: 200, ResponseBody: `{"v":2}`,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.CurrentVersion != 2 {
		t.Fatalf("CurrentVersion after edit = %d, want 2", updated.CurrentVersion)
	}
	if updated.ResponseHeaders["X-Demo"] != "1" {
		t.Fatalf("headers not preserved on edit: %+v", updated.ResponseHeaders)
	}

	versions, err := svc.ListVersions(project.ID, created.ID, dev.ID, RoleDev)
	if err != nil {
		t.Fatalf("ListVersions: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("len(versions) = %d, want 2", len(versions))
	}
	// Newest first, with operator and content preserved.
	if versions[0].Version != 2 || versions[1].Version != 1 {
		t.Fatalf("version order = [%d %d], want [2 1]", versions[0].Version, versions[1].Version)
	}
	if versions[0].CreatedBy != dev.ID || versions[0].CreatedByName != "dev" {
		t.Fatalf("v2 operator = %d/%q, want %d/dev", versions[0].CreatedBy, versions[0].CreatedByName, dev.ID)
	}
	if versions[0].ResponseBody != `{"v":2}` || versions[1].ResponseBody != `{"v":1}` {
		t.Fatalf("snapshot bodies = %q/%q", versions[0].ResponseBody, versions[1].ResponseBody)
	}
	if versions[0].ResponseHeaders["X-Demo"] != "1" {
		t.Fatalf("snapshot headers = %+v, want X-Demo preserved", versions[0].ResponseHeaders)
	}
	if versions[0].CreatedAt.IsZero() {
		t.Fatal("version CreatedAt must be set")
	}

	// Rollback: publish v1 again.
	published, err := svc.PublishVersion(project.ID, created.ID, 1, 2, dev.ID, RoleDev)
	if err != nil {
		t.Fatalf("PublishVersion: %v", err)
	}
	if published.CurrentVersion != 1 || published.ResponseBody != `{"v":1}` {
		t.Fatalf("after rollback = v%d %q, want v1 %q", published.CurrentVersion, published.ResponseBody, `{"v":1}`)
	}

	// History stays immutable after rollback: still exactly 2 versions.
	versions, err = svc.ListVersions(project.ID, created.ID, dev.ID, RoleDev)
	if err != nil || len(versions) != 2 {
		t.Fatalf("versions after rollback = %d, %v; want 2", len(versions), err)
	}

	// Editing after a rollback allocates a fresh version number (no reuse).
	again, err := svc.Update(project.ID, created.ID, dev.ID, RoleDev, dto.EndpointRequest{
		Path: "/api/users/:id", Method: "GET", StatusCode: 200, ResponseBody: `{"v":3}`,
	})
	if err != nil {
		t.Fatalf("Update after rollback: %v", err)
	}
	if again.CurrentVersion != 3 {
		t.Fatalf("CurrentVersion after rollback edit = %d, want 3", again.CurrentVersion)
	}
}

func TestEndpointPublishConflict(t *testing.T) {
	db := newTestDB(t)
	projects := repository.NewProjectRepository(db)
	svc := newEndpointService(db)
	dev := createUser(t, db, "dev", RoleDev)
	project, _ := NewProjectService(projects, discardLogger()).Create("p", "d", dev.ID, "http://localhost:3119")
	created, err := svc.Create(project.ID, dev.ID, RoleDev, dto.EndpointRequest{
		Path: "/a", Method: "GET", StatusCode: 200, ResponseBody: `{"v":1}`,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := svc.Update(project.ID, created.ID, dev.ID, RoleDev, dto.EndpointRequest{
		Path: "/a", Method: "GET", StatusCode: 200, ResponseBody: `{"v":2}`,
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	// Two publishers both saw current=2; only the first publish may succeed.
	if _, err := svc.PublishVersion(project.ID, created.ID, 1, 2, dev.ID, RoleDev); err != nil {
		t.Fatalf("first PublishVersion: %v", err)
	}
	_, err = svc.PublishVersion(project.ID, created.ID, 2, 2, dev.ID, RoleDev)
	requireConflict(t, err)

	// The failed publish must not have moved the current version.
	current, err := svc.Get(project.ID, created.ID, dev.ID, RoleDev)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if current.CurrentVersion != 1 || current.ResponseBody != `{"v":1}` {
		t.Fatalf("after failed publish = v%d %q, want v1 %q", current.CurrentVersion, current.ResponseBody, `{"v":1}`)
	}

	// Publishing a version that does not exist is a plain not-found.
	if _, err := svc.PublishVersion(project.ID, created.ID, 99, 1, dev.ID, RoleDev); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("publish missing version err = %v, want ErrNotFound", err)
	}
}

func TestEndpointUpdateConflict(t *testing.T) {
	db := newTestDB(t)
	projects := repository.NewProjectRepository(db)
	svc := newEndpointService(db)
	dev := createUser(t, db, "dev", RoleDev)
	project, _ := NewProjectService(projects, discardLogger()).Create("p", "d", dev.ID, "http://localhost:3119")
	created, err := svc.Create(project.ID, dev.ID, RoleDev, dto.EndpointRequest{
		Path: "/a", Method: "GET", StatusCode: 200, ResponseBody: `{"v":1}`,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// An edit based on the current version succeeds.
	if _, err := svc.Update(project.ID, created.ID, dev.ID, RoleDev, dto.EndpointRequest{
		Path: "/a", Method: "GET", StatusCode: 200, ResponseBody: `{"v":2}`, BaseVersion: intPtr(1),
	}); err != nil {
		t.Fatalf("Update with matching baseVersion: %v", err)
	}

	// A stale editor (still on v1) is rejected and changes nothing.
	_, err = svc.Update(project.ID, created.ID, dev.ID, RoleDev, dto.EndpointRequest{
		Path: "/a", Method: "GET", StatusCode: 200, ResponseBody: `{"v":stale}`, BaseVersion: intPtr(1),
	})
	requireConflict(t, err)

	current, err := svc.Get(project.ID, created.ID, dev.ID, RoleDev)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if current.CurrentVersion != 2 || current.ResponseBody != `{"v":2}` {
		t.Fatalf("after stale edit = v%d %q, want v2 %q", current.CurrentVersion, current.ResponseBody, `{"v":2}`)
	}
	versions, err := svc.ListVersions(project.ID, created.ID, dev.ID, RoleDev)
	if err != nil || len(versions) != 2 {
		t.Fatalf("versions after stale edit = %d, %v; want 2", len(versions), err)
	}
}

func TestLegacyEndpointEditBackfillsVersions(t *testing.T) {
	db := newTestDB(t)
	projects := repository.NewProjectRepository(db)
	svc := newEndpointService(db)
	dev := createUser(t, db, "dev", RoleDev)
	project, _ := NewProjectService(projects, discardLogger()).Create("p", "d", dev.ID, "http://localhost:3119")

	// Simulate a pre-versioning endpoint: no snapshots, current_version = 0.
	legacy := &model.MockAPI{
		ProjectID: project.ID, Path: "/legacy", Method: "GET", StatusCode: 200, ResponseBody: `{"old":true}`,
	}
	if err := db.Create(legacy).Error; err != nil {
		t.Fatalf("create legacy endpoint: %v", err)
	}

	// Listing versions of a legacy endpoint is empty, not an error.
	versions, err := svc.ListVersions(project.ID, legacy.ID, dev.ID, RoleDev)
	if err != nil || len(versions) != 0 {
		t.Fatalf("legacy ListVersions = %d, %v; want 0", len(versions), err)
	}

	// Editing must keep working and backfill the pre-edit content as v1.
	updated, err := svc.Update(project.ID, legacy.ID, dev.ID, RoleDev, dto.EndpointRequest{
		Path: "/legacy", Method: "GET", StatusCode: 200, ResponseBody: `{"new":true}`,
	})
	if err != nil {
		t.Fatalf("Update legacy: %v", err)
	}
	if updated.CurrentVersion != 2 {
		t.Fatalf("legacy CurrentVersion after edit = %d, want 2", updated.CurrentVersion)
	}
	versions, err = svc.ListVersions(project.ID, legacy.ID, dev.ID, RoleDev)
	if err != nil {
		t.Fatalf("ListVersions: %v", err)
	}
	if len(versions) != 2 || versions[0].Version != 2 || versions[1].Version != 1 {
		t.Fatalf("legacy versions = %+v", versions)
	}
	if versions[1].ResponseBody != `{"old":true}` {
		t.Fatalf("backfilled v1 body = %q, want old content", versions[1].ResponseBody)
	}
	if versions[1].CreatedByName != "dev" {
		t.Fatalf("backfilled v1 operator = %q, want dev", versions[1].CreatedByName)
	}
}

func TestMockEngineRecordsEffectiveVersion(t *testing.T) {
	db := newTestDB(t)
	projects := repository.NewProjectRepository(db)
	svc := newEndpointService(db)
	engine := NewMockEngine(repository.NewEndpointRepository(db), repository.NewRequestLogRepository(db), discardLogger())
	dev := createUser(t, db, "dev", RoleDev)
	project, _ := NewProjectService(projects, discardLogger()).Create("p", "d", dev.ID, "http://localhost:3119")
	created, err := svc.Create(project.ID, dev.ID, RoleDev, dto.EndpointRequest{
		Path: "/api/users/:id", Method: "GET", StatusCode: 200, ResponseBody: `{"v":1}`,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	lastLogVersion := func() int {
		t.Helper()
		var entry model.RequestLog
		if err := db.Order("id DESC").First(&entry).Error; err != nil {
			t.Fatalf("load last request log: %v", err)
		}
		return entry.APIVersion
	}

	if _, err := engine.Handle(project.ID, "GET", "/api/users/7", nil, nil, nil); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if v := lastLogVersion(); v != 1 {
		t.Fatalf("logged version = %d, want 1", v)
	}

	// After an edit the mock engine serves and logs the new version.
	if _, err := svc.Update(project.ID, created.ID, dev.ID, RoleDev, dto.EndpointRequest{
		Path: "/api/users/:id", Method: "GET", StatusCode: 200, ResponseBody: `{"v":2}`,
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	result, err := engine.Handle(project.ID, "GET", "/api/users/7", nil, nil, nil)
	if err != nil {
		t.Fatalf("Handle after edit: %v", err)
	}
	if result.Body != `{"v":2}` {
		t.Fatalf("body after edit = %q, want v2 content", result.Body)
	}
	if got := result.Headers["X-Mock-API-Version"]; got != "2" {
		t.Fatalf("X-Mock-API-Version = %q, want 2", got)
	}
	if v := lastLogVersion(); v != 2 {
		t.Fatalf("logged version after edit = %d, want 2", v)
	}

	// After a rollback the engine serves and logs the rolled-back version.
	if _, err := svc.PublishVersion(project.ID, created.ID, 1, 2, dev.ID, RoleDev); err != nil {
		t.Fatalf("PublishVersion: %v", err)
	}
	result, err = engine.Handle(project.ID, "GET", "/api/users/7", nil, nil, nil)
	if err != nil {
		t.Fatalf("Handle after rollback: %v", err)
	}
	if result.Body != `{"v":1}` {
		t.Fatalf("body after rollback = %q, want v1 content", result.Body)
	}
	if v := lastLogVersion(); v != 1 {
		t.Fatalf("logged version after rollback = %d, want 1", v)
	}
}

func TestEndpointDeleteRemovesVersions(t *testing.T) {
	db := newTestDB(t)
	projects := repository.NewProjectRepository(db)
	svc := newEndpointService(db)
	dev := createUser(t, db, "dev", RoleDev)
	project, _ := NewProjectService(projects, discardLogger()).Create("p", "d", dev.ID, "http://localhost:3119")
	created, err := svc.Create(project.ID, dev.ID, RoleDev, dto.EndpointRequest{
		Path: "/a", Method: "GET", StatusCode: 200, ResponseBody: `{}`,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.Delete(project.ID, created.ID, dev.ID, RoleDev); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	var count int64
	if err := db.Model(&model.MockAPIVersion{}).Where("api_id = ?", created.ID).Count(&count).Error; err != nil {
		t.Fatalf("count versions: %v", err)
	}
	if count != 0 {
		t.Fatalf("versions left after delete = %d, want 0", count)
	}
}
