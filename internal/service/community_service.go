package service

import (
	"errors"

	"Lee_Community/internal/model"
	"Lee_Community/internal/repository/mysql"
)

type CommunityService struct {
	repo       *mysql.CommunityRepository
	memberRepo *mysql.CommunityMemberRepository
}

func NewCommunityService() *CommunityService {
	return &CommunityService{
		repo:       &mysql.CommunityRepository{},
		memberRepo: &mysql.CommunityMemberRepository{},
	}
}

func (s *CommunityService) CreateCommunity(userID uint64, name, desc string) (*model.Community, error) {
	if name == "" {
		return nil, errors.New("community name required")
	}

	community := &model.Community{
		Name:        name,
		Description: desc,
		CreatorID:   userID,
	}

	if _, err := s.repo.Create(community); err != nil {
		return nil, err
	}

	return community, nil
}

func (s *CommunityService) JoinCommunity(userID, communityID uint64) error {
	return s.memberRepo.Join(&model.CommunityMember{
		CommunityID: communityID,
		UserID:      userID,
		Role:        0,
	})
}

func (s *CommunityService) LeaveCommunity(userID, communityID uint64) error {
	return s.memberRepo.Leave(communityID, userID)
}

func (s *CommunityService) ListCommunities(page, size int) ([]model.Community, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 50 {
		size = 20
	}

	offset := (page - 1) * size
	return s.repo.List(offset, size)
}
