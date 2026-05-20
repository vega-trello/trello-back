package router

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/vega-trello/trello-back/internal/auth"
	"github.com/vega-trello/trello-back/internal/handler"
	"github.com/vega-trello/trello-back/internal/middleware"
)

func SetupRouter(
	userHandler *handler.UserHandler,
	projectHandler *handler.ProjectHandler,
	columnHandler *handler.ColumnHandler,
	taskHandler *handler.TaskHandler,
	jwtManager *auth.JWTManager,
) *gin.Engine {
	gin.SetMode(gin.DebugMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:3000",
			"http://localhost:5173",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:5173",
		},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{
			"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With",
		},
		ExposeHeaders:    []string{"Content-Length", "X-Request-Id"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Healthcheck
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "timestamp": time.Now()})
	})

	// public endpoints
	authGroup := r.Group("/auth")
	{
		authGroup.POST("/register", userHandler.Register)
		authGroup.POST("/login", userHandler.Login)
		authGroup.POST("/sso/exchange", userHandler.ExchangeSSOToken)
		authGroup.POST("/logout", middleware.Auth(jwtManager), userHandler.Logout)
	}

	// protected endpoints
	protected := r.Group("")
	protected.Use(middleware.Auth(jwtManager))
	{
		// User endpoints
		protected.GET("/user", userHandler.GetProfile)
		protected.PATCH("/user", userHandler.UpdateProfile)

		// Project endpoints
		protected.GET("/projects", projectHandler.ListProjects)
		protected.POST("/projects", projectHandler.CreateProject)
		protected.GET("/projects/:projectUUID", projectHandler.GetProject)
		protected.PATCH("/projects/:projectUUID", projectHandler.UpdateProject)
		protected.DELETE("/projects/:projectUUID", projectHandler.DeleteProject)

		// Column endpoints
		protected.GET("/projects/:projectUUID/columns", columnHandler.ListProjectColumns)
		protected.POST("/projects/:projectUUID/columns", columnHandler.CreateColumn)
		protected.GET("/columns/:columnID", columnHandler.GetColumn)
		protected.PATCH("/columns/:columnID", columnHandler.UpdateColumn)
		protected.DELETE("/columns/:columnID", columnHandler.DeleteColumn)
		protected.POST("/columns/:columnID/move", columnHandler.MoveColumn)

		// task endpoints
		protected.GET("/projects/:projectUUID/tasks", taskHandler.ListProjectTasks)
		protected.POST("/projects/:projectUUID/tasks", taskHandler.CreateTask)
		protected.GET("/projects/:projectUUID/task", taskHandler.GetTask)
		protected.PATCH("/projects/:projectUUID/task", taskHandler.UpdateTask) // архивируем задачу через этот эндпоинт
		protected.DELETE("/projects/:projectUUID/task", taskHandler.DeleteTask)
	}

	return r
}
