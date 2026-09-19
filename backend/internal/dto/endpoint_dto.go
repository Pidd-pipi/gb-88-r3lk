package dto

import "github.com/mockhub/mockhub/internal/model"

// EndpointRequest is the payload for creating/updating a mock endpoint.
// BaseVersion is the optimistic-concurrency token: when present it must equal
// the endpoint's current version, otherwise the update is rejected with a
// conflict and the live version stays untouched.
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

// PublishVersionRequest carries the optimistic-concurrency token for
// publishing a historical version. It is required so concurrent publishes of
// the same endpoint are mutually exclusive.
type PublishVersionRequest struct {
	BaseVersion *int `json:"baseVersion" validate:"required,min=0"`
}

// SwaggerImportRequest carries an OpenAPI 2.0/3.0 JSON document for import.
type SwaggerImportRequest struct {
	Document any `json:"document" validate:"required"`
}
