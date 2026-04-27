package utils

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

func ExtractBearer(c *gin.Context) (string, error) {
	auth := c.GetHeader("Authorization")

	if auth == "" {
		return "", fmt.Errorf("missing Authorization header")
	}

	parts := strings.SplitN(auth, " ", 2)

	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fmt.Errorf("invalid Authorization format")
	}

	return strings.TrimSpace(parts[1]), nil
}
