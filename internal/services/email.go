package services

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"os"
	"strconv"
)

// EmailService handles sending emails via SMTP
type EmailService struct {
	host     string
	port     int
	user     string
	password string
	from     string
}

// NewEmailService creates a new email service from environment variables
func NewEmailService() *EmailService {
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if port == 0 {
		port = 587 // Default TLS port
	}

	return &EmailService{
		host:     os.Getenv("SMTP_HOST"),
		port:     port,
		user:     os.Getenv("SMTP_USER"),
		password: os.Getenv("SMTP_PASSWORD"),
		from:     os.Getenv("SMTP_FROM"),
	}
}

// IsConfigured returns true if SMTP settings are configured
func (s *EmailService) IsConfigured() bool {
	return s.host != "" && s.user != "" && s.password != "" && s.from != ""
}

// SendPasswordResetEmail sends a password reset email with the given token
func (s *EmailService) SendPasswordResetEmail(toEmail, userName, resetToken, baseURL string) error {
	if !s.IsConfigured() {
		return fmt.Errorf("SMTP não configurado")
	}

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", baseURL, resetToken)

	subject := "Recuperação de Senha - POC Finance"
	body := fmt.Sprintf(`Olá %s,

Você solicitou a recuperação de senha da sua conta no POC Finance.

Clique no link abaixo para redefinir sua senha:
%s

Este link expira em 1 hora.

Se você não solicitou esta recuperação, ignore este email.

Atenciosamente,
Equipe POC Finance
`, userName, resetLink)

	return s.sendEmail(toEmail, subject, body)
}

// sendEmail sends an email using SMTP with TLS
func (s *EmailService) sendEmail(to, subject, body string) error {
	// Build message
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=\"UTF-8\"\r\n\r\n%s",
		s.from, to, subject, body)

	// Connect to SMTP server
	addr := fmt.Sprintf("%s:%d", s.host, s.port)

	// Gmail requires TLS
	auth := smtp.PlainAuth("", s.user, s.password, s.host)

	// For Gmail, we need to use STARTTLS
	tlsConfig := &tls.Config{
		ServerName: s.host,
	}

	// Connect
	conn, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("erro ao conectar ao servidor SMTP: %w", err)
	}
	defer conn.Close()

	// Start TLS
	if err = conn.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("erro ao iniciar TLS: %w", err)
	}

	// Authenticate
	if err = conn.Auth(auth); err != nil {
		return fmt.Errorf("erro de autenticação SMTP: %w", err)
	}

	// Set sender
	if err = conn.Mail(s.from); err != nil {
		return fmt.Errorf("erro ao definir remetente: %w", err)
	}

	// Set recipient
	if err = conn.Rcpt(to); err != nil {
		return fmt.Errorf("erro ao definir destinatário: %w", err)
	}

	// Send body
	w, err := conn.Data()
	if err != nil {
		return fmt.Errorf("erro ao iniciar corpo do email: %w", err)
	}

	_, err = w.Write([]byte(msg))
	if err != nil {
		return fmt.Errorf("erro ao escrever corpo do email: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("erro ao finalizar email: %w", err)
	}

	return conn.Quit()
}
