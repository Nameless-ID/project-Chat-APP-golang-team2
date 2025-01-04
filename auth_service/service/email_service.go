package service

import (
	"auth-service/config"
	"bytes"
	"context"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"time"

	"github.com/mailersend/mailersend-go"
	"go.uber.org/zap"
)

type EmailService interface {
	Send(to, subject, template string, data interface{}) (string, error)
}

type emailService struct {
	log    *zap.Logger
	Mailer *mailersend.Mailersend
	sender mailersend.From
}

func NewEmailService(config config.EmailConfig, log *zap.Logger) EmailService {
	ms := mailersend.NewMailersend(config.ApiKey)
	sender := mailersend.From{
		Name:  config.FromName,
		Email: config.FromEmail,
	}
	return &emailService{Mailer: ms, log: log, sender: sender}
}

func (s *emailService) Send(to, subject, htmlTemplate string, data interface{}) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	message := s.Mailer.Email.NewMessage()

	recipient := mailersend.Recipient{
		Email: to,
	}
	message.SetFrom(s.sender)
	message.SetRecipients([]mailersend.Recipient{recipient})
	message.SetSubject(subject)

	// Construct the path dynamically
	// Construct the path dynamically
	baseDir, _ := os.Getwd()
	s.log.Info("Current working directory", zap.String("baseDir", baseDir))

	// Build the path
	tmplPath := filepath.Join(baseDir, "email", htmlTemplate+".html")
	s.log.Info("Attempting to load template from path", zap.String("path", tmplPath))

	// Parse the template
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Apply template with data
	var body bytes.Buffer
	if err = tmpl.Execute(&body, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	message.SetHTML(body.String())

	var response *mailersend.Response
	response, err = s.Mailer.Email.Send(ctx, message)
	if err != nil {
		return "", fmt.Errorf("failed to send email: %w", err)
	}

	return response.Header.Get("X-Message-Id"), nil
}
