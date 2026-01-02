package model

import (
	"fmt"
	"io"

	"github.com/disintegration/imaging"
	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
)

type Core struct {
	imagePath string
	db        *sqlx.DB
}

func NewCore(db *sqlx.DB) *Core {
	return &Core{
		imagePath: "static/images/tutors/",
		db:        db,
	}
}

func (c *Core) UpdateCookieConsent(accountID, answer string) error {
	query := `
    UPDATE accounts
       SET cookie_consent = $2,
           cookie_consent_at = CURRENT_TIMESTAMP
     WHERE id = $1`

	_, err := c.db.Exec(query, accountID, answer)
	return err
}

// UpdateImage use case.
func (c *Core) UpdateImage(reader io.Reader, tutorID string) error {
	img, err := imaging.Decode(reader, imaging.AutoOrientation(true))
	if err != nil {
		return err
	}

	img = imaging.Fill(img, 300, 300, imaging.Center, imaging.Lanczos)

	name := tutorID + ".jpg"
	if err := imaging.Save(img, c.imagePath+name); err != nil {
		return err
	}

	return c.updateImage(tutorID, name)
}

func (c *Core) updateImage(tutorID, image string) error {
	query := "UPDATE tutors SET image = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2"
	_, err := c.db.Exec(query, image, tutorID)
	return err
}

func (c *Core) AddTutor(tutor Tutor, subjects []string, levels []string) (string, error) {

	tx, err := c.db.Begin()
	if err != nil {
		return "", err
	}

	id := uuid.Must(uuid.NewV4())
	query := `
	INSERT INTO tutors (id, user_id, location_id, online_lessons, description, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`
	_, err = tx.Exec(query, id, tutor.UserID, tutor.LocationID, tutor.OnlineLessons, tutor.Description)
	if err != nil {
		tx.Rollback()
		return "", fmt.Errorf("failed to add tutor %w", err)
	}

	for _, subjectID := range subjects {
		query := `
		INSERT INTO tutor_subjects (tutor_id, subject_id)
			 VALUES ($1, $2)`
		_, err = tx.Exec(query, id, subjectID)
		if err != nil {
			tx.Rollback()
			return "", fmt.Errorf("failed to add subject %w", err)
		}
	}

	for _, levelID := range levels {
		query := `
		INSERT INTO tutor_levels (tutor_id, level_id)
			 VALUES ($1, $2)`
		_, err = tx.Exec(query, id, levelID)
		if err != nil {
			tx.Rollback()
			return "", fmt.Errorf("failed to add level %w", err)
		}
	}

	tx.Commit()
	return id.String(), err

}
