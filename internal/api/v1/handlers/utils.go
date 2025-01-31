package handlers

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/celestiaorg/knuu/internal/api/v1/middleware"
	"github.com/celestiaorg/knuu/internal/database/models"
)

func getUserFromContext(c *gin.Context) (*models.User, error) {
	user, ok := c.Get(middleware.UserContextKey)
	if !ok {
		return nil, errors.New("user not found in context")
	}
	authUser, ok := user.(*models.User)
	if !ok {
		return nil, errors.New("invalid user data in context")
	}
	return authUser, nil
}
