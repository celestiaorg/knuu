package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/celestiaorg/knuu/internal/api/v1/services"
)

func (h *TestHandler) CreateInstance(c *gin.Context) {
	logger := h.logger.WithFields(logrus.Fields{
		"handler":  "CreateInstance",
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

	var input services.Instance
	if err := c.ShouldBindJSON(&input); err != nil {
		logger.Debug(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	input.Scope = c.Param("scope")
	err = h.testService.CreateInstance(c.Request.Context(), user.ID, &input)
	if err != nil {
		logger.Debug(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Instance created successfully"})
}

func (h *TestHandler) GetInstance(c *gin.Context) {
	logger := h.logger.WithFields(logrus.Fields{
		"handler":  "GetInstance",
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

	instance, err := h.testService.GetInstance(c.Request.Context(), user.ID, c.Param("scope"), c.Param("name"))
	if err != nil {
		logger.Debug(err.Error())
		// c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		// return
	}
	c.JSON(http.StatusOK, instance)
}

func (h *TestHandler) GetInstanceStatus(c *gin.Context) {
	logger := h.logger.WithFields(logrus.Fields{
		"handler":  "GetInstanceStatus",
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

	status, err := h.testService.GetInstanceStatus(c.Request.Context(), user.ID, c.Param("scope"), c.Param("name"))
	if err != nil {
		logger.Debug(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": status})
}

func (h *TestHandler) ExecuteInstance(c *gin.Context) {
	logger := h.logger.WithFields(logrus.Fields{
		"handler":  "ExecuteInstance",
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
	name := c.Param("name")

	output, err := h.testService.ExecuteInstance(c.Request.Context(), user.ID, scope, name)
	if err != nil {
		logger.Debug(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output})
}
