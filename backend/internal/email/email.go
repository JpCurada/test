package email

import (
	"bytes"
	"fmt"
	"net/smtp"
	"text/template"

	"github.com/ISKOnnect/iskonnect-web/internal/config"
)

// Sender handles email sending
type Sender struct {
	config config.EmailConfig
}

// NewSender creates a new email sender
func NewSender(cfg config.EmailConfig) *Sender {
	return &Sender{
		config: cfg,
	}
}

// SendVerificationEmail sends an email with a verification link
func (s *Sender) SendVerificationEmail(to, token string) error {
    subject := "Verify Your ISKOnnect Account"
    verificationLink := fmt.Sprintf("http://localhost:8080/api/auth/verify-email?token=%s", token) // Ensure this is /api/auth/verify-email

    templateData := struct {
        Name             string
        VerificationLink string
    }{
        Name:             to,
        VerificationLink: verificationLink,
    }

    body, err := s.parseTemplate("verification.html", templateData)
    if err != nil {
        return err
    }

    return s.sendEmail(to, subject, body)
}

// SendPasswordResetEmail sends an email with a password reset OTP
func (s *Sender) SendPasswordResetEmail(to, otp string) error {
	subject := "Reset Your ISKOnnect Password"

	templateData := struct {
		Name string
		OTP  string
	}{
		Name: to,
		OTP:  otp,
	}

	body, err := s.parseTemplate("password_reset.html", templateData)
	if err != nil {
		return err
	}

	return s.sendEmail(to, subject, body)
}

// sendEmail sends an email
func (s *Sender) sendEmail(to, subject, body string) error {
	auth := smtp.PlainAuth("", s.config.SMTPUser, s.config.SMTPPassword, s.config.SMTPHost)

	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", s.config.FromName, s.config.FromEmail)
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	addr := fmt.Sprintf("%s:%s", s.config.SMTPHost, s.config.SMTPPort)
	err := smtp.SendMail(addr, auth, s.config.FromEmail, []string{to}, []byte(message))
	if err != nil {
		return err
	}

	return nil
}

// parseTemplate parses an HTML template
func (s *Sender) parseTemplate(templateFileName string, data interface{}) (string, error) {
	// In a real app, you'd load this from a file
	var templateContent string
	
	switch templateFileName {
	case "verification.html":
		templateContent = `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Verify Your ISKOnnect Account</title>
			<style>
				body {
					font-family: Arial, sans-serif;
					line-height: 1.6;
					color: #333;
					max-width: 600px;
					margin: 0 auto;
					padding: 20px;
				}
				.header {
					background-color: #A31D1D;
					color: white;
					padding: 20px;
					text-align: center;
					border-radius: 5px 5px 0 0;
				}
				.content {
					background-color: #f9f9f9;
					padding: 20px;
					border-radius: 0 0 5px 5px;
				}
				.button {
					display: inline-block;
					background-color: #A31D1D;
					color: white;
					padding: 10px 20px;
					text-decoration: none;
					border-radius: 5px;
					margin: 20px 0;
				}
				.footer {
					text-align: center;
					margin-top: 20px;
					font-size: 12px;
					color: #777;
				}
			</style>
		</head>
		<body>
			<div class="header">
				<h1>Welcome to ISKOnnect!</h1>
			</div>
			<div class="content">
				<p>Hello {{.Name}},</p>
				<p>Thank you for registering with ISKOnnect - where PUP students can share and find study materials.</p>
				<p>Please click the button below to verify your email address:</p>
				<p style="text-align: center;">
					<a href="{{.VerificationLink}}" class="button">Verify Email</a>
				</p>
				<p>If the button doesn't work, you can also copy and paste this link into your browser:</p>
				<p>{{.VerificationLink}}</p>
				<p>If you did not sign up for ISKOnnect, you can ignore this email.</p>
			</div>
			<div class="footer">
				<p>&copy; 2025 ISKOnnect. All rights reserved.</p>
				<p>Polytechnic University of the Philippines</p>
			</div>
		</body>
		</html>
		`
	case "password_reset.html":
		templateContent = `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Reset Your ISKOnnect Password</title>
			<style>
				body {
					font-family: Arial, sans-serif;
					line-height: 1.6;
					color: #333;
					max-width: 600px;
					margin: 0 auto;
					padding: 20px;
				}
				.header {
					background-color: #A31D1D;
					color: white;
					padding: 20px;
					text-align: center;
					border-radius: 5px 5px 0 0;
				}
				.content {
					background-color: #f9f9f9;
					padding: 20px;
					border-radius: 0 0 5px 5px;
				}
				.otp {
					font-size: 24px;
					font-weight: bold;
					color: #A31D1D;
					text-align: center;
					padding: 10px;
					margin: 20px 0;
					letter-spacing: 5px;
				}
				.footer {
					text-align: center;
					margin-top: 20px;
					font-size: 12px;
					color: #777;
				}
			</style>
		</head>
		<body>
			<div class="header">
				<h1>Reset Your Password</h1>
			</div>
			<div class="content">
				<p>Hello {{.Name}},</p>
				<p>You have requested to reset your password for your ISKOnnect account. Use the following One-Time Password (OTP) to complete the process:</p>
				<div class="otp">{{.OTP}}</div>
				<p>This OTP will expire in 15 minutes for security reasons.</p>
				<p>If you did not request a password reset, please ignore this email or contact support if you have concerns.</p>
			</div>
			<div class="footer">
				<p>&copy; 2025 ISKOnnect. All rights reserved.</p>
				<p>Polytechnic University of the Philippines</p>
			</div>
		</body>
		</html>
		`
	default:
		return "", fmt.Errorf("template not found: %s", templateFileName)
	}
	
	t, err := template.New(templateFileName).Parse(templateContent)
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	if err = t.Execute(buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}