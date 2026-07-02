package services

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
	"strings"
)

// Mailer 邮件发送服务，支持 SMTP 协议。
// 通过环境变量配置连接参数，未配置时不发送邮件（仅日志记录）。
type Mailer struct {
	host     string
	port     string
	username string
	password string
	from     string
}

// NewMailer 从环境变量读取 SMTP 配置并返回 Mailer 实例。
// 当 SMTP_HOST 未设置时返回 nil（降级为不发送模式）。
func NewMailer() *Mailer {
	host := os.Getenv("SMTP_HOST")
	if host == "" {
		return nil
	}
	return &Mailer{
		host:     host,
		port:     getEnv("SMTP_PORT", "587"),
		username: os.Getenv("SMTP_USER"),
		password: os.Getenv("SMTP_PASSWORD"),
		from:     getEnv("SMTP_FROM", os.Getenv("SMTP_USER")),
	}
}

// SendPasswordResetEmail 发送密码重置邮件。
// resetLink 应包含完整的重置链接（含 token），例如：
//
//	"https://example.com/reset-password?token=xxx"
func (m *Mailer) SendPasswordResetEmail(to, resetLink string) error {
	if m == nil {
		log.Printf("[MAIL] (dry-run) password reset email to=%s link=%s", to, resetLink)
		return nil
	}
	subject := "Reset Your Password"
	body := fmt.Sprintf(passwordResetTemplate, to, resetLink)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		m.from, to, subject, body)

	addr := fmt.Sprintf("%s:%s", m.host, m.port)
	auth := smtp.PlainAuth("", m.username, m.password, m.host)

	if err := smtp.SendMail(addr, auth, m.from, []string{to}, []byte(msg)); err != nil {
		log.Printf("[MAIL ERROR] send to=%s err=%v", to, err)
		return fmt.Errorf("failed to send email: %w", err)
	}
	log.Printf("[MAIL OK] password reset email sent to=%s", to)
	return nil
}

const passwordResetTemplate = `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;padding:20px;background:#f5f5f5">
<div style="max-width:600px;margin:0 auto;background:#fff;border-radius:8px;padding:30px">
<h2 style="color:#333">Password Reset Request</h2>
<p>Hello,</p>
<p>We received a request to reset your password for your account (<strong>%s</strong>). Click the button below to set a new password:</p>
<div style="margin:25px 0">
<a href="%s" style="display:inline-block;background:#4a90d9;color:#fff;padding:12px 28px;text-decoration:none;border-radius:6px;font-size:16px">Reset Password</a>
</div>
<p style="color:#888;font-size:14px">This link will expire in <strong>30 minutes</strong>. If you didn't make this request, please ignore this email.</p>
<hr style="border:none;border-top:1px solid #eee;margin:20px 0">
<p style="color:#999;font-size:12px">This is an automated message, please do not reply.</p>
</div>
</body>
</html>`

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// BuildResetLink 根据前端地址和 token 构建完整的密码重置链接。
func BuildResetLink(frontendURL, token string) string {
	base := strings.TrimRight(frontendURL, "/")
	return base + "/reset-password?token=" + token
}
