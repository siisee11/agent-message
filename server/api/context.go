package api

import (
	"context"

	"agent-message/server/models"
)

type contextKey string

const (
	contextKeyUser  contextKey = "auth_user"
	contextKeyToken contextKey = "auth_token"
)

func contextWithAuth(ctx context.Context, user models.User, token string) context.Context {
	ctx = context.WithValue(ctx, contextKeyUser, user)
	ctx = context.WithValue(ctx, contextKeyToken, token)
	return ctx
}

func userFromContext(ctx context.Context) (models.User, bool) {
	user, ok := ctx.Value(contextKeyUser).(models.User)
	return user, ok
}

func tokenFromContext(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(contextKeyToken).(string)
	return token, ok
}
