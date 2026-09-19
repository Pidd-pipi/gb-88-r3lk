package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mockhub/mockhub/internal/constants"
	"github.com/mockhub/mockhub/internal/dto"
	"github.com/mockhub/mockhub/internal/model"
	"github.com/mockhub/mockhub/internal/repository"
)

// EndpointService manages mock endpoints, their immutable versions and
// OpenAPI imports.
type EndpointService struct {
	projects  *repository.ProjectRepository
	endpoints *repository.EndpointRepository
	versions  *repository.EndpointVersionRepository
	users     *repository.UserRepository
	logger    *slog.Logger
}

// NewEndpointService builds an EndpointService.
func NewEndpointService(
	projects *repository.ProjectRepository,
	endpoints *repository.EndpointRepository,
	versions *repository.EndpointVersionRepository,
	users *repository.UserRepository,
	logger *slog.Logger,
) *EndpointService {
	return &EndpointService{projects: projects, endpoints: endpoints, versions: versions, users: users, logger: logger}
}

// List returns endpoints of a project after checking access.
func (s *EndpointService) List(projectID, userID uint, role string) ([]model.MockAPI, error) {
	if _, err := s.checkAccess(projectID, userID, role); err != nil {
		return nil, err
	}
	return s.endpoints.ListByProject(projectID)
}

// Get loads a single endpoint after checking access.
func (s *EndpointService) Get(projectID, id, userID uint, role string) (*model.MockAPI, error) {
	if _, err := s.checkAccess(projectID, userID, role); err != nil {
		return nil, err
	}
	e, err := s.endpoints.FindByID(id)
	if err != nil {
		return nil, err
	}
	if e.ProjectID != projectID {
		return nil, repository.ErrNotFound
	}
	return e, nil
}

// Create adds an endpoint to a project and records its first immutable
// version (v1) stamped with the operator.
func (s *EndpointService) Create(projectID, userID uint, role string, req dto.EndpointRequest) (*model.MockAPI, error) {
	if _, err := s.checkAccess(projectID, userID, role); err != nil {
		return nil, err
	}
	editorName, err := s.editorName(userID)
	if err != nil {
		return nil, err
	}
	e := &model.MockAPI{
		ProjectID:       projectID,
		Path:            req.Path,
		Method:          req.Method,
		StatusCode:      req.StatusCode,
		ResponseBody:    req.ResponseBody,
		ResponseHeaders: req.ResponseHeaders,
		Delay:           req.Delay,
		Conditions:      req.Conditions,
	}
	if err := s.versions.CreateWithFirstVersion(e, userID, editorName); err != nil {
		return nil, err
	}
	s.logger.Info("endpoint created", "project_id", projectID, "endpoint_id", e.ID, "method", e.Method, "path", e.Path)
	return e, nil
}

// Update edits an endpoint after checking access. Every successful edit
// appends a new immutable version stamped with the operator and flips the
// endpoint's current version to it. When the request carries a baseVersion
// that no longer matches, or a concurrent writer wins the compare-and-swap,
// the edit fails with a conflict and the current version stays untouched.
// Endpoints created before versioning are backfilled transparently: their
// pre-edit content becomes v1.
func (s *EndpointService) Update(projectID, id, userID uint, role string, req dto.EndpointRequest) (*model.MockAPI, error) {
	prev, err := s.Get(projectID, id, userID, role)
	if err != nil {
		return nil, err
	}
	if req.BaseVersion != nil && *req.BaseVersion != prev.CurrentVersion {
		return nil, versionConflictError()
	}
	editorName, err := s.editorName(userID)
	if err != nil {
		return nil, err
	}
	next := &model.MockAPI{
		ID:              prev.ID,
		ProjectID:       prev.ProjectID,
		Path:            req.Path,
		Method:          req.Method,
		StatusCode:      req.StatusCode,
		ResponseBody:    req.ResponseBody,
		ResponseHeaders: req.ResponseHeaders,
		Delay:           req.Delay,
		Conditions:      req.Conditions,
		CreatedAt:       prev.CreatedAt,
	}
	// Fields the request left untouched (nil maps) keep their previous
	// content, mirroring the previous Save-based update semantics and
	// keeping the new version snapshot consistent with the endpoint row.
	if next.ResponseHeaders == nil {
		next.ResponseHeaders = prev.ResponseHeaders
	}
	if next.Conditions == nil {
		next.Conditions = prev.Conditions
	}
	if _, err := s.versions.ApplyEdit(prev, next, userID, editorName, prev.CurrentVersion); err != nil {
		if errors.Is(err, repository.ErrVersionConflict) {
			return nil, versionConflictError()
		}
		return nil, err
	}
	s.logger.Info("endpoint updated", "project_id", projectID, "endpoint_id", id, "version", next.CurrentVersion)
	return next, nil
}

// Delete removes an endpoint together with its version history.
func (s *EndpointService) Delete(projectID, id, userID uint, role string) error {
	if _, err := s.Get(projectID, id, userID, role); err != nil {
		return err
	}
	if err := s.versions.DeleteWithVersions(id); err != nil {
		return err
	}
	s.logger.Info("endpoint deleted", "project_id", projectID, "endpoint_id", id)
	return nil
}

