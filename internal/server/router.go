package server

import (
	"net/http"

	"aireview/internal/config"
	"aireview/internal/jobs"
	"aireview/internal/server/handlers"
	"aireview/internal/session"

	"github.com/gin-gonic/gin"
)

type Dependencies struct {
	Store   session.Store
	Queue   *jobs.Queue
	Config  config.Config
	Service session.Service
}

func NewRouter(deps Dependencies) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(gin.Recovery(), CORS())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	if deps.Store != nil && deps.Queue != nil {
		reviewHandler := handlers.ReviewHandler{
			Service: deps.Service,
			Store:   deps.Store,
			Queue:   deps.Queue,
			Config:  deps.Config,
		}
		findingHandler := handlers.FindingHandler{Store: deps.Store}
		reportHandler := handlers.ReportHandler{Service: deps.Service}

		api := router.Group("/api/v1")
		api.GET("/reviews", reviewHandler.List)
		api.POST("/reviews", reviewHandler.Create)
		api.GET("/reviews/:id", reviewHandler.Get)
		api.GET("/reviews/:id/contexts", reviewHandler.Contexts)
		api.GET("/reviews/:id/report.md", reportHandler.Markdown)
		api.PATCH("/findings/:id", findingHandler.Update)
	}

	return router
}
