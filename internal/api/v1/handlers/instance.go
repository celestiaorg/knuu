package handlers

import (
	"net/http"

	"github.com/celestiaorg/knuu/internal/api/v1/services"
	"github.com/gin-gonic/gin"
)

func (h *TestHandler) CreateInstance(c *gin.Context) {
	user, err := getUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var input services.Instance
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	err = h.testService.CreateInstance(c.Request.Context(), user.ID, &input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Instance created successfully"})
}

func (h *TestHandler) GetInstance(c *gin.Context) {
	user, err := getUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	instance, err := h.testService.GetInstance(c.Request.Context(), user.ID, c.Param("scope"), c.Param("name"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, instance)
}

func (h *TestHandler) GetInstanceStatus(c *gin.Context) {
	user, err := getUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	status, err := h.testService.GetInstanceStatus(c.Request.Context(), user.ID, c.Param("scope"), c.Param("name"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": status})
}
