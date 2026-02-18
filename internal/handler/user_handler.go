package handler

import (
	"net/http"

	"Lee_Community/internal/service"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	svc *service.UserService
}

// RegisterReq 注册请求体
type RegisterReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Code     string `json:"code" binding:"required,len=6"`
}

// ResetReq 忘记密码请求体
type ResetReq struct {
	Email       string `json:"email"`
	Code        string `json:"code" binding:"required,len=6"`
	NewPassword string `json:"new_password"`
}

type ChangePasswordReq struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func NewUserHandler() *UserHandler {
	return &UserHandler{
		svc: service.NewUserService(),
	}
}

// Register 注册接口
func (h *UserHandler) Register(c *gin.Context) {
	var req RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid params"})
		return
	}

	if err := h.svc.Register(req.Username, req.Password, req.Email, req.Code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"msg": "ok"})
}

// Login 登录接口
func (h *UserHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid params"})
		return
	}

	token, err := h.svc.Login(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"AccessToken": token.AccessToken, "RefreshToken": token.RefreshToken})
}

func (h *UserHandler) Logout(c *gin.Context) {
	userIDAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"msg": "unauthorized"})
		return
	}
	userID := userIDAny.(uint64)

	if err := h.svc.Logout(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "logout failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"msg": "ok"})
}

// TokenRefresh 利用refresh来更新access
func (h *UserHandler) TokenRefresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid params"})
		return
	}

	token, err := h.svc.Refresh(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"AccessToken": token.AccessToken, "RefreshToken": token.RefreshToken})
}

func (h *UserHandler) ResetPassword(c *gin.Context) {
	var req ResetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid params"})
		return
	}

	if err := h.svc.ResetCode(req.Email, req.Code, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"msg": "reset password successfully"})
}

func (h *UserHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid params"})
		return
	}

	// 校验旧密码是否正确
	userIDAny, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"msg": "unauthorized"})
		return
	}

	err := h.svc.ChangePassword(userIDAny.(uint64), req.OldPassword, req.NewPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"msg": "change password successfully"})
}
