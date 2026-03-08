package notifications

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"time"

	"github.com/resend/resend-go/v2"
	"github.com/rs/zerolog"
)

//go:embed templates/*/*.html
var templateFS embed.FS

type ResendEmailService struct {
	client    *resend.Client
	fromAddr  string
	logger    zerolog.Logger
	templates map[string]*template.Template
}

func NewResendEmailService(apiKey, fromAddr string, logger zerolog.Logger) *ResendEmailService {
	svc := &ResendEmailService{
		client:    resend.NewClient(apiKey),
		fromAddr:  fromAddr,
		logger:    logger,
		templates: make(map[string]*template.Template),
	}
	svc.loadTemplates()
	return svc
}

func (s *ResendEmailService) loadTemplates() {
	templatePaths := []string{
		"templates/otp/en.html",
		"templates/otp/hi.html",
		"templates/otp/mr.html",
		"templates/break_glass/en.html",
		"templates/break_glass/hi.html",
		"templates/break_glass/mr.html",
		"templates/consent_granted/en.html",
		"templates/consent_revoked/en.html",
		"templates/welcome_physician/en.html",
		"templates/physician_approved/en.html",
		"templates/account_locked/en.html",
		"templates/account_locked/hi.html",
		"templates/account_locked/mr.html",
		"templates/document_complete/en.html",
		"templates/document_complete/hi.html",
		"templates/document_complete/mr.html",
		"templates/document_failed/en.html",
		"templates/document_failed/hi.html",
		"templates/document_failed/mr.html",
		"templates/new_prescription/en.html",
		"templates/lab_result_ready/en.html",
		"templates/lab_result_ready/hi.html",
		"templates/lab_result_ready/mr.html",
		"templates/consent_request/en.html",
	}

	for _, path := range templatePaths {
		data, err := templateFS.ReadFile(path)
		if err != nil {
			s.logger.Warn().Str("template", path).Err(err).Msg("failed to load email template")
			continue
		}
		tmpl, err := template.New(path).Parse(string(data))
		if err != nil {
			s.logger.Warn().Str("template", path).Err(err).Msg("failed to parse email template")
			continue
		}
		s.templates[path] = tmpl
	}
}

func (s *ResendEmailService) renderTemplate(templatePath string, data interface{}) (string, error) {
	tmpl, ok := s.templates[templatePath]
	if !ok {
		return "", fmt.Errorf("template not found: %s", templatePath)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}
	return buf.String(), nil
}

func (s *ResendEmailService) sendEmail(ctx context.Context, to, subject, html string) error {
	params := &resend.SendEmailRequest{
		From:    s.fromAddr,
		To:      []string{to},
		Subject: subject,
		Html:    html,
	}

	_, err := s.client.Emails.SendWithContext(ctx, params)
	if err != nil {
		s.logger.Error().Err(err).Str("to", to).Str("subject", subject).Msg("failed to send email")
		return err
	}
	return nil
}

func (s *ResendEmailService) SendOTP(ctx context.Context, toEmail, fullName, otp, purpose string, lang string) error {
	if lang == "" {
		lang = "en"
	}
	templatePath := fmt.Sprintf("templates/otp/%s.html", lang)

	data := map[string]string{
		"FullName":  fullName,
		"OTP":       otp,
		"ExpiresIn": "10 minutes",
		"Purpose":   purpose,
	}

	html, err := s.renderTemplate(templatePath, data)
	if err != nil {
		// Fallback to English
		html, err = s.renderTemplate("templates/otp/en.html", data)
		if err != nil {
			return err
		}
	}

	subject := "Your MediLink Verification Code"
	return s.sendEmail(ctx, toEmail, subject, html)
}

func (s *ResendEmailService) SendBreakGlassNotification(ctx context.Context, req BreakGlassNotification) error {
	lang := req.Lang
	if lang == "" {
		lang = "en"
	}
	templatePath := fmt.Sprintf("templates/break_glass/%s.html", lang)

	data := map[string]string{
		"PatientName":   req.PatientName,
		"PhysicianName": req.PhysicianName,
		"OrgName":       req.OrgName,
		"AccessTime":    req.AccessTime.Format(time.RFC1123),
		"Reason":        req.Reason,
		"SupportEmail":  req.SupportEmail,
	}

	html, err := s.renderTemplate(templatePath, data)
	if err != nil {
		html, err = s.renderTemplate("templates/break_glass/en.html", data)
		if err != nil {
			return err
		}
	}

	return s.sendEmail(ctx, req.PatientEmail, "Emergency Access to Your Medical Records", html)
}

