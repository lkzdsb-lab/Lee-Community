package handler

import (
	"Lee_Community/internal/service"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type PostHandler struct {
	svc *service.PostService
}

type CreatePostReq struct {
	CommunityID uint64 `json:"community_id"`
	Title       string `json:"title"`
	Content     string `json:"content"`
}

func NewPostHandler() *PostHandler {
	return &PostHandler{
		svc: service.NewPostService(),
	}
}

// CreatePost 创建帖子接口
func (h *PostHandler) CreatePost(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint64)

	var req CreatePostReq

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid params"})
		return
	}

	post, err := h.svc.CreatePost(userID, req.CommunityID, req.Title, req.Content)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": post.ID})
}

// ListByCommunity 获取帖子列表接口（优先游标分页，兼容页码）
func (h *PostHandler) ListByCommunity(c *gin.Context) {
	idStr := c.Param("id")
	communityID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || communityID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid community id"})
		return
	}

	// 游标参数（可选）
	lastIDStr := c.Query("last_id")
	lastTSStr := c.Query("last_created_at")

	// 如果提供了游标，则走游标分页
	if lastIDStr != "" || lastTSStr != "" {
		var lastID uint64
		var lastTS int64
		if lastIDStr != "" {
			if v, e := strconv.ParseUint(lastIDStr, 10, 64); e == nil {
				lastID = v
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid last_id"})
				return
			}
		}
		if lastTSStr != "" {
			if v, e := strconv.ParseInt(lastTSStr, 10, 64); e == nil {
				lastTS = v
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid last_created_at"})
				return
			}
		}

		size, _ := strconv.Atoi(c.Query("size"))

		list, nextID, nextTS, err := h.svc.ListByCommunityCursor(communityID, lastID, lastTS, size)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"msg": "list failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"list":              list,
			"next_last_id":      nextID,
			"next_created_at":   nextTS,
			"next_created_at_s": time.Unix(nextTS, 0).Format(time.RFC3339),
		})
		return
	}

	// 兼容页码查询（不推荐深页使用）
	page, err1 := strconv.Atoi(c.Query("page"))
	size, err2 := strconv.Atoi(c.Query("size"))
	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid page/size"})
		return
	}

	list, err := h.svc.ListByCommunity(communityID, page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "list failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"list": list,
		"page": page,
		"size": size,
	})
}

// DeletePost 删除帖子接口
func (h *PostHandler) DeletePost(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint64)

	idStr := c.Param("id")
	postID, _ := strconv.ParseUint(idStr, 10, 64)

	if err := h.svc.DeletePost(userID, postID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"msg": "deleted"})
}
