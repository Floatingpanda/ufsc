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
	query := "SELECT id, first_name, last_name, email, phone, sms_opt_in, status, active_role,is_admin FROM users WHERE id = $1"
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
	query := "SELECT id, user_id, alias, online_lessons, description, image FROM tutors WHERE user_id = $1"
	var t Tutor
	if err := r.db.Get(&t, query, userID); err != nil {
		return nil, err
	}

	err := r.db.Select(&t.Subjects, `
		SELECT s.id, s.name
		  FROM tutor_subjects AS ts
		  JOIN subjects AS s ON ts.subject_id = s.id
		 WHERE ts.tutor_id = $1
	`, t.ID)
	if err != nil {
		return nil, err
	}

	err = r.db.Select(&t.Levels, `
		SELECT l.id, l.name
		  FROM tutor_levels AS tl
		  JOIN levels AS l ON tl.level_id = l.id
		 WHERE tl.tutor_id = $1
	`, t.ID)
	if err != nil {
		return nil, err
	}

	err = r.db.Select(&t.Locations, `
		SELECT loc.id, loc.name
		  FROM tutor_locations AS tl
		  JOIN locations AS loc ON tl.location_id = loc.id
		 WHERE tl.tutor_id = $1
	`, t.ID)
	if err != nil {
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
	query := "SELECT id, name FROM locations where id > 0"
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
			t.alias,
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
			t.alias,
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

func (r *Repository) SentLessonRequests(userID string) ([]LessonView, error) {

	var result []LessonView
	query := `
	SELECT 
		l.id as id,
		l.student_id as student_id,
		u.first_name as student_name,	
		l.subject_id as subject_id,
		l.level_id as level_id,
		l.location_id as location_id,
		l.online_lesson as online_lesson,
		l.title as title,
		l.description as description,
		l.tutor_id as tutor_id,
		l.created_at as created_at,
		l.updated_at as updated_at,
		s."name" as subject_name,
		lvl."name" as level_name,
		loc."name" as location_name,
		COALESCE(req.status, 'PENDING') as booked_status,
		req.accepted_at as accepted_at,
		l.deleted_at as deleted_at
	FROM lessons AS l
		JOIN users AS u ON l.student_id = u.id 
		JOIN subjects as s on l.subject_id = s.id
		JOIN levels as lvl on l.level_id = lvl.id 
		JOIN locations as loc on l.location_id = loc.id 
		LEFT JOIN lesson_requests as req on l.id = req.lesson_id AND req.status = 'ACCEPTED'
	WHERE l.student_id = $1 
	`

	if err := r.db.Select(&result, query, userID); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *Repository) ReceivedLessonRequests(tutorID string) ([]LessonView, error) {
	var result []LessonView
	query := `
	SELECT 
		l.id as id,
		l.student_id as student_id,
		u.first_name as student_name,
		l.subject_id as subject_id,
		l.level_id as level_id,
		l.location_id as location_id,
		l.online_lesson as online_lesson,
		l.title as title,
		l.description as description,
		l.tutor_id as tutor_id,
		l.created_at as created_at,
		l.updated_at as updated_at,
		s."name" as subject_name,
		lvl."name" as level_name,
		loc."name" as location_name,
		COALESCE(req.status, 'PENDING') as booked_status,
		req.accepted_at as accepted_at
	FROM lessons AS l
		JOIN users AS u ON l.student_id = u.id 
		JOIN subjects as s on l.subject_id = s.id
		JOIN levels as lvl on l.level_id = lvl.id 
		JOIN locations as loc on l.location_id = loc.id 
		JOIN lesson_requests as req on l.id = req.lesson_id
	WHERE req.tutor_id = $1
	AND (req.status = 'PENDING' OR l.tutor_id = $1)
	AND l.deleted_at IS NULL
	`

	if err := r.db.Select(&result, query, tutorID); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *Repository) GetLesson(lessonID string) (*LessonView, error) {

	var result LessonView
	query := `
	SELECT 
		l.id as id,
		l.student_id as student_id,
		u.first_name as student_name,	
		l.subject_id as subject_id,
		l.level_id as level_id,
		l.location_id as location_id,
		l.online_lesson as online_lesson,
		l.title as title,
		l.description as description,
		l.tutor_id as tutor_id,
		l.created_at as created_at,
		l.updated_at as updated_at,
		s."name" as subject_name,
		lvl."name" as level_name,
		loc."name" as location_name,
		COALESCE(req.status, 'PENDING') as booked_status,
		req.accepted_at as accepted_at,
		l.deleted_at as deleted_at
	FROM lessons AS l
		JOIN users AS u ON l.student_id = u.id 
		JOIN subjects as s on l.subject_id = s.id
		JOIN levels as lvl on l.level_id = lvl.id 
		JOIN locations as loc on l.location_id = loc.id 
		LEFT JOIN lesson_requests as req on l.id = req.lesson_id AND req.status = 'ACCEPTED'
	WHERE l.id = $1 
	`

	if err := r.db.Get(&result, query, lessonID); err != nil {
		return nil, err
	}
	return &result, nil
}

// func (c *Core) ReceivedLessonRequests(userID string) (string, error) {

// }
