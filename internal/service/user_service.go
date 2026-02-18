package service

import (
	"Lee_Community/internal/model"
	"Lee_Community/internal/pkg"
	"Lee_Community/internal/repository/mysql"
	"Lee_Community/internal/repository/redis"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo     *mysql.UserRepository
	rUser    *redis.UserRepository
	rEmail   *redis.EmailRepository
	emailSvc *EmailService
}

func NewUserService() *UserService {
	return &UserService{
		repo:     &mysql.UserRepository{},
		rUser:    &redis.UserRepository{},
		rEmail:   &redis.EmailRepository{},
		emailSvc: &EmailService{},
	}
}

func (s *UserService) Register(username, password, email, code string) error {
	// 验证code是否正确
	_, err := s.emailSvc.VerifyCode("register", email, code)
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := &model.User{
		Username: username,
		Password: string(hash),
		Email:    email,
	}

	return s.repo.Create(user)
}

func (s *UserService) Login(username, password string) (*pkg.Pair, error) {
	user, err := s.repo.FindByUsername(username)
	if err != nil {
		return nil, errors.New("user not found")
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		return nil, errors.New("invalid password")
	}
	// 将token写入redis
	token, err := pkg.GeneratePair(user.ID)
	if err != nil {
		return nil, err
	}
	err = s.rUser.AddUserToken(user.ID, token.AccessToken)
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (s *UserService) Logout(usrID uint64) error {
	if err := s.rUser.DeleteUserToken(usrID); err != nil {
		return err
	}
	return nil
}

func (s *UserService) ResetCode(email, code, newPassword string) error {
	// 校验code正确性
	ok, err := s.emailSvc.VerifyCode("reset", email, code)
	if err != nil || !ok {
		return errors.New("verification failed")
	}

	// 获取用户信息并更新密码
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	err = s.repo.UpdatePassword(user, string(hash))
	if err != nil {
		return err
	}

	return nil
}

func (s *UserService) Refresh(refreshToken string) (*pkg.Pair, error) {
	return pkg.Refresh(refreshToken)
}

// ChangePassword 登录态修改密码
func (s *UserService) ChangePassword(usrId uint64, oldPassword, newPassword string) error {
	user, err := s.repo.FindByID(usrId)
	if err != nil {
		return err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword))
	if err != nil {
		return errors.New("old password is incorrect")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	err = s.repo.UpdatePassword(user, string(hash))
	if err != nil {
		return err
	}

	err = s.Logout(usrId)
	return err
}
