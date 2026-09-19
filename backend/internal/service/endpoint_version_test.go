package service

import (
	"errors"
	"testing"

	"gorm.io/gorm"

	"github.com/mockhub/mockhub/internal/dto"
	"github.com/mockhub/mockhub/internal/model"
	"github.com/mockhub/mockhub/internal/repository"
)

// newVersionTestStack builds the service plus every repository it needs.
func newVersionTestStack(t *testing.T) (*EndpointService, *gorm.DB) {
	t.Helper()
	db := newTestDB(t)
	svc := NewEndpointService(
		repository.NewProjectRepository(db),
		repository.NewEndpointRepository(db),
		repository.NewAPIVersionRepository(db),
		repository.NewUserRepository(db),
		discardLogger(),
	)
	return svc, db
}

func createVersionedProject(t *testing.T, db *gorm.DB, ownerID uint) *model.Project {
	t.Helper()
	project, err := NewProjectService(repository.NewProjectRepository(db), discardLogger()).
		Create("p", "d", ownerID, "http://localhost:3119")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	return project
}

func intPtr(n int) *int { return &n }

func TestEndpointVersionLifecycle(t *testing.T) {
	svc, db := newVersionTestStack(t)
	dev := createUser(t, db, "dev", RoleDev)
	project := createVersionedProject(t, db, dev.ID)

	created, err := svc.Create(project.ID, dev.ID, RoleDev, dto.EndpointRequest{
		Path: "/api/users", Method: "GET", StatusCode: 200, ResponseBody: `{"v":1}`,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.CurrentVersion != 1 {
		t.Fatalf("CurrentVersion = %d, want 1", created.CurrentVersion)
	}

	t.Run("edit appends immutable version with editor", func(t *testing.T) {
		updated, err := svc.Update(project.ID, created.ID, dev.ID, RoleDev, dto.EndpointRequest{
			Path: "/api/users", Method: "GET", StatusCode: 200, ResponseBody: `{"v":2}`,
			BaseVersion: intPtr(1),
		})
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if updated.CurrentVersion != 2 {
			t.Fatalf("CurrentVersion = %d, want 2", updated.CurrentVersion)
		}

		versions, err := svc.ListVersions(project.ID, created.ID, dev.ID, RoleDev)
		if err != nil || len(versions) != 2 {
			t.Fatalf("ListVersions = %d, %v; want 2 versions", len(versions), err)
		}
		// Newest first.
		if versions[0].Version != 2 || versions[1].Version != 1 {
			t.Fatalf("version order = [%d %d], want [2 1]", versions[0].Version, versions[1].Version)
		}
		if versions[0].EditorName != "dev" || versions[0].EditorID != dev.ID {
			t.Fatalf("editor = %q/%d, want dev/%d", versions[0].EditorName, versions[0].EditorID, dev.ID)
		}
		if versions[0].CreatedAt.IsZero() {
			t.Fatal("version CreatedAt must be recorded")
		}

		// The v1 snapshot keeps the original body: versions are immutable.
		v1, err := svc.GetVersion(project.ID, created.ID, 1, dev.ID, RoleDev)
		if err != nil {
			t.Fatalf("GetVersion: %v", err)
		}
		if v1.ResponseBody != `{"v":1}` {
			t.Fatalf("v1 body = %q, want {\"v\":1}", v1.ResponseBody)
		}
	})

	t.Run("publish rolls back to historical version", func(t *testing.T) {
		api, err := svc.PublishVersion(project.ID, created.ID, 1, dev.ID, RoleDev, 2)
		if err != nil {
			t.Fatalf("PublishVersion: %v", err)
		}
		if api.CurrentVersion != 1 || api.ResponseBody != `{"v":1}` {
			t.Fatalf("after rollback = v%d %q, want v1 {\"v\":1}", api.CurrentVersion, api.ResponseBody)
		}
		// History is preserved after rollback.
		versions, _ := svc.ListVersions(project.ID, created.ID, dev.ID, RoleDev)
		if len(versions) != 2 {
			t.Fatalf("versions after rollback = %d, want 2", len(versions))
		}
	})

	t.Run("publish live version is idempotent", func(t *testing.T) {
		api, err := svc.PublishVersion(project.ID, created.ID, 1, dev.ID, RoleDev, 999)
		if err != nil {
			t.Fatalf("idempotent publish: %v", err)
		}
		if api.CurrentVersion != 1 {
			t.Fatalf("CurrentVersion = %d, want 1", api.CurrentVersion)
		}
	})

	t.Run("publish missing version is not found", func(t *testing.T) {
		_, err := svc.PublishVersion(project.ID, created.ID, 42, dev.ID, RoleDev, 1)
		if !errors.Is(err, repository.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestEndpointPublishConcurrency(t *testing.T) {
	svc, db := newVersionTestStack(t)
	dev := createUser(t, db, "dev", RoleDev)
	project := createVersionedProject(t, db, dev.ID)

	created, err := svc.Create(project.ID, dev.ID, RoleDev, dto.EndpointRequest{
		Path: "/api/a", Method: "GET", StatusCode: 200, ResponseBody: `{"v":1}`,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := svc.Update(project.ID, created.ID, dev.ID, RoleDev, dto.EndpointRequest{
		Path: "/api/a", Method: "GET", StatusCode: 200, ResponseBody: `{"v":2}`,
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	// Current version is now 2. Two publishes race from the same base: the
	// first wins, the second must fail without changing the live version.
	if _, err := svc.PublishVersion(project.ID, created.ID, 1, dev.ID, RoleDev, 2); err != nil {
		t.Fatalf("first publish: %v", err)
	}
	if _, err := svc.PublishVersion(project.ID, created.ID, 2, dev.ID, RoleDev, 2); !errors.Is(err, repository.ErrVersionConflict) {
		t.Fatalf("second publish err = %v, want ErrVersionConflict", err)
	}

	api, err := svc.Get(project.ID, created.ID, dev.ID, RoleDev)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if api.CurrentVersion != 1 || api.ResponseBody != `{"v":1}` {
		t.Fatalf("live version = v%d %q, want v1 {\"v\":1} (loser must not change it)", api.CurrentVersion, api.ResponseBody)
	}
}

func TestEndpointUpdateConflict(t *testing.T) {
	svc, db := newVersionTestStack(t)
	dev := createUser(t, db, "dev", RoleDev)
	project := createVersionedProject(t, db, dev.ID)

	created, err := svc.Create(project.ID, dev.ID, RoleDev, dto.EndpointRequest{
		Path: "/api/a", Method: "GET", StatusCode: 200, ResponseBody: `{"v":1}`,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := svc.Update(project.ID, created.ID, dev.ID, RoleDev, dto.EndpointRequest{
		Path: "/api/a", Method: "GET", StatusCode: 200, ResponseBody: `{"v":2}`,
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	// A stale editor still on baseVersion=1 must be rejected and the live
	// version must stay exactly as it was.
	_, err = svc.Update(project.ID, created.ID, dev.ID, RoleDev, dto.EndpointRequest{
		Path: "/api/a", Method: "POST", StatusCode: 201, ResponseBody: `{"stale":true}`,
		BaseVersion: intPtr(1),
	})
	if !errors.Is(err, repository.ErrVersionConflict) {
		t.Fatalf("stale update err = %v, want ErrVersionConflict", err)
	}
	api, _ := svc.Get(project.ID, created.ID, dev.ID, RoleDev)
	if api.CurrentVersion != 2 || api.Method != "GET" || api.ResponseBody != `{"v":2}` {
		t.Fatalf("live version changed by failed update: %+v", api)
	}
	versions, _ := svc.ListVersions(project.ID, created.ID, dev.ID, RoleDev)
	if len(versions) != 2 {
		t.Fatalf("versions = %d, want 2 (failed update must not append)", len(versions))
	}
}

func TestLegacyEndpointWithoutVersionData(t *testing.T) {
	svc, db := newVersionTestStack(t)
	dev := createUser(t, db, "dev", RoleDev)
	project := createVersionedProject(t, db, dev.ID)

	// Simulate a pre-upgrade row: no version pointer, no version rows.
	legacy := &model.MockAPI{
		ProjectID: project.ID, Path: "/old", Method: "GET",
		StatusCode: 200, ResponseBody: `{"old":true}`,
	}
	if err := db.Create(legacy).Error; err != nil {
		t.Fatalf("create legacy endpoint: %v", err)
	}

	t.Run("edit works and creates first version", func(t *testing.T) {
		updated, err := svc.Update(project.ID, legacy.ID, dev.ID, RoleDev, dto.EndpointRequest{
			Path: "/old", Method: "GET", StatusCode: 200, ResponseBody: `{"old":false}`,
		})
		if err != nil {
			t.Fatalf("Update legacy: %v", err)
		}
		if updated.CurrentVersion != 1 {
			t.Fatalf("CurrentVersion = %d, want 1", updated.CurrentVersion)
		}
		versions, err := svc.ListVersions(project.ID, legacy.ID, dev.ID, RoleDev)
		if err != nil || len(versions) != 1 {
			t.Fatalf("ListVersions = %d, %v; want 1", len(versions), err)
		}
	})

	t.Run("backfill snapshots legacy rows", func(t *testing.T) {
		legacy2 := &model.MockAPI{
			ProjectID: project.ID, Path: "/older", Method: "POST",
			StatusCode: 201, ResponseBody: `{"older":true}`,
		}
		if err := db.Create(legacy2).Error; err != nil {
			t.Fatalf("create legacy endpoint 2: %v", err)
		}
		n, err := svc.BackfillLegacyVersions()
		if err != nil || n != 1 {
			t.Fatalf("BackfillLegacyVersions = %d, %v; want 1", n, err)
		}
		api, err := svc.Get(project.ID, legacy2.ID, dev.ID, RoleDev)
		if err != nil || api.CurrentVersion != 1 {
			t.Fatalf("backfilled endpoint = v%d, %v; want v1", api.CurrentVersion, err)
		}
		v1, err := svc.GetVersion(project.ID, legacy2.ID, 1, dev.ID, RoleDev)
		if err != nil {
			t.Fatalf("GetVersion: %v", err)
		}
		if v1.ResponseBody != `{"older":true}` || v1.StatusCode != 201 || v1.EditorName != "system" {
			t.Fatalf("backfilled snapshot = %+v", v1)
		}

		// Idempotent: a second run migrates nothing.
		n, err = svc.BackfillLegacyVersions()
		if err != nil || n != 0 {
			t.Fatalf("second backfill = %d, %v; want 0", n, err)
		}
	})
}

func TestMockEngineRecordsAPIVersion(t *testing.T) {
	svc, db := newVersionTestStack(t)
	dev := createUser(t, db, "dev", RoleDev)
	project := createVersionedProject(t, db, dev.ID)

	created, err := svc.Create(project.ID, dev.ID, RoleDev, dto.EndpointRequest{
		Path: "/api/users", Method: "GET", StatusCode: 200, ResponseBody: `{"ok":true}`,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	engine := NewMockEngine(repository.NewEndpointRepository(db), repository.NewRequestLogRepository(db), discardLogger())
	logs := repository.NewRequestLogRepository(db)

	result, err := engine.Handle(project.ID, "GET", "/api/users", nil, nil, nil)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if result.APIVersion != 1 {
		t.Fatalf("APIVersion = %d, want 1", result.APIVersion)
	}

	// Edit bumps the live version; the next request must record v2.
	if _, err := svc.Update(project.ID, created.ID, dev.ID, RoleDev, dto.EndpointRequest{
		Path: "/api/users", Method: "GET", StatusCode: 200, ResponseBody: `{"ok":2}`,
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	result, err = engine.Handle(project.ID, "GET", "/api/users", nil, nil, nil)
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if result.APIVersion != 2 {
		t.Fatalf("APIVersion after edit = %d, want 2", result.APIVersion)
	}

	entries, total, err := logs.ListByProject(project.ID, 1, 10)
	if err != nil || total != 2 {
		t.Fatalf("ListByProject = %d, %v; want 2 logs", total, err)
	}
	// Logs are newest first: v2 then v1, and they survive a fresh read
	// (i.e. the version is persisted, not computed).
	if entries[0].APIVersion != 2 || entries[1].APIVersion != 1 {
		t.Fatalf("logged versions = [%d %d], want [2 1]", entries[0].APIVersion, entries[1].APIVersion)
	}
}
