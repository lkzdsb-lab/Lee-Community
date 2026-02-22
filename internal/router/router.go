package router

import (
	"Lee_Community/internal/handler"
	"Lee_Community/internal/middleware"
	"Lee_Community/internal/pkg"

	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {
	r := gin.Default()

	// 配置邮件环境
	emailCfg := pkg.SMTPConfig{
		Host:     "3140605455@qq.com",
		Port:     587,
		Username: "no-reply@qq.com",
		Password: "apple123456",
		From:     "NoReply <no-reply@example.com>",
	}

	user := handler.NewUserHandler()
	email := handler.NewEmailHandler(emailCfg)
	community := handler.NewCommunityHandler()
	post := handler.NewPostHandler()
	follow := handler.NewFollowHandler()

	// 邮件相关接口
	emailGroup := r.Group("/api/email")
	{
		emailGroup.POST("/:scope/code", email.SendCode)
	}

	// 用户相关接口
	userGroup := r.Group("/api/user")
	{
		userGroup.POST("/register", user.Register)
		userGroup.POST("/login", user.Login)
		userGroup.POST("/logout", user.Logout)
		userGroup.POST("/reset", user.ResetPassword)
	}

	// token相关接口
	tokenGroup := r.Group("/api/token")
	{
		tokenGroup.POST("/refresh", user.TokenRefresh)
	}

	// 登录态接口
	authGroup := r.Group("/api/auth")
	authGroup.Use(middleware.AuthMiddleware())
	{
		authGroup.POST("/change-password", user.ChangePassword)
	}

	// 社区相关接口
	communityGroup := r.Group("/api/community")
	communityGroup.Use(middleware.AuthMiddleware())
	{
		communityGroup.POST("/create", community.Create)
		communityGroup.POST("/join", community.Join)
		communityGroup.POST("/leave", community.Leave)
		communityGroup.GET("/list", community.List)
	}

	// 帖子相关接口
	postGroup := r.Group("/api/post")
	postGroup.Use(middleware.AuthMiddleware())
	{
		postGroup.POST("/create", post.CreatePost)
		postGroup.DELETE("/:id", post.DeletePost)
		postGroup.GET("/list/:id", post.ListByCommunity)
	}

	// 用户关注相关接口
	FollowGroup := r.Group("/api/follow")
	FollowGroup.Use(middleware.AuthMiddleware())
	{
		FollowGroup.POST("/", follow.Follow)
		FollowGroup.GET("/followings", follow.ListFollowings)
		FollowGroup.GET("/followers", follow.ListFollowers)
		FollowGroup.GET("/relation", follow.Relation)
	}

	return r
}
