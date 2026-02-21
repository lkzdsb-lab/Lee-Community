package service

import (
	"errors"

	"Lee_Community/internal/model"
	"Lee_Community/internal/repository/mysql"
)

type PostService struct {
	repo       *mysql.PostRepository
	memberRepo *mysql.CommunityMemberRepository
}

func NewPostService() *PostService {
	return &PostService{
		repo:       &mysql.PostRepository{},
		memberRepo: &mysql.CommunityMemberRepository{},
	}
}

func (s *PostService) CreatePost(userID, communityID uint64, title, content string) (*model.Post, error) {
	if title == "" {
		return nil, errors.New("title required")
	}

	// 判断是否是 community 成员
	ok, err := s.memberRepo.IsMember(communityID, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("not a member")
	}

	post := &model.Post{
		CommunityID: communityID,
		AuthorID:    userID,
		Title:       title,
		Content:     content,
	}

	if err := s.repo.Create(post); err != nil {
		return nil, err
	}

	return post, nil
}

// ListByCommunity 社区帖子列表
func (s *PostService) ListByCommunity(communityID uint64, page, size int) ([]model.Post, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 50 {
		size = 20
	}

	offset := (page - 1) * size
	return s.repo.ListByCommunity(communityID, offset, size)
}

// ListByCommunityCursor 游标分页：首次不传 lastID/lastCreatedAt（或传 0）
// 返回 nextLastID/nextLastCreatedAt 供下一页使用
func (s *PostService) ListByCommunityCursor(communityID uint64, lastID uint64, lastCreatedAt int64, size int) ([]model.Post, uint64, int64, error) {
	if size <= 0 || size > 50 {
		size = 20
	}
	list, err := s.repo.ListByCommunityCursor(communityID, lastID, lastCreatedAt, size)
	if err != nil {
		return nil, 0, 0, err
	}
	var nextID uint64
	var nextTS int64
	if len(list) > 0 {
		last := list[len(list)-1]
		nextID = last.ID
		nextTS = last.CreatedAt.Unix()
	}
	return list, nextID, nextTS, nil
}

// DeletePost 幂等删除：成功/已删除均返回 nil；仅无权限时报错
func (s *PostService) DeletePost(userID, postID uint64) error {
	affected, err := s.repo.DeleteWithPermission(postID, userID)
	if err != nil {
		return err
	}
	// affected==1：本次从未删->已删；affected==0：可能已删除或无权限
	// 进一步区分无权限与已删除（可选）：尝试读取帖子是否仍存在
	if affected == 0 {
		// 若帖子已被删除或不存在，视为幂等成功
		if _, err := s.repo.FindByID(postID); err != nil {
			return nil
		}
		// 还能读到帖子且未删除，则说明无权限
		return errors.New("no permission")
	}
	return nil
}
