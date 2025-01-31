package handlers

import (
	"errors"
	"net/http"

	"github.com/celestiaorg/knuu/internal/api/v1/middleware"
	"github.com/celestiaorg/knuu/internal/api/v1/services"
	"github.com/celestiaorg/knuu/internal/database/models"
	"github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService services.UserService
	auth        *middleware.Auth
	logger      *logrus.Logger
}

func NewUserHandler(userService services.UserService, auth *middleware.Auth, logger *logrus.Logger) *UserHandler {
	if logger == nil {
		logger = logrus.New()
	}
	return &UserHandler{
		userService: userService,
		auth:        auth,
		logger:      logger,
	}
}

func (h *UserHandler) Register(c *gin.Context) {
	logger := h.logger.WithFields(logrus.Fields{
		"handler":  "Register",
		"method":   c.Request.Method,
		"path":     c.Request.URL.Path,
		"clientIP": c.ClientIP(),
	})
	var input models.User
	if err := c.ShouldBindJSON(&input); err != nil {
		logger.Debug(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	_, err := h.userService.Register(c.Request.Context(), &input)
	if err != nil {
		logger.Debug(err.Error())
		if errors.Is(err, services.ErrUsernameAlreadyTaken) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully"})
}

func (h *UserHandler) Login(c *gin.Context) {
	logger := h.logger.WithFields(logrus.Fields{
		"handler":  "Login",
		"method":   c.Request.Method,
		"path":     c.Request.URL.Path,
		"clientIP": c.ClientIP(),
	})
	var input struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		logger.Debug(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	user, err := h.userService.Authenticate(c.Request.Context(), input.Username, input.Password)
	if err != nil {
		logger.Debug(err.Error())
		c.JSON(http.StatusUnauthorized, gin.H{"error": ErrInvalidCredentials.Error()})
		return
	}

	token, err := h.auth.GenerateToken(user)
	if err != nil {
		logger.Debug(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}
