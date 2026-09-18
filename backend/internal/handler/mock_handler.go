package handler

import (
	"io"
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/mockhub/mockhub/internal/constants"
	"github.com/mockhub/mockhub/internal/service"
	"github.com/mockhub/mockhub/internal/util"
)

// MockHandler serves the public mock endpoints at /mock/:projectId/*path.
type MockHandler struct {
	engine *service.MockEngine
	logger *slog.Logger
}

// NewMockHandler builds a MockHandler.
func NewMockHandler(engine *service.MockEngine, logger *slog.Logger) *MockHandler {
	return &MockHandler{engine: engine, logger: logger}
}

// Serve handles any method against the mock engine.
func (h *MockHandler) Serve(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("projectId"), 10, 64)
	if err != nil {
		util.Fail(c, constants.NewAppError(constants.CodeBadRequest, "无效的项目 ID"))
		return
	}
	path := c.Param("path")
	if path == "" {
		path = "/"
	}
	method := c.Request.Method

	query := map[string]string{}
	for k, vs := range c.Request.URL.Query() {
		if len(vs) > 0 {
			query[k] = vs[0]
		}
	}
	headers := service.RequestHeaders(c.Request.Header)

	var body any
	if c.Request.Body != nil {
		raw, err := io.ReadAll(c.Request.Body)
		if err == nil {
			body = service.JSONBody(raw)
		}
	}

	result, err := h.engine.Handle(uint(projectID), method, path, query, headers, body)
	if err != nil {
		if service.IsNotFound(err) {
			util.Fail(c, constants.NewAppError(constants.CodeNotFound, "未找到匹配的 Mock 接口"))
			return
		}
		util.Fail(c, err)
		return
	}

	for k, v := range result.Headers {
		c.Header(k, v)
	}
	c.Data(result.StatusCode, "application/json; charset=utf-8", []byte(result.Body))
}
