package service

import (
	"Lee_Community/internal/pkg"
	"Lee_Community/internal/repository/redis"
)

type EmailService struct {
	emailCfg pkg.SMTPConfig
	rds      *redis.EmailRepository
}

func NewEmailService(cfg pkg.SMTPConfig) *EmailService {
	return &EmailService{emailCfg: cfg, rds: &redis.EmailRepository{}}
}

// SendRegisterCode 发送注册验证码
func (s *EmailService) SendRegisterCode(email string) error {
	code, err := pkg.RandDigits(6)
	if err != nil {
		return err
	}
	err = s.rds.RegisterEmailCodePending(email, code)
	if err != nil {
		return err
	}

	html := pkg.EmailCodeHTML("注册验证", code, redis.DefaultEmailCodeTTL)
	if err = pkg.SendEmail(s.emailCfg, email, "注册验证码", html); err != nil {
		return err
	}

	if err = s.rds.MarkRegisterCodePending(email); err != nil {
		_ = s.rds.DeleteRegisterCodePending(email)
		return err
	}

	return nil
}

// SendResetCode 发送重置密码验证码
func (s *EmailService) SendResetCode(email string) error {
	code, err := pkg.RandDigits(6)
	if err != nil {
		return err
	}

	// 先写入pending键
	err = s.rds.ResetEmailCodePending(email, code)
	if err != nil {
		return err
	}

	// 发送邮件
	html := pkg.EmailCodeHTML("重置密码", code, redis.DefaultEmailCodeTTL)
	if err = pkg.SendEmail(s.emailCfg, email, "密码重置验证码", html); err != nil {
		return err
	}

	// 邮件发送后再将pending转为confirmed
	if err = s.rds.MarkCodePending(email); err != nil {
		// 如果确认失败，清楚pending键
		_ = s.rds.DeleteCodePending(email)
		return err
	}

	return nil
}

// VerifyCode 校验验证码并一次性删除
func (s *EmailService) VerifyCode(scope, email, code string) (bool, error) {
	val, err := s.rds.GetEmailCode(scope, email)
	if err != nil {
		// redis.Nil 表示不存在或已过期
		return false, err
	}
	ok := val == code
	if ok {
		err = s.rds.DeleteEmailCode(scope, email)
		if err != nil {
			return false, err
		}
		return ok, nil
	}
	return false, nil
}
