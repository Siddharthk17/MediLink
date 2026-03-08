package notifications

import (
	"context"
	"time"
)

// EmailService sends transactional emails.
type EmailService interface {
	SendOTP(ctx context.Context, toEmail, fullName, otp, purpose string, lang string) error
	SendBreakGlassNotification(ctx context.Context, req BreakGlassNotification) error
	SendConsentGranted(ctx context.Context, req ConsentNotification) error
	SendConsentRevoked(ctx context.Context, req ConsentNotification) error
	SendWelcomePhysician(ctx context.Context, toEmail, fullName string) error
	SendPhysicianApproved(ctx context.Context, toEmail, fullName string) error
	SendAccountLocked(ctx context.Context, toEmail, fullName string, unlockAt time.Time) error
	SendDocumentProcessingComplete(ctx context.Context, toEmail, fullName, jobID string) error
	SendDocumentProcessingFailed(ctx context.Context, toEmail, fullName, jobID, reason string) error
	SendDocumentNeedsReview(ctx context.Context, toEmail, fullName, jobID string) error
	SendTemplated(ctx context.Context, toEmail, templateName, lang string, data map[string]interface{}) error
}

type BreakGlassNotification struct {
	PatientEmail  string
	PatientName   string
	PhysicianName string
	OrgName       string
	AccessTime    time.Time
	Reason        string
	SupportEmail  string
	Lang          string
}

type ConsentNotification struct {
	ToEmail       string
	ToName        string
	PatientName   string
	PhysicianName string
}

// NoopEmailService is a no-op implementation for testing/dev.
type NoopEmailService struct{}

func (n *NoopEmailService) SendOTP(_ context.Context, _, _, _, _ string, _ string) error { return nil }
func (n *NoopEmailService) SendBreakGlassNotification(_ context.Context, _ BreakGlassNotification) error {
	return nil
}
func (n *NoopEmailService) SendConsentGranted(_ context.Context, _ ConsentNotification) error {
	return nil
}
func (n *NoopEmailService) SendConsentRevoked(_ context.Context, _ ConsentNotification) error {
	return nil
}
func (n *NoopEmailService) SendWelcomePhysician(_ context.Context, _, _ string) error { return nil }
func (n *NoopEmailService) SendPhysicianApproved(_ context.Context, _, _ string) error { return nil }
func (n *NoopEmailService) SendAccountLocked(_ context.Context, _, _ string, _ time.Time) error {
	return nil
}
func (n *NoopEmailService) SendDocumentProcessingComplete(_ context.Context, _, _, _ string) error {
	return nil
}
func (n *NoopEmailService) SendDocumentProcessingFailed(_ context.Context, _, _, _, _ string) error {
	return nil
}
func (n *NoopEmailService) SendDocumentNeedsReview(_ context.Context, _, _, _ string) error {
	return nil
}
func (n *NoopEmailService) SendTemplated(_ context.Context, _, _, _ string, _ map[string]interface{}) error {
	return nil
}
