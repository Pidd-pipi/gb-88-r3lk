package service

import (
	"errors"
	"fmt"
	"testing"

	"github.com/mockhub/mockhub/internal/constants"
	"github.com/mockhub/mockhub/internal/repository"
)

func TestProjectServiceCreateAndList(t *testing.T) {
	db := newTestDB(t)
	repo := repository.NewProjectRepository(db)
	svc := NewProjectService(repo, discardLogger())

	lead := createUser(t, db, "lead", RoleLead)
	dev := createUser(t, db, "dev", RoleDev)

	project, err := svc.Create("用户服务", "user service", dev.ID, "http://localhost:3119")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if project.ID == 0 {
		t.Fatal("Create did not assign an id")
	}
	if project.BaseURL != fmt.Sprintf("http://localhost:3119/mock/%d", project.ID) {
		t.Fatalf("BaseURL = %q, want computed mock base URL", project.BaseURL)
	}

	tests := []struct {
		name      string
		userID    uint
		role      string
		wantCount int
	}{
		{name: "owner sees own project", userID: dev.ID, role: RoleDev, wantCount: 1},
		{name: "lead sees every project", userID: lead.ID, role: RoleLead, wantCount: 1},
		{name: "other dev sees no project", userID: lead.ID, role: RoleDev, wantCount: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projects, err := svc.List(tt.userID, tt.role)
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(projects) != tt.wantCount {
				t.Fatalf("List count = %d, want %d", len(projects), tt.wantCount)
			}
		})
	}
}

func TestProjectServiceGetUpdateDelete(t *testing.T) {
	db := newTestDB(t)
	repo := repository.NewProjectRepository(db)
	svc := NewProjectService(repo, discardLogger())
	dev := createUser(t, db, "dev", RoleDev)
	other := createUser(t, db, "other", RoleDev)
	project, err := svc.Create("项目", "desc", dev.ID, "http://localhost:3119")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	t.Run("owner can read", func(t *testing.T) {
		got, err := svc.Get(project.ID, dev.ID, RoleDev)
		if err != nil || got.ID != project.ID {
			t.Fatalf("Get = %+v, %v", got, err)
		}
	})

	t.Run("other dev forbidden", func(t *testing.T) {
		_, err := svc.Get(project.ID, other.ID, RoleDev)
		if !errors.Is(err, constants.ErrForbidden) {
			t.Fatalf("Get err = %v, want ErrForbidden", err)
		}
	})

	t.Run("update owner", func(t *testing.T) {
		updated, err := svc.Update(project.ID, dev.ID, RoleDev, "新名称", "新描述")
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if updated.Name != "新名称" || updated.Description != "新描述" {
			t.Fatalf("Update = %+v", updated)
		}
	})

	t.Run("update forbidden", func(t *testing.T) {
		_, err := svc.Update(project.ID, other.ID, RoleDev, "x", "x")
		if !errors.Is(err, constants.ErrForbidden) {
			t.Fatalf("Update err = %v, want ErrForbidden", err)
		}
	})

	t.Run("delete owner", func(t *testing.T) {
		if err := svc.Delete(project.ID, dev.ID, RoleDev); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		if _, err := svc.Get(project.ID, dev.ID, RoleDev); !errors.Is(err, repository.ErrNotFound) {
			t.Fatalf("Get after delete err = %v, want ErrNotFound", err)
		}
	})
}
