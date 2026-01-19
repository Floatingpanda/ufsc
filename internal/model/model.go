package model

import (
	"database/sql"
	"time"
)

// ContextKey identifier.
type ContextKey string

// ContextKeys.
const (
	ContextKeyProfile ContextKey = "profile"
	ContextKeyTutor   ContextKey = "tutor"
)

type ActiveRole string

const (
	ActiveRoleStudent ActiveRole = "STUDENT"
	ActiveRoleTutor   ActiveRole = "TUTOR"
)

type Location struct {
	ID       int    `db:"id" json:"id"`
	Name     string `db:"name" json:"name"`
	Selected bool
}

type Subject struct {
	ID       int    `db:"id" json:"id"`
	Name     string `db:"name" json:"name"`
	Selected bool
}

type Level struct {
	ID       int    `db:"id" json:"id"`
	Name     string `db:"name" json:"name"`
	Selected bool
}

type User struct {
	ID            string `db:"id" json:"id"`
	FirstName     string `db:"first_name" json:"first_name"`
	LastName      string `db:"last_name" json:"last_name"`
	Email         string `db:"email" json:"email"`
	Phone         string `db:"phone" json:"phone"`
	SMSOptIn      bool   `db:"sms_opt_in" json:"sms_opt_in"`
	Status        string `db:"status" json:"status"`
	CookieConsent string `db:"cookie_consent" json:"cookie_consent"`
	ActiveRole    string `db:"active_role" json:"active_role"`
	IsAdmin       bool   `db:"is_admin" json:"is_admin"`
}

type Tutor struct {
	ID            string    `db:"id"`
	UserID        string    `db:"user_id"`
	Alias         string    `db:"alias"`
	Image         string    `db:"image"`
	OnlineLessons bool      `db:"online_lessons"`
	Bio           string    `db:"description"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`

	Locations []Location
	Subjects  []Subject
	Levels    []Level
}

func (t Tutor) MeetsRequirements(online bool, locationID, subjectID, levelID int) bool {
	if online && !t.OnlineLessons {
		return false
	}

	if !online {
		found := false
		for _, loc := range t.Locations {
			if loc.ID == locationID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	found := false
	for _, subj := range t.Subjects {
		if subj.ID == subjectID {
			found = true
			break
		}
	}
	if !found {

		return false
	}

	found = false
	for _, lev := range t.Levels {
		if lev.ID == levelID {
			found = true
			break
		}
	}
	if !found {

		return false
	}

	return true
}

type TutorView struct {
	ID            string    `db:"id"`
	UserID        string    `db:"user_id"`
	Alias         string    `db:"alias"`
	Image         string    `db:"image"`
	OnlineLessons bool      `db:"online_lessons"`
	Bio           string    `db:"description"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
	FirstName     string    `db:"first_name" json:"first_name"`
	LastName      string    `db:"last_name" json:"last_name"`
	Email         string    `db:"email" json:"email"`
	Phone         string    `db:"phone" json:"phone"`
	SMSOptIn      bool      `db:"sms_opt_in" json:"sms_opt_in"`
	Status        string    `db:"status" json:"status"`

	BookedStatus string `db:"booked_status"`
	Selected     bool   // utility field for selection in UI

	Subjects []Subject
	Levels   []Level
}

type LessonView struct {
	ID           string         `db:"id"`
	StudentID    string         `db:"student_id"`
	StudentName  string         `db:"student_name"`
	SubjectID    int            `db:"subject_id"`
	LevelID      int            `db:"level_id"`
	LocationID   int            `db:"location_id"`
	OnlineLesson bool           `db:"online_lesson"`
	Title        string         `db:"title"`
	Description  string         `db:"description"`
	TutorID      sql.NullString `db:"tutor_id"`
	CreatedAt    time.Time      `db:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at"`
	SubjectName  string         `db:"subject_name"`
	LevelName    string         `db:"level_name"`
	LocationName string         `db:"location_name"`
	BookedStatus string         `db:"booked_status"`
	AcceptedAt   sql.NullTime   `db:"accepted_at"`
	DeletedAt    sql.NullTime   `db:"deleted_at"`
}

type Profile struct {
	User    User
	IsTutor bool
}
