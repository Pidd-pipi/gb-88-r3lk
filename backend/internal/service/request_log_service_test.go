package service

import (
	"errors"
	"testing"

	"github.com/mockhub/mockhub/internal/constants"
	"github.com/mockhub/mockhub/internal/model"
	"github.com/mockhub/mockhub/internal/repository"
)

func TestRequestLogServiceListAndClear(t *testing.T) {
	db := newTestDB(t)
	projects := repository.NewProjectRepository(db)
	logs := repository.NewRequestLogRepository(db)
	projectSvc := NewProjectService(projects, discardLogger())
	svc := NewRequestLogService(projects, logs, discardLogger())

	dev := createUser(t, db, "dev", RoleDev)
	other := createUser(t, db, "other", RoleDev)
	project, err := projectSvc.Create("p", "d", dev.ID, "http://localhost:3119")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	for i := 1; i <= 3; i++ {
		if err := logs.Create(&model.RequestLog{
			ProjectID: project.ID, Method: "GET", Path: "/api/users",
			ResponseStatus: 200, ResponseBody: map[string]any{"i": i},
		}); err != nil {
			t.Fatalf("create log: %v", err)
		}
	}

	t.Run("owner can list", func(t *testing.T) {
		items, total, err := svc.List(project.ID, dev.ID, RoleDev, 1, 2)
		if err != nil || total != 3 || len(items) != 2 {
			t.Fatalf("List = %d/%d, %v; want total 3 and page size 2", len(items), total, err)
		}
	})

	t.Run("page size capped", func(t *testing.T) {
		items, _, err := svc.List(project.ID, dev.ID, RoleDev, 1, 999)
		if err != nil || len(items) != 3 {
			t.Fatalf("List capped = %d, %v", len(items), err)
		}
	})

	t.Run("other dev forbidden", func(t *testing.T) {
		_, _, err := svc.List(project.ID, other.ID, RoleDev, 1, 50)
		if !errors.Is(err, constants.ErrForbidden) {
			t.Fatalf("List err = %v, want ErrForbidden", err)
		}
	})

	t.Run("clear owner then empty", func(t *testing.T) {
		if err := svc.Clear(project.ID, dev.ID, RoleDev); err != nil {
			t.Fatalf("Clear: %v", err)
		}
		items, total, err := svc.List(project.ID, dev.ID, RoleDev, 1, 50)
		if err != nil || total != 0 || len(items) != 0 {
			t.Fatalf("List after clear = %d/%d, %v", len(items), total, err)
		}
	})
}
