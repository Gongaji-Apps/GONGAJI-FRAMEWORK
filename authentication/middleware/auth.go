package middleware

import (
	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/contextx"
	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"
	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/response"
	"github.com/gin-gonic/gin"
)

// Auth adalah middleware utama yang mendelegasikan ke strategy
func Auth(strategies ...AuthStrategy) gin.HandlerFunc {
	return func(c *gin.Context) {

		for _, s := range strategies {
			if !s.CanHandle(c) {
				continue
			}

			// Setiap strategy ekstrak token dengan caranya sendiri
			rawToken, err := s.ExtractToken(c)
			if err != nil {
				response.Error(c, err)
				c.Abort()
				return
			}

			claims, err := s.Authenticate(c.Request.Context(), rawToken)
			if err != nil {
				response.Error(c, err)
				c.Abort()
				return
			}

			ctx := c.Request.Context()

			ctx = contextx.WithSubjectUUID(ctx, claims.SubjectUUID)

			c.Set("subject_uuid", claims.SubjectUUID)
			c.Set("role_code", claims.Role)
			c.Set("permission_codes", claims.PermissionCodes)
			c.Set("auth_type", s.Name())

			for k, v := range claims.Extra {
				c.Set(k, v)
			}

			c.Request = c.Request.WithContext(ctx)
			c.Next()
			return
		}

		response.Error(c, errors.NewUnauthorized("[Unauthorized] Metode autentikasi tidak dikenali."))
		c.Abort()
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
