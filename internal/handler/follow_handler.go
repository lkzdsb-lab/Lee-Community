package handler

import (
	"net/http"
	"strconv"

	"Lee_Community/internal/service"

	"github.com/gin-gonic/gin"
)

type FollowHandler struct {
	svc *service.FollowService
}

func NewFollowHandler() *FollowHandler {
	return &FollowHandler{svc: service.NewFollowService()}
}

type followReq struct {
	FolloweeID uint64 `json:"followee_id" binding:"required"`
	Action     string `json:"action" binding:"required,oneof=follow unfollow"`
}

// Follow 关注接口
func (h *FollowHandler) Follow(c *gin.Context) {
	var req followReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid params"})
		return
	}
	uid := userIDFromCtx(c)
	var (
		changed bool
		err     error
	)
	if req.Action == "follow" {
		changed, err = h.svc.Follow(c.Request.Context(), uid, req.FolloweeID)
	} else {
		changed, err = h.svc.Unfollow(c.Request.Context(), uid, req.FolloweeID)
	}
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"changed": changed})
}

// ListFollowings 获取关注者列表
func (h *FollowHandler) ListFollowings(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Query("user_id"), 10, 64)
	cursor, _ := strconv.ParseUint(c.Query("cursor"), 10, 64)
	limit, _ := strconv.Atoi(c.Query("limit"))
	rows, next, err := h.svc.ListFollowings(c.Request.Context(), userID, cursor, limit)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"list": rows, "next_cursor": next})
}

// ListFollowers 获取粉丝列表
func (h *FollowHandler) ListFollowers(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Query("user_id"), 10, 64)
	cursor, _ := strconv.ParseUint(c.Query("cursor"), 10, 64)
	limit, _ := strconv.Atoi(c.Query("limit"))
	rows, next, err := h.svc.ListFollowers(c.Request.Context(), userID, cursor, limit)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"list": rows, "next_cursor": next})
}

// Relation 获取用户间关系
func (h *FollowHandler) Relation(c *gin.Context) {
	from, _ := strconv.ParseUint(c.Query("from"), 10, 64)
	to, _ := strconv.ParseUint(c.Query("to"), 10, 64)
	ok, err := h.svc.IsFollowing(c.Request.Context(), from, to)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"following": ok})
}

func userIDFromCtx(c *gin.Context) uint64 {
	if v, ok := c.Get("user_id"); ok {
		if id, ok2 := v.(uint64); ok2 {
			return id
		}
	}
	return 0
}
