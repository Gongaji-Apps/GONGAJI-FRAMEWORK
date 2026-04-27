package middleware

import (
	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"
	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/response"
	"github.com/gin-gonic/gin"
)

// Auth adalah middleware utama yang mendelegasikan ke strategy
func Auth(strategies ...AuthStrategy) gin.HandlerFunc {
	return func(ctx *gin.Context) {

		for _, s := range strategies {
			if !s.CanHandle(ctx) {
				continue
			}

			// Setiap strategy ekstrak token dengan caranya sendiri
			rawToken, err := s.ExtractToken(ctx)
			if err != nil {
				response.Error(ctx, err)
				ctx.Abort()
				return
			}

			claims, err := s.Authenticate(ctx.Request.Context(), rawToken)
			if err != nil {
				response.Error(ctx, err)
				ctx.Abort()
				return
			}

			ctx.Set("subject_uuid", claims.SubjectUUID)
			ctx.Set("role_code", claims.Role)
			ctx.Set("permission_codes", claims.PermissionCodes)
			ctx.Set("auth_type", s.Name())
			for k, v := range claims.Extra {
				ctx.Set(k, v)
			}

			ctx.Next()
			return
		}

		response.Error(ctx, errors.NewUnauthorized("[Unauthorized] Metode autentikasi tidak dikenali."))
		ctx.Abort()
	}
}

func RequirePermission(value string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		permission_codes, exists := ctx.Get("permission_codes")

		if !exists {
			ctx.AbortWithStatusJSON(403, gin.H{"error": "forbidden"})
			return
		}

		permission_code, ok := permission_codes.(map[string]bool)
		if !ok || !permission_code[value] {
			ctx.AbortWithStatusJSON(403, gin.H{"error": "insufficient permissions"})
			return
		}
		ctx.Next()
	}
}
