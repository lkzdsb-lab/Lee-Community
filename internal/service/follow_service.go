package service

import (
	"context"
	"errors"

	"Lee_Community/internal/repository/mysql"
)

type FollowService struct {
	repo *mysql.FollowRepository
}

func NewFollowService() *FollowService {
	return &FollowService{
		repo: &mysql.FollowRepository{},
	}
}

func (s *FollowService) Follow(ctx context.Context, followerID, followeeID uint64) (bool, error) {
	if followerID == 0 || followeeID == 0 {
		return false, errors.New("invalid user id")
	}
	if followerID == followeeID {
		return false, errors.New("cannot follow self")
	}
	return s.repo.Follow(ctx, followerID, followeeID)
}

func (s *FollowService) Unfollow(ctx context.Context, followerID, followeeID uint64) (bool, error) {
	if followerID == 0 || followeeID == 0 {
		return false, errors.New("invalid user id")
	}
	if followerID == followeeID {
		return false, errors.New("cannot unfollow self")
	}
	return s.repo.Unfollow(ctx, followerID, followeeID)
}

func (s *FollowService) IsFollowing(ctx context.Context, followerID, followeeID uint64) (bool, error) {
	if followerID == 0 || followeeID == 0 {
		return false, errors.New("invalid user id")
	}
	return s.repo.IsFollowing(ctx, followerID, followeeID)
}

func (s *FollowService) ListFollowings(ctx context.Context, userID uint64, cursor uint64, limit int) (any, uint64, error) {
	return s.repo.ListFollowings(ctx, userID, cursor, limit)
}

func (s *FollowService) ListFollowers(ctx context.Context, userID uint64, cursor uint64, limit int) (any, uint64, error) {
	return s.repo.ListFollowers(ctx, userID, cursor, limit)
}
