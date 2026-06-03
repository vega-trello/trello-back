package router

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vega-trello/trello-back/internal/auth"
	"github.com/vega-trello/trello-back/internal/handler"
	"github.com/vega-trello/trello-back/internal/middleware"
	"github.com/vega-trello/trello-back/internal/service"
)

func SetupRouter(
	userHandler *handler.UserHandler,
	projectHandler *handler.ProjectHandler,
	columnHandler *handler.ColumnHandler,
	taskHandler *handler.TaskHandler,
	memberHandler *handler.MemberHandler,
	assigneeHandler *handler.AssigneeHandler,
	tagHandler *handler.TagHandler,
	roleHandler *handler.RoleHandler,
	statusHandler *handler.StatusHandler,
	permissionHandler *handler.PermissionHandler,
	jwtManager *auth.JWTManager,
	permissionChecker service.PermissionChecker,
	permissionDB *pgxpool.Pool,
) *gin.Engine {
	gin.SetMode(gin.DebugMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:8080",
			"http://localhost:5173",
			"http://127.0.0.1:8080",
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

	// PROTECTED ROUTES (все требуют аутентификации)
	protected := r.Group("")
	protected.Use(middleware.Auth(jwtManager))
	{
		// User endpoints
		protected.GET("/self", userHandler.GetSelfProfile)
		protected.PATCH("/self", userHandler.UpdateSelfProfile)
		protected.GET("/user", userHandler.GetOtherUserProfile)

		//Permission endpoints
		protected.GET("/projects/permissions", permissionHandler.ListPermissions)

		// Project endpoints
		protected.GET("/projects", projectHandler.ListProjects)
		protected.POST("/projects", projectHandler.CreateProject)
		protected.GET("/projects/:projectUUID",
			middleware.RequirePermission(permissionChecker, service.PermViewProject),
			projectHandler.GetProject)
		protected.PATCH("/projects/:projectUUID",
			middleware.RequirePermission(permissionChecker, service.PermManageProject),
			projectHandler.UpdateProject)
		protected.DELETE("/projects/:projectUUID",
			middleware.RequirePermission(permissionChecker, service.PermManageProject),
			projectHandler.DeleteProject)

		// Column endpoints
		protected.GET("/projects/:projectUUID/columns",
			middleware.RequirePermission(permissionChecker, service.PermViewProject),
			columnHandler.ListProjectColumns)
		protected.POST("/projects/:projectUUID/columns",
			middleware.RequirePermission(permissionChecker, service.PermManageColumns),
			columnHandler.CreateColumn)
		protected.GET("/columns/:columnID",
			middleware.RequireColumnPermission(permissionChecker, permissionDB, service.PermViewProject), // ← Новый миддлвар
			columnHandler.GetColumn)
		protected.PATCH("/columns/:columnID",
			middleware.RequireColumnPermission(permissionChecker, permissionDB, service.PermManageColumns), // ← Новый миддлвар
			columnHandler.UpdateColumn)
		protected.DELETE("/columns/:columnID",
			middleware.RequireColumnPermission(permissionChecker, permissionDB, service.PermManageColumns), // ← Новый миддлвар
			columnHandler.DeleteColumn)
		protected.POST("/columns/:columnID/move",
			middleware.RequireColumnPermission(permissionChecker, permissionDB, service.PermManageColumns),
			columnHandler.MoveColumn)

		//Task endpoints
		protected.GET("/projects/:projectUUID/tasks",
			middleware.RequirePermission(permissionChecker, service.PermViewProject),
			taskHandler.ListProjectTasks)
		protected.POST("/projects/:projectUUID/tasks",
			middleware.RequirePermission(permissionChecker, service.PermManageTasks),
			taskHandler.CreateTask)
		protected.GET("/projects/:projectUUID/task",
			middleware.RequirePermission(permissionChecker, service.PermViewProject),
			taskHandler.GetTask)
		protected.PATCH("/projects/:projectUUID/task",
			middleware.RequirePermission(permissionChecker, service.PermManageTasks),
			taskHandler.UpdateTask)
		protected.DELETE("/projects/:projectUUID/task",
			middleware.RequirePermission(permissionChecker, service.PermManageTasks),
			taskHandler.DeleteTask)
		protected.POST("/projects/:projectUUID/task/move",
			middleware.RequirePermission(permissionChecker, service.PermManageTasks),
			taskHandler.MoveTask)

		// Member endpoints
		protected.GET("/projects/:projectUUID/members",
			middleware.RequirePermission(permissionChecker, service.PermViewProject),
			memberHandler.ListProjectMembers)
		protected.POST("/projects/:projectUUID/members",
			middleware.RequirePermission(permissionChecker, service.PermManageMembers),
			memberHandler.AddMember)
		protected.GET("/projects/:projectUUID/member",
			middleware.RequirePermission(permissionChecker, service.PermViewProject),
			memberHandler.GetMember)
		protected.PATCH("/projects/:projectUUID/member",
			middleware.RequirePermission(permissionChecker, service.PermManageMembers),
			memberHandler.UpdateMemberRole)
		protected.DELETE("/projects/:projectUUID/member",
			middleware.RequirePermission(permissionChecker, service.PermManageMembers),
			memberHandler.RemoveMember)

		// Assignee endpoints
		assigneeGroup := protected.Group("/projects/:projectUUID")
		{
			assigneeGroup.GET("/assignees",
				middleware.RequirePermission(permissionChecker, service.PermViewProject),
				assigneeHandler.ListTaskAssignees)
			assigneeGroup.POST("/assignees",
				middleware.RequirePermission(permissionChecker, service.PermManageAssignees),
				assigneeHandler.AddAssignee)
			assigneeGroup.DELETE("/assignee",
				middleware.RequirePermission(permissionChecker, service.PermManageAssignees),
				assigneeHandler.RemoveAssignee)
		}

		// Tag endpoints
		protected.GET("/projects/:projectUUID/tag",
			middleware.RequirePermission(permissionChecker, service.PermViewProject),
			tagHandler.ListProjectTags)
		protected.POST("/projects/:projectUUID/tag",
			middleware.RequirePermission(permissionChecker, service.PermManageTags),
			tagHandler.CreateTag)
		protected.PATCH("/projects/:projectUUID/tag",
			middleware.RequirePermission(permissionChecker, service.PermManageTags),
			tagHandler.UpdateTag)
		protected.DELETE("/projects/:projectUUID/tag",
			middleware.RequirePermission(permissionChecker, service.PermManageTags),
			tagHandler.DeleteTag)

		// Tag endpoints
		protected.GET("/projects/:projectUUID/task/tags",
			middleware.RequirePermission(permissionChecker, service.PermViewProject),
			tagHandler.ListTaskTags)
		protected.POST("/projects/:projectUUID/task/tags",
			middleware.RequirePermission(permissionChecker, service.PermManageTags),
			tagHandler.AddTagToTask)
		protected.DELETE("/projects/:projectUUID/task/tags",
			middleware.RequirePermission(permissionChecker, service.PermManageTags),
			tagHandler.RemoveTagFromTask)

		// Role endpoints
		roleGroup := protected.Group("/projects/:projectUUID/roles")
		{
			roleGroup.GET("",
				middleware.RequirePermission(permissionChecker, service.PermViewProject),
				roleHandler.ListProjectRoles)
			roleGroup.POST("",
				middleware.RequirePermission(permissionChecker, service.PermManageRoles),
				roleHandler.CreateRole)
			roleGroup.GET("/:roleID",
				middleware.RequirePermission(permissionChecker, service.PermViewProject),
				roleHandler.GetRole)
			roleGroup.PATCH("/:roleID",
				middleware.RequirePermission(permissionChecker, service.PermManageRoles),
				roleHandler.UpdateRole)
			roleGroup.DELETE("/:roleID",
				middleware.RequirePermission(permissionChecker, service.PermManageRoles),
				roleHandler.DeleteRole)
			roleGroup.GET("/:roleID/permissions",
				middleware.RequirePermission(permissionChecker, service.PermViewProject),
				roleHandler.GetRolePermissions)
		}

		// Status endpoints
		statusGroup := protected.Group("/projects/:projectUUID/statuses")
		{
			statusGroup.GET("",
				middleware.RequirePermission(permissionChecker, service.PermViewProject),
				statusHandler.ListProjectStatuses)
			statusGroup.POST("",
				middleware.RequirePermission(permissionChecker, service.PermManageStatuses),
				statusHandler.CreateStatus)
			statusGroup.GET("/:statusID",
				middleware.RequirePermission(permissionChecker, service.PermViewProject),
				statusHandler.GetStatus)
			statusGroup.PATCH("/:statusID",
				middleware.RequirePermission(permissionChecker, service.PermManageStatuses),
				statusHandler.UpdateStatus)
			statusGroup.DELETE("/:statusID",
				middleware.RequirePermission(permissionChecker, service.PermManageStatuses),
				statusHandler.DeleteStatus)
		}
	}

	return r
}
