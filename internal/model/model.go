package model

import "time"

// ContextKey identifier.
type ContextKey string

// ContextKeys.
const (
	ContextKeyProfile ContextKey = "profile"
	ContextKeyTutor   ContextKey = "tutor"
)

type Location struct {
	ID   int64  `db:"id" json:"id"`
	Name string `db:"name" json:"name"`
}

type Subject struct {
	ID   int64  `db:"id" json:"id"`
	Name string `db:"name" json:"name"`
}

type Level struct {
	ID   int64  `db:"id" json:"id"`
	Name string `db:"name" json:"name"`
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
	IsAdmin       bool   `db:"is_admin" json:"is_admin"`
}

type Tutor struct {
	ID            string    `db:"id"`
	UserID        string    `db:"user_id"`
	Image         string    `db:"image"`
	OnlineLessons bool      `db:"online_lessons"`
	Description   string    `db:"description"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`

	Subjects []Subject
	Levels   []Level
}

type TutorView struct {
	ID            string    `db:"id"`
	UserID        string    `db:"user_id"`
	Image         string    `db:"image"`
	OnlineLessons bool      `db:"online_lessons"`
	Description   string    `db:"description"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
	FirstName     string    `db:"first_name" json:"first_name"`
	LastName      string    `db:"last_name" json:"last_name"`
	Email         string    `db:"email" json:"email"`
	Phone         string    `db:"phone" json:"phone"`
	SMSOptIn      bool      `db:"sms_opt_in" json:"sms_opt_in"`
	Status        string    `db:"status" json:"status"`

	Subjects []Subject
	Levels   []Level
}

type Profile struct {
	User    User
	IsTutor bool
}
