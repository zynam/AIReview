package handlers

import (
	"fmt"
	"net/http"

	"aireview/internal/session"

	"github.com/gin-gonic/gin"
)

type FindingHandler struct {
	Store session.Store
}

type updateFindingRequest struct {
	FeedbackStatus string `json:"feedback_status" binding:"required"`
}

func (h FindingHandler) Update(c *gin.Context) {
	var req updateFindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	if !validFeedbackStatus(req.FeedbackStatus) {
		writeError(c, http.StatusBadRequest, fmt.Errorf("invalid feedback_status %q", req.FeedbackStatus))
		return
	}
	if err := h.Store.UpdateFindingFeedback(c.Request.Context(), c.Param("id"), req.FeedbackStatus); err != nil {
		writeStoreError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"feedback_status": req.FeedbackStatus})
}

func validFeedbackStatus(status string) bool {
	switch status {
	case "useful", "false_positive", "fixed", "ignored":
		return true
	default:
		return false
	}
}
