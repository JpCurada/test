package email

import (
	"bytes"
	"fmt"
	"net/smtp"
	"text/template"

	"github.com/ISKOnnect/iskonnect-web/internal/config"
)

type Sender struct {
	cfg config.EmailConfig
}

func NewSender(cfg config.EmailConfig) *Sender {
	return &Sender{cfg: cfg}
}

func (s *Sender) SendVerificationEmail(to, token string) error {
	subject := "Verify Your ISKOnnect Account"
	link := fmt.Sprintf("http://localhost:8080/api/auth/verify-email?token=%s", token)
	body, err := s.parseTemplate("verification", map[string]string{"Link": link})
	if err != nil {
		return err
	}
	return s.send(to, subject, body)
}

func (s *Sender) SendPasswordResetEmail(to, otp string) error {
	subject := "Reset Your ISKOnnect Password"
	body, err := s.parseTemplate("reset", map[string]string{"OTP": otp})
	if err != nil {
		return err
	}
	return s.send(to, subject, body)
}

func (s *Sender) send(to, subject, body string) error {
	auth := smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPassword, s.cfg.SMTPHost)
	headers := map[string]string{
		"From":         fmt.Sprintf("%s <%s>", s.cfg.FromName, s.cfg.FromEmail),
		"To":           to,
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": "text/html; charset=UTF-8",
	}

	msg := ""
	for k, v := range headers {
		msg += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	msg += "\r\n" + body

	addr := fmt.Sprintf("%s:%s", s.cfg.SMTPHost, s.cfg.SMTPPort)
	return smtp.SendMail(addr, auth, s.cfg.FromEmail, []string{to}, []byte(msg))
}

func (s *Sender) parseTemplate(name string, data map[string]string) (string, error) {
	templates := map[string]string{
		"verification": `
			<!DOCTYPE html>
			<html>
			<body style="font-family: Arial; max-width: 600px; margin: 20px auto;">
				<div style="background: #A31D1D; color: white; padding: 20px; text-align: center;">
					<h1>Welcome to ISKOnnect!</h1>
				</div>
				<div style="padding: 20px; background: #f9f9f9;">
					<p>Please verify your email by clicking below:</p>
					<a href="{{.Link}}" style="display: block; background: #A31D1D; color: white; padding: 10px; text-align: center; text-decoration: none;">Verify Email</a>
					<p>Or use this link: {{.Link}}</p>
				</div>
			</body>
			</html>
		`,
		"reset": `
			<!DOCTYPE html>
			<html>
			<body style="font-family: Arial; max-width: 600px; margin: 20px auto;">
				<div style="background: #A31D1D; color: white; padding: 20px; text-align: center;">
					<h1>Reset Your Password</h1>
				</div>
				<div style="padding: 20px; background: #f9f9f9;">
					<p>Use this OTP to reset your password:</p>
					<div style="font-size: 24px; text-align: center; color: #A31D1D;">{{.OTP}}</div>
					<p>Expires in 15 minutes.</p>
				</div>
			</body>
			</html>
		`,
	}

	tmpl, err := template.New(name).Parse(templates[name])
	if err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}