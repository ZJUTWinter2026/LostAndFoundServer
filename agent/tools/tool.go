package tools

import (
	"context"
)

// SystemConfig 保存从数据库读取的系统配置，随每次请求注入 MessageModifier
type SystemConfig struct {
	FeedbackTypes     []string
	ItemTypes         []string
	ClaimValidityDays int
	PublishLimit      int
}

type ToolContext struct {
	UserID       int64
	SystemConfig *SystemConfig
}

type toolContextKey struct{}

// WithToolContext 将工具上下文写入请求上下文。
func WithToolContext(ctx context.Context, toolCtx *ToolContext) context.Context {
	return context.WithValue(ctx, toolContextKey{}, toolCtx)
}

// GetToolContext 从请求上下文中读取工具上下文。
func GetToolContext(ctx context.Context) *ToolContext {
	if v := ctx.Value(toolContextKey{}); v != nil {
		return v.(*ToolContext)
	}
	return nil
}
