package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/celestiaorg/knuu/internal/api/v1/services"
	"github.com/celestiaorg/knuu/internal/database/models"
)

type TestHandler struct {
	testService *services.TestService
	logger      *logrus.Logger
}

func NewTestHandler(ts *services.TestService, logger *logrus.Logger) *TestHandler {
	if logger == nil {
		logger = logrus.New()
	}
	return &TestHandler{
		testService: ts,
		logger:      logger,
	}
}

func (h *TestHandler) CreateTest(c *gin.Context) {
	logger := h.logger.WithFields(logrus.Fields{
		"handler":  "CreateTest",
		"method":   c.Request.Method,
		"path":     c.Request.URL.Path,
		"clientIP": c.ClientIP(),
	})
	user, err := getUserFromContext(c)
	if err != nil {
		logger.Debug(err.Error())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var input models.Test
	if err := c.ShouldBindJSON(&input); err != nil {
		logger.Debug(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	input.UserID = user.ID
	if err := h.testService.Create(c.Request.Context(), &input); err != nil {
		if errors.Is(err, services.ErrTestAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("test already exists with scope: %s", input.Scope)})
			return
		}
		logger.Debug(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create test"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Test created successfully"})
}

func (h *TestHandler) GetTestDetails(c *gin.Context) {
	logger := h.logger.WithFields(logrus.Fields{
		"handler":  "GetTestDetails",
		"method":   c.Request.Method,
		"path":     c.Request.URL.Path,
		"clientIP": c.ClientIP(),
	})
	_ = logger
	var test models.Test
	c.JSON(http.StatusOK, test)
}

func (h *TestHandler) GetTestLogs(c *gin.Context) {
	logger := h.logger.WithFields(logrus.Fields{
		"handler":  "GetTestLogs",
		"method":   c.Request.Method,
		"path":     c.Request.URL.Path,
		"clientIP": c.ClientIP(),
	})

	user, err := getUserFromContext(c)
	if err != nil {
		logger.Debug(err.Error())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	scope := c.Param("scope")

	logFilePath, err := h.testService.TestLogsPath(c.Request.Context(), user.ID, scope)
	if err != nil {
		logger.Debug(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get test logs"})
		return
	}

	c.FileAttachment(logFilePath, fmt.Sprintf("%s.log", scope))
}
