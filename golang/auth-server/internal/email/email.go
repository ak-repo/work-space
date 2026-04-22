// Package email defines the Sender interface and ships a ConsoleSender
// that logs emails to stdout — useful for local development.
//
// Production options to implement:
//   - SMTP (net/smtp)
//   - SendGrid  (github.com/sendgrid/sendgrid-go)
//   - AWS SES   (github.com/aws/aws-sdk-go-v2/service/ses)
//   - Resend    (github.com/resendlabs/resend-go)
package email

import (
	"context"
	"fmt"
	"log/slog"
)

// Message is a simple email payload.
type Message struct {
	To      string
	Subject string
	HTML    string
	Text    string
}

// Sender sends a single email message.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

// ── Console sender ─────────────────────────────────────────────────────────

// ConsoleSender prints emails to stdout. Default for dev/test.
type ConsoleSender struct{}

func (ConsoleSender) Send(_ context.Context, msg Message) error {
	slog.Info("📧 [email]",
		slog.String("to", msg.To),
		slog.String("subject", msg.Subject),
	)
	fmt.Printf("--- EMAIL ---\nTo: %s\nSubject: %s\n\n%s\n-------------\n",
		msg.To, msg.Subject, msg.Text)
	return nil
}

// ── Template helpers ───────────────────────────────────────────────────────

// VerifyEmailMessage builds a verification email.
func VerifyEmailMessage(to, verifyURL string) Message {
	return Message{
		To:      to,
		Subject: "Verify your email address",
		Text:    fmt.Sprintf("Click the link to verify your email:\n\n%s\n\nThis link expires in 24 hours.", verifyURL),
		HTML:    fmt.Sprintf(`<p>Click the link below to verify your email:</p><p><a href="%s">Verify Email</a></p><p>This link expires in 24 hours.</p>`, verifyURL),
	}
}

// ResetPasswordMessage builds a password-reset email.
func ResetPasswordMessage(to, resetURL string) Message {
	return Message{
		To:      to,
		Subject: "Reset your password",
		Text:    fmt.Sprintf("Click the link to reset your password:\n\n%s\n\nThis link expires in 1 hour.", resetURL),
		HTML:    fmt.Sprintf(`<p>Click the link below to reset your password:</p><p><a href="%s">Reset Password</a></p><p>This link expires in 1 hour.</p>`, resetURL),
	}
}
