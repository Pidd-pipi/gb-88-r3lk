package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/mockhub/mockhub/internal/model"
	"github.com/mockhub/mockhub/internal/repository"
	"github.com/mockhub/mockhub/internal/util"
)

// MockResult is the outcome of a mock request.
type MockResult struct {
	StatusCode   int
	Body         string
	Headers      map[string]string
	MatchedAPIID uint
	APIVersion   int
}

// MockEngine renders dynamic mock responses and records request logs.
type MockEngine struct {
	endpoints *repository.EndpointRepository
	logs      *repository.RequestLogRepository
	logger    *slog.Logger
}

// NewMockEngine builds a MockEngine.
func NewMockEngine(endpoints *repository.EndpointRepository, logs *repository.RequestLogRepository, logger *slog.Logger) *MockEngine {
	return &MockEngine{endpoints: endpoints, logs: logs, logger: logger}
}

// Handle processes a mock request.
func (e *MockEngine) Handle(projectID uint, method, path string, query map[string]string, headers map[string]string, body any) (*MockResult, error) {
	endpoint, params, err := e.endpoints.Match(projectID, method, path)
	if err != nil {
		return nil, err
	}

	var status int
	var responseBody string

	// Conditional response evaluation against query params and body fields.
	if len(endpoint.Conditions) > 0 {
		ctx := map[string]string{}
		for k, v := range query {
			ctx[k] = v
		}
		if bodyMap, ok := body.(map[string]any); ok {
			for k, v := range bodyMap {
				ctx[k] = fmt.Sprintf("%v", v)
			}
		}
		if cond := e.matchCondition(endpoint.Conditions, ctx); cond != nil {
			status = cond.StatusCode
			if status == 0 {
				status = endpoint.StatusCode
			}
			rendered, err := util.RenderTemplate(cond.ResponseBody)
			if err != nil {
				return nil, fmt.Errorf("render condition body: %w", err)
			}
			responseBody = rendered
		}
	}

	if status == 0 {
		status = endpoint.StatusCode
		// Replace path params inside the template ({{path.id}}).
		tpl := endpoint.ResponseBody
		for key, val := range params {
			tpl = strings.ReplaceAll(tpl, "{{path."+key+"}}", val)
		}
		// Replace query params inside the template ({{query.xxx}}).
		for key, val := range query {
			tpl = strings.ReplaceAll(tpl, "{{query."+key+"}}", val)
		}
		rendered, err := util.RenderTemplate(tpl)
		if err != nil {
			return nil, fmt.Errorf("render response body: %w", err)
		}
		responseBody = rendered
	}

	// Simulate configured latency.
	if endpoint.Delay > 0 {
		time.Sleep(time.Duration(endpoint.Delay) * time.Millisecond)
	}

	// Record the request log with the version that actually served it.
	logEntry := &model.RequestLog{
		ProjectID:      projectID,
		APIID:          endpoint.ID,
		APIVersion:     endpoint.CurrentVersion,
		Method:         method,
		Path:           path,
		Headers:        headers,
		Body:           body,
		Query:          query,
		ResponseStatus: status,
		ResponseBody:   responseBody,
	}
	if err := e.logs.Create(logEntry); err != nil {
		e.logger.Warn("record request log failed", "error", err)
	}

	respHeaders := map[string]string{"Content-Type": "application/json; charset=utf-8"}
	for k, v := range endpoint.ResponseHeaders {
		respHeaders[k] = v
	}
	return &MockResult{
		StatusCode:   status,
		Body:         responseBody,
		Headers:      respHeaders,
		MatchedAPIID: endpoint.ID,
		APIVersion:   endpoint.CurrentVersion,
	}, nil
}

func (e *MockEngine) matchCondition(conds []model.ConditionRule, ctx map[string]string) *model.ConditionRule {
	for i := range conds {
		cond := &conds[i]
		if cond.Field == "" {
			continue
		}
		actual, ok := ctx[cond.Field]
		if !ok {
			continue
		}
		switch cond.Operator {
		case "equals":
			if actual == cond.Value {
				return cond
			}
		case "contains":
			if strings.Contains(actual, cond.Value) {
				return cond
			}
		case "startsWith":
			if strings.HasPrefix(actual, cond.Value) {
				return cond
			}
		case "endsWith":
			if strings.HasSuffix(actual, cond.Value) {
				return cond
			}
		}
	}
	return nil
}

// JSONBody parses an incoming JSON body, tolerating empty input.
func JSONBody(raw []byte) any {
	if len(raw) == 0 {
		return nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	return v
}

// RequestHeaders converts an http.Header into a map for storage. Credential
// headers are intentionally dropped so secrets never end up in request logs.
func RequestHeaders(h http.Header) map[string]string {
	out := map[string]string{}
	for k, vs := range h {
		if len(vs) == 0 {
			continue
		}
		key := http.CanonicalHeaderKey(k)
		if key == "Authorization" || key == "Cookie" || key == "X-Api-Key" {
			continue
		}
		out[k] = strings.Join(vs, ", ")
	}
	return out
}

// IsNotFound reports a mock miss.
func IsNotFound(err error) bool {
	return errors.Is(err, repository.ErrNotFound)
}
