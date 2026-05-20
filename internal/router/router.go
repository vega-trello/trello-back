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
	jwtManager *auth.JWTManager,
) *gin.Engine {
	// gin.DebugMode - подробные логи для разработки
	// gin.ReleaseMode - оптимизация для продакшена
	gin.SetMode(gin.DebugMode)

	r := gin.New()

	r.Use(gin.Recovery())

	r.Use(gin.Logger())

	r.Use(cors.New(cors.Config{
		// Разрешённые источники (фронтенд)
		AllowOrigins: []string{
			"http://localhost:3000",
			"http://localhost:5173",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:5173",
			// В продакшене добавить что-то типо "https://app.trello-clone.com"
		},
		AllowMethods: []string{
			"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS",
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Requested-With",
		},
		ExposeHeaders: []string{
			"Content-Length",
			"X-Request-Id",
		},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Healthcheck для мониторинга, балансировщиков и Docker
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "timestamp": time.Now()})
	})

	// Группа /auth - регистрация, логин, SSO
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
		// Пользовательский профиль
		protected.GET("/user", userHandler.GetProfile)
		protected.PATCH("/user", userHandler.UpdateProfile)

		// Пример (будет реализовать позже)
		// protected.GET("/projects", projectHandler.List)
		// protected.POST("/projects", projectHandler.Create)
		// protected.GET("/projects/:id", projectHandler.Get)
		// protected.PATCH("/projects/:id", projectHandler.Update)
		// protected.DELETE("/projects/:id", projectHandler.Delete)
		//
		// protected.POST("/projects/:id/columns", columnHandler.Create)
		// protected.PATCH("/columns/:id", columnHandler.Update)
		// protected.DELETE("/columns/:id", columnHandler.Delete)
		//
		// protected.POST("/columns/:id/tasks", taskHandler.Create)
		// protected.PATCH("/tasks/:id", taskHandler.Update)
		// protected.POST("/tasks/:id/move", taskHandler.Move)
		// protected.POST("/tasks/:id/archive", taskHandler.Archive)
	}

	return r
}
