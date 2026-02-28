package tools

import (
	"context"
)

type ToolContext struct {
	UserID   int64
	UserType string
}

type toolContextKey struct{}

func WithToolContext(ctx context.Context, toolCtx *ToolContext) context.Context {
	return context.WithValue(ctx, toolContextKey{}, toolCtx)
}

func GetToolContext(ctx context.Context) *ToolContext {
	if v := ctx.Value(toolContextKey{}); v != nil {
		return v.(*ToolContext)
	}
	return nil
}
