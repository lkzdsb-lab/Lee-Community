package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"Lee_Community/internal/middleware"
	"Lee_Community/internal/service"
)

type PostLikeHandler struct {
	svc *service.PostLikeService
}

func NewPostLikeHandler() *PostLikeHandler {
	return &PostLikeHandler{
		svc: service.NewPostLikeService(),
	}
}

func (h *PostLikeHandler) Like(c *gin.Context) {
	uid, _ := c.Get(middleware.ContextUserIDKey)
	pidStr := c.Param("id")
	pid, _ := strconv.ParseUint(pidStr, 10, 64)
	changed, err := h.svc.Like(c.Request.Context(), uid.(uint64), pid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "changed": changed})
}

func (h *PostLikeHandler) Unlike(c *gin.Context) {
	uid, _ := c.Get(middleware.ContextUserIDKey)
	pidStr := c.Param("id")
	pid, _ := strconv.ParseUint(pidStr, 10, 64)
	changed, err := h.svc.Unlike(c.Request.Context(), uid.(uint64), pid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "changed": changed})
}

func (h *PostLikeHandler) IsLiked(c *gin.Context) {
	uid, _ := c.Get(middleware.ContextUserIDKey)
	pidStr := c.Param("id")
	pid, _ := strconv.ParseUint(pidStr, 10, 64)
	liked, err := h.svc.IsLiked(c.Request.Context(), uid.(uint64), pid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "liked": liked})
}

func (h *PostLikeHandler) Count(c *gin.Context) {
	uid, _ := c.Get(middleware.ContextUserIDKey)
	pidStr := c.Param("id")
	pid, _ := strconv.ParseUint(pidStr, 10, 64)
	cnt, err := h.svc.GetCountWithLock(c.Request.Context(), uid.(uint64), pid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "count": cnt})
}
