package model

import (
	"fmt"
	"io"
	"log"
	"upforschool/internal/pkg/worldline"

	"github.com/disintegration/imaging"
	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
)

type Core struct {
	imagePath string
	db        *sqlx.DB

	worldline *worldline.Worldline
}

func NewCore(db *sqlx.DB, worldline *worldline.Worldline) *Core {
	return &Core{
		imagePath: "static/images/tutors/",
		db:        db,
		worldline: worldline,
	}
}

func (c *Core) UpdateCookieConsent(userID, answer string) error {
	query := `
    UPDATE users
       SET cookie_consent = $2,
           cookie_consent_at = CURRENT_TIMESTAMP
     WHERE id = $1`

	_, err := c.db.Exec(query, userID, answer)
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

func (c *Core) UpdateProfile(userID string, firstname, lastname, phone string, smsOptIn bool, locations, subjects, levels []string, bio string, onlineLessons bool, tutorID string) error {
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}

	// Update user info
	query := `UPDATE users SET first_name = $1, last_name = $2, phone = $3, sms_opt_in = $4, updated_at = CURRENT_TIMESTAMP WHERE id = $5`
	_, err = tx.Exec(query, firstname, lastname, phone, smsOptIn, userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update user: %w", err)
	}

	// If tutor, update tutor profile and relationships
	if tutorID != "" {
		// Update tutor bio and online lessons
		query = `UPDATE tutors SET description = $1, online_lessons = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3`
		_, err = tx.Exec(query, bio, onlineLessons, tutorID)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to update tutor: %w", err)
		}

		// Delete existing relationships
		_, err = tx.Exec("DELETE FROM tutor_locations WHERE tutor_id = $1", tutorID)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to delete locations: %w", err)
		}

		_, err = tx.Exec("DELETE FROM tutor_subjects WHERE tutor_id = $1", tutorID)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to delete subjects: %w", err)
		}

		_, err = tx.Exec("DELETE FROM tutor_levels WHERE tutor_id = $1", tutorID)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to delete levels: %w", err)
		}

		// Add new locations
		for _, locationID := range locations {
			_, err = tx.Exec("INSERT INTO tutor_locations (tutor_id, location_id) VALUES ($1, $2)", tutorID, locationID)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to add location: %w", err)
			}
		}

		// Add new subjects
		for _, subjectID := range subjects {
			_, err = tx.Exec("INSERT INTO tutor_subjects (tutor_id, subject_id) VALUES ($1, $2)", tutorID, subjectID)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to add subject: %w", err)
			}
		}

		// Add new levels
		for _, levelID := range levels {
			_, err = tx.Exec("INSERT INTO tutor_levels (tutor_id, level_id) VALUES ($1, $2)", tutorID, levelID)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to add level: %w", err)
			}
		}
	}

	return tx.Commit()
}