func (s *ResendEmailService) SendConsentGranted(ctx context.Context, req ConsentNotification) error {
	data := map[string]string{
		"ToName":        req.ToName,
		"PatientName":   req.PatientName,
		"PhysicianName": req.PhysicianName,
	}

	html, err := s.renderTemplate("templates/consent_granted/en.html", data)
	if err != nil {
		return err
	}

	return s.sendEmail(ctx, req.ToEmail, "Consent Granted - MediLink", html)
}

func (s *ResendEmailService) SendConsentRevoked(ctx context.Context, req ConsentNotification) error {
	data := map[string]string{
		"ToName":        req.ToName,
		"PatientName":   req.PatientName,
		"PhysicianName": req.PhysicianName,
	}

	html, err := s.renderTemplate("templates/consent_revoked/en.html", data)
	if err != nil {
		return err
	}

	return s.sendEmail(ctx, req.ToEmail, "Consent Revoked - MediLink", html)
}

func (s *ResendEmailService) SendWelcomePhysician(ctx context.Context, toEmail, fullName string) error {
	data := map[string]string{"FullName": fullName}
	html, err := s.renderTemplate("templates/welcome_physician/en.html", data)
	if err != nil {
		return err
	}
	return s.sendEmail(ctx, toEmail, "Welcome to MediLink - Account Pending Verification", html)
}

func (s *ResendEmailService) SendPhysicianApproved(ctx context.Context, toEmail, fullName string) error {
	data := map[string]string{"FullName": fullName}
	html, err := s.renderTemplate("templates/physician_approved/en.html", data)
	if err != nil {
		return err
	}
	return s.sendEmail(ctx, toEmail, "MediLink Account Approved", html)
}

func (s *ResendEmailService) SendAccountLocked(ctx context.Context, toEmail, fullName string, unlockAt time.Time) error {
	data := map[string]string{
		"FullName": fullName,
		"UnlockAt": unlockAt.Format(time.RFC1123),
	}
	html, err := s.renderTemplate("templates/account_locked/en.html", data)
	if err != nil {
		return err
	}
	return s.sendEmail(ctx, toEmail, "MediLink Account Security Alert", html)
}

func (s *ResendEmailService) SendDocumentProcessingComplete(ctx context.Context, toEmail, fullName, jobID string) error {
	s.logger.Info().Str("job_id", jobID).Str("email", toEmail).Msg("document processing complete notification")
	return s.sendEmail(ctx, toEmail, "MediLink - Lab Report Processed",
		fmt.Sprintf("<p>Hello %s,</p><p>Your lab report (Job ID: %s) has been processed successfully. You can view the results in your MediLink dashboard.</p>", fullName, jobID))
}

func (s *ResendEmailService) SendDocumentProcessingFailed(ctx context.Context, toEmail, fullName, jobID, reason string) error {
	s.logger.Warn().Str("job_id", jobID).Str("email", toEmail).Str("reason", reason).Msg("document processing failed notification")
	return s.sendEmail(ctx, toEmail, "MediLink - Lab Report Processing Issue",
		fmt.Sprintf("<p>Hello %s,</p><p>We encountered an issue processing your lab report (Job ID: %s). Reason: %s. Please try re-uploading or contact support.</p>", fullName, jobID, reason))
}

func (s *ResendEmailService) SendDocumentNeedsReview(ctx context.Context, toEmail, fullName, jobID string) error {
	s.logger.Info().Str("job_id", jobID).Str("email", toEmail).Msg("document needs manual review notification")
	return s.sendEmail(ctx, toEmail, "MediLink - Lab Report Needs Review",
		fmt.Sprintf("<p>Hello %s,</p><p>Your lab report (Job ID: %s) has been partially processed and requires manual review. A clinician will review it shortly.</p>", fullName, jobID))
}

// subjectForTemplate returns a subject line for a given template name.
func subjectForTemplate(name string) string {
	switch name {
	case "document_complete":
		return "MediLink - Lab Report Processed"
	case "document_failed":
		return "MediLink - Lab Report Processing Issue"
	case "new_prescription":
		return "MediLink - New Prescription"
	case "lab_result_ready":
		return "MediLink - Lab Results Ready"
	case "consent_request":
		return "MediLink - Consent Request"
	default:
		return "MediLink Notification"
	}
}

// SendTemplated sends an email using a named template with arbitrary data.
func (s *ResendEmailService) SendTemplated(ctx context.Context, toEmail, templateName, lang string, data map[string]interface{}) error {
	if lang == "" {
		lang = "en"
	}
	templatePath := fmt.Sprintf("templates/%s/%s.html", templateName, lang)

	html, err := s.renderTemplate(templatePath, data)
	if err != nil {
		// Fallback to English
		html, err = s.renderTemplate(fmt.Sprintf("templates/%s/en.html", templateName), data)
		if err != nil {
			return fmt.Errorf("render template %s: %w", templateName, err)
		}
	}

	return s.sendEmail(ctx, toEmail, subjectForTemplate(templateName), html)
}
