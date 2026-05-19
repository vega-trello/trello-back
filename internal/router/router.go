package router

import (
	"github.com/gin-gonic/gin"
	"github.com/vega-trello/trello-back/internal/auth"
	"github.com/vega-trello/trello-back/internal/handler"
	"github.com/vega-trello/trello-back/internal/middleware"
)

func SetupRouter(
	userHandler *handler.UserHandler,
	jwtManager *auth.JWTManager,
) *gin.Engine {
	// gin.DebugMode - подробные логи, stack traces при паниках (для разработки)
	// gin.ReleaseMode - минимальные логи, оптимизация (для продакшена)
	// В будущем вынести в конфиг/env
	gin.SetMode(gin.DebugMode)

	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	//public router
	authGroup := r.Group("/auth")
	{
		authGroup.POST("/register", userHandler.Register)
		authGroup.POST("/login", userHandler.Login)
		authGroup.POST("/sso/exchange", userHandler.ExchangeSSOToken)

		authGroup.POST("/logout", middleware.Auth(jwtManager), userHandler.Logout)
	}

	//protected router
	protected := r.Group("")
	//Middleware применяется ко всем роутам внутри группы
	protected.Use(middleware.Auth(jwtManager))
	{
		protected.GET("/user", userHandler.GetProfile)
		protected.PATCH("/user", userHandler.UpdateProfile)

		// protected.GET("/projects", projectHandler.List)
		// protected.POST("/projects", projectHandler.Create)
		// protected.GET("/tasks", taskHandler.List)
		// protected.POST("/tasks", taskHandler.Create)
		// etc.
	}

	return r
}
