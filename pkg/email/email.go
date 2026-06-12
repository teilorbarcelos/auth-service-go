package email

import (
	"fmt"
	"log"
)

type SendEmailParams struct {
	To      string
	Subject string
	Body    string
	Context map[string]interface{}
}

type Provider interface {
	SendEmail(params SendEmailParams) error
}

type mockEmailProvider struct{}

func NewMockEmailProvider() Provider {
	return &mockEmailProvider{}
}

func (m *mockEmailProvider) SendEmail(params SendEmailParams) error {
	log.Printf("[EmailProvider] Enviando e-mail para: %s", params.To)
	log.Printf("[EmailProvider] Assunto: %s", params.Subject)
	log.Printf("[EmailProvider] Contexto: %v", params.Context)
	
	// Simula a renderização básica para o log
	if token, ok := params.Context["token"]; ok {
		fmt.Printf("\n--- EMAIL RECOVERY ---\nTo: %s\nToken: %v\n----------------------\n\n", params.To, token)
	}
	
	return nil
}
