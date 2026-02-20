package handler

import (
	"net/http"
	"strconv"

	"Lee_Community/internal/service"

	"github.com/gin-gonic/gin"
)

type CommunityHandler struct {
	svc *service.CommunityService
}

type CommunityCreateReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func NewCommunityHandler() *CommunityHandler {
	return &CommunityHandler{
		svc: service.NewCommunityService(),
	}
}

func (h *CommunityHandler) Create(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint64)

	var req CommunityCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid params"})
		return
	}

	community, err := h.svc.CreateCommunity(userID, req.Name, req.Description)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          community.ID,
		"name":        community.Name,
		"description": community.Description,
	})
}

func (h *CommunityHandler) Join(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint64)

	idStr := c.Param("id")
	communityID, _ := strconv.ParseUint(idStr, 10, 64)

	if err := h.svc.JoinCommunity(userID, communityID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"msg": "ok"})
}

func (h *CommunityHandler) Leave(c *gin.Context) {
	userIDAny, _ := c.Get("user_id")
	userID := userIDAny.(uint64)

	idStr := c.Param("id")
	communityID, _ := strconv.ParseUint(idStr, 10, 64)

	if err := h.svc.LeaveCommunity(userID, communityID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"msg": "ok"})
}

func (h *CommunityHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	size, _ := strconv.Atoi(c.Query("size"))

	list, err := h.svc.ListCommunities(page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "list failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"list": list})
}