func (c *Core) AddTutor(tutor Tutor, locations []string, subjects []string, levels []string) (string, error) {

	tx, err := c.db.Begin()
	if err != nil {
		return "", err
	}

	id := uuid.Must(uuid.NewV4())
	query := `
	INSERT INTO tutors (id, user_id, alias, online_lessons, description, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`
	_, err = tx.Exec(query, id, tutor.UserID, tutor.Alias, tutor.OnlineLessons, tutor.Bio)
	if err != nil {
		tx.Rollback()
		return "", fmt.Errorf("failed to add tutor %w", err)
	}

	for _, locationID := range locations {
		query := `
		INSERT INTO tutor_locations (tutor_id, location_id)
			 VALUES ($1, $2)`
		_, err = tx.Exec(query, id, locationID)
		if err != nil {
			tx.Rollback()
			return "", fmt.Errorf("failed to add level %w", err)
		}
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

type LessonRequest struct {
	SubjectID   string   `json:"subject"`
	LevelID     string   `json:"level"`
	LocationID  string   `json:"location"`
	IsOnline    bool     `json:"isOnline"`
	Tutors      []string `json:"tutors"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Duration    int      `json:"duration"` // minutes
}

func (c *Core) AddLesson(userID string, r LessonRequest) (string, error) {

	if r.LocationID == "online" {
		r.LocationID = "-1"
	}

	log.Printf("%+v", r)
	tx, err := c.db.Begin()
	if err != nil {
		return "", err
	}

	id := uuid.Must(uuid.NewV4())

	query := `
	INSERT INTO lessons (
		id,
		student_id,
		subject_id,
		level_id,
		location_id,
		online_lesson,
		title,
		description
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err = tx.Exec(query, id, userID, r.SubjectID, r.LevelID, r.LocationID, r.IsOnline, r.Title, r.Description)
	if err != nil {
		tx.Rollback()
		return "", fmt.Errorf("failed to add lesson %w", err)
	}

	for _, tutorID := range r.Tutors {
		query := `
		INSERT INTO lesson_requests (lesson_id, tutor_id, status, created_at, updated_at)
			 VALUES ($1, $2, 'PENDING', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`
		_, err = tx.Exec(query, id, tutorID)
		if err != nil {
			tx.Rollback()
			return "", fmt.Errorf("failed to add lesson request %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return "", err
	}

	return id.String(), nil
}

func (c *Core) UpdateLesson(lessonID string, r LessonRequest) error {

	if r.LocationID == "online" {
		r.LocationID = "-1"
	}

	tx, err := c.db.Begin()
	if err != nil {
		return err
	}

	query := `
	UPDATE lessons
	SET 
	    title = $2,
	    description = $3,
	    updated_at = CURRENT_TIMESTAMP
	WHERE id = $1`

	_, err = tx.Exec(query, lessonID, r.Title, r.Description)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update lesson %w", err)
	}

	// Delete existing lesson requests
	// _, err = tx.Exec("DELETE FROM lesson_requests WHERE lesson_id = $1", lessonID)
	// if err != nil {
	// 	tx.Rollback()
	// 	return fmt.Errorf("failed to delete old lesson requests %w", err)
	// }

	// Add new lesson requests
	for _, tutorID := range r.Tutors {
		log.Println("adding lesson request for tutor:", lessonID, tutorID)
		query := `
		INSERT INTO lesson_requests (lesson_id, tutor_id, status, created_at, updated_at)
			 VALUES ($1, $2, 'PENDING', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`
		_, err = tx.Exec(query, lessonID, tutorID)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to add lesson request %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (c *Core) AcceptLesson(lessonID, tutorID string) (string, error) {

	tx, err := c.db.Begin()
	if err != nil {
		return "", err
	}

	query := `
	UPDATE lesson_requests
	   SET status = 'ACCEPTED',
	       updated_at = CURRENT_TIMESTAMP,
		   accepted_at = CURRENT_TIMESTAMP
	 WHERE lesson_id = $1 AND tutor_id = $2`
	_, err = tx.Exec(query, lessonID, tutorID)
	if err != nil {
		tx.Rollback()
		return "", fmt.Errorf("failed to accept lesson %w", err)
	}

	_, err = tx.Exec(`UPDATE lessons
	   SET tutor_id = $1,
	       updated_at = CURRENT_TIMESTAMP
	 WHERE id = $2`, tutorID, lessonID)
	if err != nil {
		tx.Rollback()
		return "", fmt.Errorf("failed to update lesson %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return "", err
	}

	return lessonID, nil
}

func (c *Core) DeleteLesson(lessonID string) error {
	_, err := c.db.Exec("UPDATE lessons set deleted_at = NOW() where id = $1", lessonID)
	return err
}

// SwitchOrganization use cae.
func (c *Core) SwitchRole(userID string, role ActiveRole) error {
	_, err := c.db.Exec("UPDATE users SET active_role = $1 WHERE id = $2", role, userID)
	return err
}
