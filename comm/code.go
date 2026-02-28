package comm

import "github.com/zjutjh/mygo/kit"

var CodeOK = kit.CodeOK

var (
	CodeServerError      = kit.CodeUnknownError
	CodeNotLoggedIn      = kit.CodeNotLoggedIn
	CodePermissionDenied = kit.CodePermissionDenied
	CodeParameterInvalid = kit.CodeParameterInvalid
	CodeDataNotFound     = kit.CodeDataNotFound
	CodeDataConflict     = kit.CodeDataConflict
)

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
	CodePostNotOwner          = kit.NewCode(30011, "您没有权限操作该发布记录")
	CodePostStatusInvalid     = kit.NewCode(30012, "当前状态不允许此操作")
	CodePostCannotModify      = kit.NewCode(30013, "该发布记录不能修改")
	CodeAdminPermissionDenied = kit.NewCode(30014, "仅管理员可操作")
	CodeReviewReasonRequired  = kit.NewCode(30015, "驳回理由必填")
	CodeReviewReasonTooLong   = kit.NewCode(30016, "驳回理由超长")
	CodeUserDisabled          = kit.NewCode(30017, "用户账号已被禁用")
	CodePublishLimitExceeded  = kit.NewCode(30018, "今日发布数量已达上限")
	CodeArchiveReasonRequired = kit.NewCode(30019, "归档处理方式必填")
	CodeArchiveNotExpired     = kit.NewCode(30020, "未超过认领时效，无法归档")
)

var (
	CodeLLMNotConfigured    = kit.NewCode(30100, "大模型服务未配置")
	CodeLLMCallFailed       = kit.NewCode(30101, "大模型调用失败")
	CodeEmbeddingFailed     = kit.NewCode(30102, "文本向量化失败")
	CodeMilvusNotConnected   = kit.NewCode(30103, "Milvus未连接")
	CodeVectorOperationFailed = kit.NewCode(30104, "向量操作失败")
	CodeAgentChatFailed      = kit.NewCode(30105, "AI对话失败")
	CodeSessionNotFound      = kit.NewCode(30106, "会话不存在")
	CodeSessionAccessDenied  = kit.NewCode(30107, "无权访问该会话")
)
