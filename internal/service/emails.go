package service

import (
	"fmt"

	"github.com/zhashkevych/creatly-backend/internal/config"
	emailProvider "github.com/zhashkevych/creatly-backend/pkg/email"
)

const (
	nameField            = "name"
	verificationLinkTmpl = "https://%s/verification?code=%s" // https://<school host>/verification?code=<verification_code>
)

type EmailService struct {
	provider emailProvider.Provider
	sender   emailProvider.Sender
	config   config.EmailConfig
}

// Structures used for templates.
type verificationEmailInput struct {
	VerificationLink string
}

type purchaseSuccessfulEmailInput struct {
	Name       string
	CourseName string
}

func NewEmailsService(provider emailProvider.Provider, sender emailProvider.Sender, config config.EmailConfig) *EmailService {
	return &EmailService{provider: provider, sender: sender, config: config}
}

func (s *EmailService) AddToList(name, email string) error {
	return s.provider.AddEmailToList(emailProvider.AddEmailInput{
		Email:  email,
		ListID: s.config.SendPulse.ListID,
		Variables: map[string]string{
			nameField: name,
		},
	})
}

func (s *EmailService) SendStudentVerificationEmail(input VerificationEmailInput) error {
	subject := fmt.Sprintf(s.config.Subjects.Verification, input.Name)

	templateInput := verificationEmailInput{s.createVerificationLink(input.Domain, input.VerificationCode)}
	sendInput := emailProvider.SendEmailInput{Subject: subject, To: input.Email}

	if err := sendInput.GenerateBodyFromHTML(s.config.Templates.Verification, templateInput); err != nil {
		return err
	}

	return s.sender.Send(sendInput)
}

func (s *EmailService) SendStudentPurchaseSuccessfulEmail(input StudentPurchaseSuccessfulEmailInput) error {
	templateInput := purchaseSuccessfulEmailInput{Name: input.Name, CourseName: input.CourseName}
	sendInput := emailProvider.SendEmailInput{Subject: s.config.Subjects.PurchaseSuccessful, To: input.Email}

	if err := sendInput.GenerateBodyFromHTML(s.config.Templates.PurchaseSuccessful, templateInput); err != nil {
		return err
	}

	return s.sender.Send(sendInput)
}

func (s *EmailService) SendUserVerificationEmail(input VerificationEmailInput) error {
	// todo implement
	return nil
}

func (s *EmailService) createVerificationLink(domain, code string) string {
	return fmt.Sprintf(verificationLinkTmpl, domain, code)
}
