package constants

// User-facing messages used by handlers and services.
const (
	MsgOK              = "ok"
	MsgInvalidParams   = "请求参数格式错误"
	MsgValidationFail  = "参数校验失败"
	MsgUnauthorized    = "未登录或登录已过期"
	MsgForbidden       = "没有权限"
	MsgNotFound        = "资源不存在"
	MsgUsernameTaken   = "用户名已存在"
	MsgLoginFailed     = "用户名或密码错误"
	MsgInternalError   = "服务器内部错误"
	MsgTooManyRequests = "请求过于频繁，请稍后再试"
	MsgVersionConflict = "接口版本已被他人变更，请刷新后重试"
)
