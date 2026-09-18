package service

import (
	"github.com/mockhub/mockhub/internal/constants"
	"github.com/mockhub/mockhub/internal/model"
	"github.com/mockhub/mockhub/internal/repository"
)

// checkProjectAccess loads a project and verifies that the caller is either
// an administrator or the owner of the project. It is used by every service
// that exposes project-scoped resources.
func checkProjectAccess(projects *repository.ProjectRepository, projectID, userID uint, role string) (*model.Project, error) {
	p, err := projects.FindByID(projectID)
	if err != nil {
		return nil, err
	}
	if role != RoleLead && p.UserID != userID {
		return nil, constants.ErrForbidden
	}
	return p, nil
}
