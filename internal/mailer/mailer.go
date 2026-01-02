package mailer

import (
	"upforschool/internal/viewer"

	"github.com/jmoiron/sqlx"
)

const (
	from    = "Up For School <no-reply@upforschool.se>"
	charset = "UTF-8"
)

type Service struct {
	db     *sqlx.DB
	viewer *viewer.Viewer
}

func NewService(viewer *viewer.Viewer, db *sqlx.DB) *Service {
	return &Service{
		db:     db,
		viewer: viewer,
	}
}

// Message model.
type message struct {
	to      string
	from    string
	subject string
	body    string
}

func (s *Service) send(m *message) error {
	query := `
	INSERT INTO mails ("to", subject, body, status, sent_at)
		 VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)`
	_, err := s.db.Exec(query, m.to, m.subject, m.body, "sent")
	return err
}
