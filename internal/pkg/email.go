package pkg

import (
	"crypto/tls"
	"fmt"
	"time"

	"gopkg.in/gomail.v2"
)

type SMTPConfig struct {
	Host     string
	Port     int
	Username string // 发件人邮箱
	Password string // 授权码/密码
	From     string // 显示的发件人，可与 Username 相同
}

func SendEmail(cfg SMTPConfig, to, subject, htmlBody string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", cfg.From)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", htmlBody)

	d := gomail.NewDialer(cfg.Host, cfg.Port, cfg.Username, cfg.Password)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: false}
	return d.DialAndSend(m)
}

func EmailCodeHTML(subject, code string, ttl time.Duration) string {
	minM := int(ttl.Minutes())
	return fmt.Sprintf(`<p>您好，</p><p>您正在进行 <b>%s</b> 操作，验证码为：<b style="font-size:18px;">%s</b>。</p><p>有效期 %d 分钟，请勿泄露给他人。</p>`, subject, code, minM)
}
