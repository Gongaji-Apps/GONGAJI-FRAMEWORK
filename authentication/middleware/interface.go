package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
)

type AuthClaims struct {
	SubjectUUID     string
	Role            string
	PermissionCodes map[string]bool
	Extra           map[string]any
}

type AuthStrategy interface {
	Name() string
	CanHandle(ctx *gin.Context) bool
	ExtractToken(ctx *gin.Context) (string, error)
	Authenticate(ctx context.Context, rawToken string) (*AuthClaims, error)
}
