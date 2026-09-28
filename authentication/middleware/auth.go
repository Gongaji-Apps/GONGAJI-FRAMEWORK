package middleware

import (
	"context"

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
			if claims.SubjectUserID != nil {
				ctx = contextx.WithSubjectUserID(ctx, *claims.SubjectUserID)
			}
			ctx = contextx.WithSubjectFullName(ctx, claims.SubjectFullName)
			ctx = contextx.WithSubjectEmail(ctx, claims.SubjectEmail)
			ctx = contextx.WithSubjectTester(ctx, claims.SubjectTester)
			ctx = contextx.WithRoleCode(ctx, claims.Role)
			ctx = contextx.WithPermissionCodes(ctx, claims.PermissionCodes)
			ctx = contextx.WithAuthType(ctx, s.Name())

			for k, v := range claims.Extra {
				ctx = context.WithValue(ctx, k, v)
			}

			c.Request = c.Request.WithContext(ctx)
			c.Next()
			return
		}

		response.Error(c, errors.NewUnauthorized("Sesi Anda tidak dikenali. Silakan masuk kembali."))
		c.Abort()
	}
}

// RequirePermission menjaga endpoint berdasarkan permission code (RBAC).
// Sumber permission codes = request context (diisi Auth lewat
// contextx.WithPermissionCodes), konsisten dengan Auth di atas. Respons gagal
// memakai envelope framework (response.Error), bukan JSON mentah.
func RequirePermission(value string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		codes := contextx.GetPermissionCodes(ctx.Request.Context())

		if codes == nil || !codes[value] {
			response.Error(ctx, errors.NewForbidden("Anda tidak memiliki akses untuk tindakan ini."))
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}
