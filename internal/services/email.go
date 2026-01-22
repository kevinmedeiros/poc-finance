package services

import (
	"fmt"
	"os"

	"github.com/resend/resend-go/v2"
)

// EmailService handles sending emails via Resend API
type EmailService struct {
	apiKey string
	from   string
	client *resend.Client
}

// NewEmailService creates a new email service from environment variables
func NewEmailService() *EmailService {
	apiKey := os.Getenv("RESEND_API_KEY")
	from := os.Getenv("EMAIL_FROM")
	if from == "" {
		from = "onboarding@resend.dev" // Default Resend test sender
	}

	var client *resend.Client
	if apiKey != "" {
		client = resend.NewClient(apiKey)
	}

	return &EmailService{
		apiKey: apiKey,
		from:   from,
		client: client,
	}
}

// IsConfigured returns true if email service is configured
func (s *EmailService) IsConfigured() bool {
	return s.apiKey != "" && s.client != nil
}

// SendPasswordResetEmail sends a password reset email with the given token
func (s *EmailService) SendPasswordResetEmail(toEmail, userName, resetToken, baseURL string) error {
	if !s.IsConfigured() {
		return fmt.Errorf("serviço de email não configurado")
	}

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", baseURL, resetToken)

	subject := "Recuperação de Senha - POC Finance"
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #2563eb;">Recuperação de Senha</h2>
        <p>Olá <strong>%s</strong>,</p>
        <p>Você solicitou a recuperação de senha da sua conta no POC Finance.</p>
        <p>Clique no botão abaixo para redefinir sua senha:</p>
        <div style="text-align: center; margin: 30px 0;">
            <a href="%s" style="background-color: #2563eb; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; display: inline-block;">Redefinir Senha</a>
        </div>
        <p style="color: #666; font-size: 14px;">Ou copie e cole este link no seu navegador:</p>
        <p style="color: #666; font-size: 14px; word-break: break-all;">%s</p>
        <hr style="border: none; border-top: 1px solid #eee; margin: 30px 0;">
        <p style="color: #999; font-size: 12px;">Este link expira em 1 hora.</p>
        <p style="color: #999; font-size: 12px;">Se você não solicitou esta recuperação, ignore este email.</p>
        <p style="color: #999; font-size: 12px;">— Equipe POC Finance</p>
    </div>
</body>
</html>
`, userName, resetLink, resetLink)

	params := &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{toEmail},
		Subject: subject,
		Html:    html,
	}

	_, err := s.client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("erro ao enviar email via Resend: %w", err)
	}

	return nil
}
