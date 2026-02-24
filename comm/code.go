package comm

import "github.com/zjutjh/mygo/kit"

var CodeOK = kit.NewCode(0, "成功")

// 系统错误码
var (
	CodeUnknownError           = kit.NewCode(10000, "未知错误")
	CodeThirdServiceError      = kit.NewCode(10001, "三方服务错误")
	CodeDatabaseError          = kit.NewCode(10002, "数据库错误")
	CodeRedisError             = kit.NewCode(10003, "Redis错误")
	CodeMiddlewareServiceError = kit.NewCode(10004, "中间件服务错误")
)

// 业务通用错误码
var (
	CodeNotLoggedIn        = kit.NewCode(20000, "用户未登录")
	CodeLoginExpired       = kit.NewCode(20001, "登录过期，请重新登录")
	CodePermissionDenied   = kit.NewCode(20002, "用户无权限")
	CodeParameterInvalid   = kit.NewCode(20003, "参数非法")
	CodeDataParseError     = kit.NewCode(20004, "数据解析异常")
	CodeDataNotFound       = kit.NewCode(20005, "数据不存在")
	CodeDataConflict       = kit.NewCode(20006, "数据冲突")
	CodeServiceMaintenance = kit.NewCode(20007, "系统维护中")
	CodeTooFrequently      = kit.NewCode(20008, "操作过于频繁/未获得锁")
)

// 业务错误码 从 30000 开始
var (
	CodeUserNotExist          = kit.NewCode(30000, "用户不存在")
	CodeHashError             = kit.NewCode(30001, "加密失败")
	CodePasswordError         = kit.NewCode(30002, "密码错误")
	CodeTokenError            = kit.NewCode(30003, "生成token失败")
	CodeClaimDuplicate        = kit.NewCode(30004, "已有待确认或已匹配的认领申请")
	CodeClaimAlreadyMatched   = kit.NewCode(30005, "该物品已有已匹配的认领")
	CodeClaimOwnItem          = kit.NewCode(30006, "不能认领自己发布的物品")
	CodeClaimStatusInvalid    = kit.NewCode(30007, "认领申请状态不允许此操作")
	CodeClaimNotFound         = kit.NewCode(30008, "认领申请不存在")
	CodeFeedbackTypeInvalid   = kit.NewCode(30009, "投诉类型无效")
	CodeFeedbackTypeOther     = kit.NewCode(30010, "投诉类型为其它时必须填写说明")
	CodePostNotOwner          = kit.NewCode(30011, "您没有权限操作该发布记录")
	CodePostStatusInvalid     = kit.NewCode(30012, "当前状态不允许此操作")
	CodePostCannotModify      = kit.NewCode(30013, "该发布记录不能修改")
	CodeAdminPermissionDenied = kit.NewCode(30014, "仅管理员可操作")
	CodeReviewReasonRequired  = kit.NewCode(30015, "驳回理由必填")
	CodeReviewReasonTooLong   = kit.NewCode(30016, "驳回理由超长")
)
