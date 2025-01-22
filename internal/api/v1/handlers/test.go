package handlers

import (
	"net/http"

	"github.com/celestiaorg/knuu/internal/api/v1/services"
	"github.com/celestiaorg/knuu/internal/database/models"
	"github.com/gin-gonic/gin"
)

type TestHandler struct {
	testService *services.TestService
}

func NewTestHandler(ts *services.TestService) *TestHandler {
	return &TestHandler{testService: ts}
}

func (h *TestHandler) CreateTest(c *gin.Context) {
	user, err := getUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var input models.Test
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	input.UserID = user.ID
	if err := h.testService.Create(c.Request.Context(), &input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Test created successfully"})
}
