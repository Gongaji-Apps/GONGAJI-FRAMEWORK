package middleware

import (
	"strings"

	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/contextx"
	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"
	"github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/response"
	"github.com/gin-gonic/gin"
)

// forbiddenMessage is the standardized 403 message used across the
// framework's authorization middlewares.
const forbiddenMessage = "[Forbidden] Afwan, Anda tidak memiliki izin untuk mengakses endpoint ini."

// AuthorizeRoles guards an endpoint by role code. The caller's role is
// read from the gin context key "role_code" (set by Auth). The check
// is case-insensitive: AuthorizeRoles("ADMIN", "STAFF") matches a
// role_code of "admin", "Admin", or "ADMIN".
//
// If no roles are listed, the middleware always rejects with 403 —
// this prevents accidentally registering an unrestricted route by
// passing an empty slice.
//
// Use together with Auth:
//
//	api := r.Group("/api/v1", middleware.Auth(jwtStrategy))
//	api.DELETE("/users/:uuid",
//	    middleware.AuthorizeRoles("ADMIN"),
//	    userHdl.Delete,
//	)
func AuthorizeRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleCode := contextx.GetRoleCode(c.Request.Context())

		if roleCode == "" {
			response.Error(c, errors.NewForbidden(forbiddenMessage))
			c.Abort()
			return
		}

		for _, role := range roles {
			if strings.EqualFold(roleCode, role) {
				c.Next()
				return
			}
		}

		response.Error(c, errors.NewForbidden(forbiddenMessage))
		c.Abort()
	}
}
