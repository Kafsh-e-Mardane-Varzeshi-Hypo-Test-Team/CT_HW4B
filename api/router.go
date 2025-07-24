package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetupRouter(h *Handler) *gin.Engine {
	r := gin.Default()
	r.LoadHTMLGlob("templates/*.html")
	r.Static("/static", "./static")

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "home.html", nil)
	})
	r.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})
	r.GET("/dashboard", func(c *gin.Context) {
		c.HTML(http.StatusOK, "dashboard.html", nil)
	})
	r.GET("/project/:id", func(c *gin.Context) {
		c.HTML(http.StatusOK, "project.html", nil)
	})

	r.POST("/api/signup", h.SignupHandler)
	r.POST("/api/login", h.LoginHandler)
	r.POST("/api/validate-session", h.ValidateSessionHandler)
	r.POST("/api/projects", h.CreateProjectHandler)
	r.GET("/api/projects", h.GetProjectsHandler)
	r.GET("/api/projects/:id", h.GetProjectHandler)
	r.GET("/api/projects/:id/events", h.GetEventsHandler)
	r.GET("/api/projects/:id/events/details", h.GetEventDetailsHandler)
	r.POST("/api/logs", h.SubmitLogHandler)

	r.GET("/health", func(c *gin.Context) {
		if err := h.cockroachClient.HealthCheck(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	r.GET("/cluster-status", func(c *gin.Context) {
		status, err := h.cockroachClient.GetClusterStatus()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, status)
	})

	return r
}
