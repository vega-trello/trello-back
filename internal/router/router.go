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
	memberHandler *handler.MemberHandler,
	assigneeHandler *handler.AssigneeHandler,
	tagHandler *handler.TagHandler,
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

	authGroup := r.Group("/auth")
	{
		authGroup.POST("/register", userHandler.Register)
		authGroup.POST("/login", userHandler.Login)
		authGroup.POST("/sso/exchange", userHandler.ExchangeSSOToken)
		authGroup.POST("/logout", middleware.Auth(jwtManager), userHandler.Logout)
	}

	protected := r.Group("")
	protected.Use(middleware.Auth(jwtManager))
	{
		//User endpoints
		protected.GET("/user", userHandler.GetProfile)
		protected.PATCH("/user", userHandler.UpdateProfile)

		//Project endpoints
		protected.GET("/projects", projectHandler.ListProjects)
		protected.POST("/projects", projectHandler.CreateProject)
		protected.GET("/projects/:projectUUID", projectHandler.GetProject)
		protected.PATCH("/projects/:projectUUID", projectHandler.UpdateProject)
		protected.DELETE("/projects/:projectUUID", projectHandler.DeleteProject)

		//Column endpoints
		protected.GET("/projects/:projectUUID/columns", columnHandler.ListProjectColumns)
		protected.POST("/projects/:projectUUID/columns", columnHandler.CreateColumn)
		protected.GET("/columns/:columnID", columnHandler.GetColumn)
		protected.PATCH("/columns/:columnID", columnHandler.UpdateColumn)
		protected.DELETE("/columns/:columnID", columnHandler.DeleteColumn)
		protected.POST("/columns/:columnID/move", columnHandler.MoveColumn)

		//Task endpoints
		protected.GET("/projects/:projectUUID/tasks", taskHandler.ListProjectTasks)
		protected.POST("/projects/:projectUUID/tasks", taskHandler.CreateTask)
		protected.GET("/projects/:projectUUID/task", taskHandler.GetTask)
		protected.PATCH("/projects/:projectUUID/task", taskHandler.UpdateTask)
		protected.DELETE("/projects/:projectUUID/task", taskHandler.DeleteTask)

		//Member endpoints
		protected.GET("/projects/:projectUUID/members", memberHandler.ListProjectMembers)
		protected.POST("/projects/:projectUUID/members", memberHandler.AddMember)
		protected.GET("/projects/:projectUUID/member", memberHandler.GetMember)
		protected.PATCH("/projects/:projectUUID/member", memberHandler.UpdateMemberRole)
		protected.DELETE("/projects/:projectUUID/member", memberHandler.RemoveMember)

		//assignee endpoints
		rg := protected.Group("/projects/:projectUUID")
		rg.GET("/assignees", assigneeHandler.ListTaskAssignees)
		rg.POST("/assignees", assigneeHandler.AddAssignee)
		rg.DELETE("/assignee", assigneeHandler.RemoveAssignee)

		//project tags endpoints
		rg.GET("/tag", tagHandler.ListProjectTags)
		rg.POST("/tag", tagHandler.CreateTag)
		rg.PATCH("/tag", tagHandler.UpdateTag)
		rg.DELETE("/tag", tagHandler.DeleteTag)

		// Task tags endpoints
		rg.GET("/task/tags", tagHandler.ListTaskTags)
		rg.POST("/task/tags", tagHandler.AddTagToTask)
		rg.DELETE("/task/tags/:tagID", tagHandler.RemoveTagFromTask)
	}

	return r
}
