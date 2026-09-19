package service

import (
	"errors"
	"testing"

	"github.com/mockhub/mockhub/internal/constants"
	"github.com/mockhub/mockhub/internal/dto"
	"github.com/mockhub/mockhub/internal/repository"
)

func TestEndpointServiceCRUDAndAccess(t *testing.T) {
	db := newTestDB(t)
	projects := repository.NewProjectRepository(db)
	svc := newEndpointService(db)

	dev := createUser(t, db, "dev", RoleDev)
	other := createUser(t, db, "other", RoleDev)
	lead := createUser(t, db, "lead", RoleLead)
	project, err := NewProjectService(projects, discardLogger()).Create("p", "d", dev.ID, "http://localhost:3119")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	req := dto.EndpointRequest{
		Path:            "/api/users/:id",
		Method:          "GET",
		StatusCode:      200,
		ResponseBody:    `{"ok":true}`,
		ResponseHeaders: map[string]string{"X-Demo": "1"},
	}
	created, err := svc.Create(project.ID, dev.ID, RoleDev, req)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == 0 || created.ResponseHeaders["X-Demo"] != "1" {
		t.Fatalf("Create = %+v", created)
	}

	t.Run("owner list", func(t *testing.T) {
		apis, err := svc.List(project.ID, dev.ID, RoleDev)
		if err != nil || len(apis) != 1 {
			t.Fatalf("List = %d, %v; want 1 api", len(apis), err)
		}
	})

	t.Run("lead list", func(t *testing.T) {
		apis, err := svc.List(project.ID, lead.ID, RoleLead)
		if err != nil || len(apis) != 1 {
			t.Fatalf("List lead = %d, %v", len(apis), err)
		}
	})

	t.Run("other dev forbidden", func(t *testing.T) {
		_, err := svc.List(project.ID, other.ID, RoleDev)
		if !errors.Is(err, constants.ErrForbidden) {
			t.Fatalf("List err = %v, want ErrForbidden", err)
		}
	})

	t.Run("update", func(t *testing.T) {
		updated, err := svc.Update(project.ID, created.ID, dev.ID, RoleDev, dto.EndpointRequest{
			Path: "/api/users/:id", Method: "PUT", StatusCode: 200, ResponseBody: `{"updated":true}`,
		})
		if err != nil || updated.Method != "PUT" {
			t.Fatalf("Update = %+v, %v", updated, err)
		}
	})

	t.Run("delete then missing", func(t *testing.T) {
		if err := svc.Delete(project.ID, created.ID, dev.ID, RoleDev); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		if _, err := svc.Get(project.ID, created.ID, dev.ID, RoleDev); !errors.Is(err, repository.ErrNotFound) {
			t.Fatalf("Get after delete err = %v, want ErrNotFound", err)
		}
	})
}

func TestEndpointServiceImportOpenAPI(t *testing.T) {
	db := newTestDB(t)
	projects := repository.NewProjectRepository(db)
	svc := newEndpointService(db)
	dev := createUser(t, db, "dev", RoleDev)
	project, _ := NewProjectService(projects, discardLogger()).Create("p", "d", dev.ID, "http://localhost:3119")

	paths := map[string]any{}
	paths["/api/orders"] = map[string]any{
		"get": map[string]any{
			"responses": map[string]any{
				"200": map[string]any{"content": map[string]any{"application/json": map[string]any{"example": map[string]any{"code": 0}}}},
			},
		},
		"post": map[string]any{
			"responses": map[string]any{"201": map[string]any{"content": map[string]any{"application/json": map[string]any{"example": map[string]any{"id": 1}}}}},
		},
	}
	doc := map[string]any{"openapi": "3.0.0", "paths": paths}
	created, err := svc.ImportOpenAPI(project.ID, dev.ID, RoleDev, doc)
	if err != nil {
		t.Fatalf("ImportOpenAPI: %v", err)
	}
	if created != 2 {
		t.Fatalf("ImportOpenAPI created = %d, want 2", created)
	}
}
