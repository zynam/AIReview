package handlers

import (
	"net/http"
	"os"
	"strings"

	"aireview/internal/config"
	"aireview/internal/jobs"
	"aireview/internal/session"

	"github.com/gin-gonic/gin"
)

type ReviewHandler struct {
	Service session.Service
	Store   session.Store
	Queue   *jobs.Queue
	Config  config.Config
}

type createReviewRequest struct {
	PRURL      string             `json:"pr_url" binding:"required"`
	ReviewerID string             `json:"reviewer_id" binding:"required"`
	Config     createReviewConfig `json:"config"`
}

type createReviewConfig struct {
	Model       string   `json:"model"`
	ReviewFocus []string `json:"review_focus"`
	MaxFiles    int      `json:"max_files"`
}

func (h ReviewHandler) Create(c *gin.Context) {
	var req createReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	result, err := h.Service.Create(c.Request.Context(), session.CreateSessionRequest{
		PRURL:      req.PRURL,
		ReviewerID: req.ReviewerID,
	})
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	cfg := mergeConfig(h.Config, req.Config)
	if err := h.Queue.Enqueue(c.Request.Context(), jobs.ReviewJob{
		SessionID: result.Session.ID,
		Ref:       result.Ref,
		Config:    cfg,
		MaxFiles:  req.Config.MaxFiles,
	}); err != nil {
		_ = h.Store.UpdateStatus(c.Request.Context(), result.Session.ID, session.StatusFailed, err.Error())
		writeError(c, http.StatusServiceUnavailable, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"id":     result.Session.ID,
		"status": result.Session.Status,
	})
}

func (h ReviewHandler) List(c *gin.Context) {
	items, err := h.Store.ListSessions(c.Request.Context(), session.ListFilter{
		ReviewerID: strings.TrimSpace(c.Query("reviewer_id")),
		Owner:      strings.TrimSpace(c.Query("owner")),
		Repo:       strings.TrimSpace(c.Query("repo")),
		Status:     session.Status(strings.TrimSpace(c.Query("status"))),
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h ReviewHandler) Get(c *gin.Context) {
	detail, err := h.Service.Detail(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeStoreError(c, err)
		return
	}
	c.JSON(http.StatusOK, detail)
}

func (h ReviewHandler) Contexts(c *gin.Context) {
	chunks, err := h.Store.ListContexts(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": chunks})
}

func (h ReviewHandler) Events(c *gin.Context) {
	events, err := h.Store.ListEvents(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	if strings.Contains(c.GetHeader("Accept"), "text/event-stream") || c.Query("stream") == "1" {
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("Content-Type", "text/event-stream")
		for _, event := range events {
			c.SSEvent(event.Type, event)
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": events})
}

func mergeConfig(base config.Config, req createReviewConfig) config.Config {
	cfg := base
	if strings.TrimSpace(os.Getenv("LLM_MODEL")) == "" && strings.TrimSpace(req.Model) != "" {
		cfg.LLM.Model = strings.TrimSpace(req.Model)
	}
	if len(req.ReviewFocus) > 0 {
		cfg.ReviewFocus = req.ReviewFocus
	}
	return cfg
}

func writeStoreError(c *gin.Context, err error) {
	if err == session.ErrNotFound {
		writeError(c, http.StatusNotFound, err)
		return
	}
	writeError(c, http.StatusInternalServerError, err)
}
