package mailer

import (
	"bytes"
	"fmt"
	"log"
)

// SignupCodeMessage result.
func (s *Service) signupCode(email, id, code string) (*message, error) {
	log.Println("signupCode", email, id, code)

	template := "email-info.html"
	title := "Sista steget innan du kan börja använda tjänsten!"
	content := "Fyll i denna verifieringskod för att slutföra din registrering och börja använda Up For Sports:"
	subject := "Verifieringskod till Up For Sports"
	preview := ""

	page := s.viewer.
		Page(template).
		Add("Preview", preview).
		Add("Title", title).
		Add("Content", []string{content, code}).
		Add("Link", "/auth/confirm/"+id)

	var buff bytes.Buffer
	if err := s.viewer.Execute(&buff, page); err != nil {
		return nil, fmt.Errorf("could not create signp view: %w", err)
	}

	return &message{
		to:      email,
		from:    from,
		subject: subject,
		body:    buff.String(),
	}, nil
}

// SendSignupCode mail.
func (s *Service) SendSignupCode(email, id, code string) {
	msg, err := s.signupCode(email, id, code)
	if err != nil {
		log.Printf("unable to create signup message: %v", err)
		return
	}

	if err := s.send(msg); err != nil {
		log.Printf("unable to send signup message: %v", err)
	}
}
