package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/mockhub/mockhub/internal/constants"
	"github.com/mockhub/mockhub/internal/util"
)

// parseID extracts :projectId or :id from the request path.
func parseID(c *gin.Context) (uint, bool) {
	idStr := c.Param("projectId")
	if idStr == "" {
		idStr = c.Param("id")
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		util.Fail(c, constants.NewAppError(constants.CodeBadRequest, "无效的 ID"))
		return 0, false
	}
	return uint(id), true
}

// parseIDParam extracts a named uint path parameter.
func parseIDParam(c *gin.Context, name string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil {
		util.Fail(c, constants.NewAppError(constants.CodeBadRequest, "无效的 ID"))
		return 0, false
	}
	return uint(id), true
}

// parseVersion extracts the :version path parameter as a positive int.
func parseVersion(c *gin.Context) (int, bool) {
	n, err := strconv.Atoi(c.Param("version"))
	if err != nil || n < 1 {
		util.Fail(c, constants.NewAppError(constants.CodeBadRequest, "无效的版本号"))
		return 0, false
	}
	return n, true
}
