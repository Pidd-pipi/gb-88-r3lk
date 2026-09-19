package dto

import "github.com/mockhub/mockhub/internal/model"

// EndpointRequest is the payload for creating/updating a mock endpoint.
// BaseVersion is the version the editor started from; when provided, the
// update is rejected with a conflict if the endpoint has moved on since.
type EndpointRequest struct {
	Path            string                `json:"path" validate:"required,startswith=/"`
	Method          string                `json:"method" validate:"required,oneof=GET POST PUT PATCH DELETE OPTIONS HEAD"`
	StatusCode      int                   `json:"statusCode" validate:"required,min=100,max=599"`
	ResponseBody    string                `json:"responseBody"`
	ResponseHeaders map[string]string     `json:"responseHeaders"`
	Delay           int                   `json:"delay" validate:"min=0,max=60000"`
	Conditions      []model.ConditionRule `json:"conditions"`
	BaseVersion     *int                  `json:"baseVersion" validate:"omitempty,min=0"`
}

// PublishVersionRequest is the payload for publishing (rolling back to) an
// immutable endpoint version. ExpectedVersion is the compare-and-swap token:
// the publish only succeeds when the endpoint's current version still equals
// it, so concurrent publishers cannot clobber each other.
type PublishVersionRequest struct {
	ExpectedVersion int `json:"expectedVersion" validate:"required,min=1"`
}

// SwaggerImportRequest carries an OpenAPI 2.0/3.0 JSON document for import.
type SwaggerImportRequest struct {
	Document any `json:"document" validate:"required"`
}
