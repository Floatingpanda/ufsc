package model

import (
	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) User(id string) (*User, error) {
	query := "SELECT id, first_name, last_name, email, phone, sms_opt_in, status, is_admin FROM users WHERE id = $1"
	var u User
	if err := r.db.Get(&u, query, id); err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) UserByEmail(email string) (*User, error) {
	query := "SELECT id, first_name, last_name, email, phone, sms_opt_in, status, is_admin FROM users WHERE email = $1"
	var u User
	if err := r.db.Get(&u, query, email); err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) Tutor(id string) (*Tutor, error) {
	query := "SELECT id, user_id, description, image FROM tutors WHERE id = $1"
	var t Tutor
	if err := r.db.Get(&t, query, id); err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repository) TutorByUserID(userID string) (*Tutor, error) {
	query := "SELECT id, user_id, description, image FROM tutors WHERE user_id = $1"
	var t Tutor
	if err := r.db.Get(&t, query, userID); err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repository) Subjects() ([]Subject, error) {
	query := "SELECT id, name FROM subjects"
	var result []Subject
	if err := r.db.Select(&result, query); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *Repository) Levels() ([]Level, error) {
	query := "SELECT id, name FROM levels"
	var result []Level
	if err := r.db.Select(&result, query); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *Repository) Locations() ([]Location, error) {
	query := "SELECT id, name FROM locations"
	var result []Location
	if err := r.db.Select(&result, query); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *Repository) AllTutors() ([]TutorView, error) {
	query := `
		SELECT 
			t.id,
			t.user_id,
			t.image,
			t.location_id,
			t.online_lessons,
			t.description,
			first_name,
			u.last_name,
			u.email,
			u.phone,
			u.sms_opt_in,
			u.status
		FROM users AS u
		LEFT JOIN tutors AS t ON t.user_id = u.id
		WHERE u.status = 'CONFIRMED'
	`
	var result []TutorView
	if err := r.db.Select(&result, query); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *Repository) Tutors(online_lessons bool, locationID, subjectID, levelID int) ([]TutorView, error) {
	query := `
		SELECT DISTINCT
			t.id,
			t.user_id,
			t.image,
			t.online_lessons,
			t.description,
			first_name,
			u.last_name,
			u.email,
			u.phone,
			u.sms_opt_in,
			u.status
		FROM users AS u
		JOIN tutors AS t ON t.user_id = u.id
		LEFT JOIN tutor_locations loc ON t.id = loc.tutor_id
		JOIN tutor_levels l ON t.id = l.tutor_id
		JOIN tutor_subjects s ON t.id = s.tutor_id
		WHERE u.status = 'CONFIRMED'
		AND ((t.online_lessons AND $1) OR loc.location_id = $2) 
		AND s.subject_id = $3
		AND l.level_id = $4
		
	`
	var result []TutorView
	if err := r.db.Select(&result, query, online_lessons, locationID, subjectID, levelID); err != nil {
		return nil, err
	}

	return result, nil
}