// ListVersions returns the immutable version history of an endpoint, newest
// first. Endpoints predating versioning simply return an empty list.
func (s *EndpointService) ListVersions(projectID, id, userID uint, role string) ([]model.MockAPIVersion, error) {
	if _, err := s.Get(projectID, id, userID, role); err != nil {
		return nil, err
	}
	return s.versions.ListByAPI(id)
}

// PublishVersion rolls the endpoint back (or forward) to an existing
// immutable version. expectedVersion must equal the endpoint's current
// version, so exactly one of several concurrent publishers succeeds and the
// losers leave the current version untouched.
func (s *EndpointService) PublishVersion(projectID, id uint, version int, expectedVersion int, userID uint, role string) (*model.MockAPI, error) {
	if _, err := s.Get(projectID, id, userID, role); err != nil {
		return nil, err
	}
	v, err := s.versions.FindByAPIAndVersion(id, version)
	if err != nil {
		return nil, err
	}
	if err := s.versions.ApplyPublish(id, v, expectedVersion); err != nil {
		if errors.Is(err, repository.ErrVersionConflict) {
			return nil, versionConflictError()
		}
		return nil, err
	}
	s.logger.Info("endpoint version published", "project_id", projectID, "endpoint_id", id, "version", version, "user_id", userID)
	return s.endpoints.FindByID(id)
}

// ImportOpenAPI parses an OpenAPI 2.0/3.0 document and creates endpoints
// from every path + method pair that has a JSON example response.
func (s *EndpointService) ImportOpenAPI(projectID, userID uint, role string, doc any) (int, error) {
	if _, err := s.checkAccess(projectID, userID, role); err != nil {
		return 0, err
	}
	editorName, err := s.editorName(userID)
	if err != nil {
		return 0, err
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		return 0, constants.NewAppError(constants.CodeBadRequest, "无效的 OpenAPI 文档")
	}
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return 0, constants.NewAppError(constants.CodeBadRequest, "无效的 OpenAPI JSON")
	}
	paths, ok := parsed["paths"].(map[string]any)
	if !ok {
		return 0, constants.NewAppError(constants.CodeBadRequest, "文档缺少 paths 字段")
	}
	created := 0
	for path, methodsAny := range paths {
		methods, ok := methodsAny.(map[string]any)
		if !ok {
			continue
		}
		for method, opAny := range methods {
			method = upper(method)
			if !isHTTPMethod(method) {
				continue
			}
			op, ok := opAny.(map[string]any)
			if !ok {
				continue
			}
			body := s.exampleResponse(op)
			status := 200
			if responses, ok := op["responses"].(map[string]any); ok {
				for codeStr, respAny := range responses {
					code := atoiSafe(codeStr)
					if code >= 200 && code < 300 {
						status = code
						if resp, ok := respAny.(map[string]any); ok {
							if ex := exampleFromResponse(resp); ex != "" {
								body = ex
							}
						}
						break
					}
				}
			}
			e := &model.MockAPI{
				ProjectID:    projectID,
				Path:         path,
				Method:       method,
				StatusCode:   status,
				ResponseBody: body,
			}
			if err := s.versions.CreateWithFirstVersion(e, userID, editorName); err != nil {
				continue
			}
			created++
		}
	}
	s.logger.Info("openapi imported", "project_id", projectID, "created", created)
	return created, nil
}

func (s *EndpointService) checkAccess(projectID, userID uint, role string) (*model.Project, error) {
	return checkProjectAccess(s.projects, projectID, userID, role)
}

// editorName resolves the operator's username for version stamps.
func (s *EndpointService) editorName(userID uint) (string, error) {
	u, err := s.users.FindByID(userID)
	if err != nil {
		return "", fmt.Errorf("resolve version operator: %w", err)
	}
	return u.Username, nil
}

// versionConflictError builds the user-facing conflict error returned when a
// compare-and-swap on the current version loses a race.
func versionConflictError() error {
	return constants.NewAppError(constants.CodeConflict, constants.MsgVersionConflict)
}

func (s *EndpointService) exampleResponse(op map[string]any) string {
	if reqBody, ok := op["requestBody"].(map[string]any); ok {
		if content, ok := reqBody["content"].(map[string]any); ok {
			if js, ok := content["application/json"].(map[string]any); ok {
				if ex, ok := js["example"]; ok {
					b, _ := json.Marshal(ex)
					return string(b)
				}
			}
		}
	}
	return `{"code":0,"message":"ok"}`
}

func exampleFromResponse(resp map[string]any) string {
	if content, ok := resp["content"].(map[string]any); ok {
		if js, ok := content["application/json"].(map[string]any); ok {
			if ex, ok := js["example"]; ok {
				b, _ := json.Marshal(ex)
				return string(b)
			}
		}
	}
	return ""
}

func upper(s string) string {
	return strings.ToUpper(s)
}

func isHTTPMethod(s string) bool {
	switch s {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD":
		return true
	}
	return false
}

func atoiSafe(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}
