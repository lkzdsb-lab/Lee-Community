package handler

import (
	"Lee_Community/internal/pkg"
	"Lee_Community/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type EmailHandler struct {
	svc *service.EmailService
}

type SendCodeReq struct {
	Scope string `json:"scope" binding:"required,oneof=register reset"`
	Email string `json:"email" binding:"required,email"`
}
type VerifyReq struct {
	Scope string `json:"scope" binding:"required,oneof=register reset"`
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code"  binding:"required,len=6"`
}

func NewEmailHandler(cfg pkg.SMTPConfig) *EmailHandler {
	return &EmailHandler{svc: service.NewEmailService(cfg)}
}

func (h *EmailHandler) SendCode(c *gin.Context) {
	if err := c.ShouldBindBodyWithJSON(&SendCodeReq{}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid params"})
		return
	}

	// 从url获取值
	scope := c.Query("scope")
	email := c.Query("email")

	switch scope {
	case "register":
		if err := h.svc.SendRegisterCode(email); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
			return
		}
	case "reset":
		if err := h.svc.SendResetCode(email); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid scope"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"msg": "Send code successfully"})
}

// VerifyCode 校验code和服务端的是否相同
func (h *EmailHandler) VerifyCode(c *gin.Context) {
	var req VerifyReq

	if err := c.ShouldBindBodyWithJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid params"})
		return
	}

	// 从url获取值
	scope := c.Param("scope")
	email := req.Email
	code := req.Code

	ok, err := h.svc.VerifyCode(scope, email, code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "Verify successfully", "valid": ok})
}
