package upforauth

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

var emptyID = uuid.Nil.String()

// Service for auth.
type Service struct {
	db *sqlx.DB
}

// User model.
type User struct {
	ID     string  `db:"id"`
	Email  string  `db:"email"`
	Status string  `db:"status"`
	SSN    *string `db:"ssn"`
}

// UserByEmail result.
func (s *Service) UserByEmail(email string) (*User, error) {
	query := `
    SELECT u.id,
           u.email,
           u.status,
		   u.ssn
      FROM users AS u
     WHERE u.email = $1`

	var u User
	return &u, s.db.Get(&u, query, email)
}

// UserByEmail result.
func (s *Service) UserBySSN(ssn string) (*User, error) {
	query := `
    SELECT u.id,
           u.email,
           u.status
      FROM users AS u
     WHERE u.ssn = $1`

	var u User
	return &u, s.db.Get(&u, query, ssn)
}

type user struct {
	ID       string `db:"id"`
	Password string `db:"password"`
	Status   string `db:"status"`
}

// Login and return the login ID.
func (s *Service) LoginSSN(ssn string) (string, error) {
	ssn = strings.TrimSpace(strings.ToLower(ssn))

	if len(ssn) != 12 {
		return emptyID, errors.New("bad SSN")
	}

	query := `
	SELECT u.id, u.status
	  FROM users AS u
	 WHERE u.ssn = $1`

	var u user
	if err := s.db.Get(&u, query, ssn); err != nil {
		return emptyID, fmt.Errorf("unauthenticated: %w", err)
	}

	if u.Status != "CONFIRMED" {
		return emptyID, errors.New("unauthenticated: not confirmed")
	}

	loginID, err := uuid.NewV4()
	if err != nil {
		return emptyID, fmt.Errorf("unauthenticated: %w", err)
	}

	insert := "INSERT INTO logins (id, user_id) VALUES($1, $2);"
	if _, err := s.db.Exec(insert, loginID, u.ID); err != nil {
		return emptyID, fmt.Errorf("unauthenticated: %w", err)
	}

	return loginID.String(), nil
}

// Login and return the login ID.
func (s *Service) Login(username, password string) (string, error) {
	username = strings.TrimSpace(strings.ToLower(username))

	if len(username) == 0 || len(password) == 0 || !strings.Contains(username, "@") {
		log.Println("-1")
		return emptyID, errors.New("unauthented")
	}

	query := `
	SELECT u.id, u.password, u.status
	  FROM users AS u
	 WHERE u.email = $1`

	var u user
	if err := s.db.Get(&u, query, username); err != nil {
		log.Println("-1")
		return emptyID, fmt.Errorf("unauthenticated: %w", err)
	}

	if len(u.Password) == 0 {
		log.Println("-2")
		return emptyID, errors.New("unauthented")
	}

	if u.Status != "CONFIRMED" {
		log.Println("-3")
		return emptyID, errors.New("unauthenticated: not confirmed")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)); err != nil {
		log.Println("-4")
		return emptyID, fmt.Errorf("unauthenticated: %w", err)
	}

	loginID, err := uuid.NewV4()
	if err != nil {
		log.Println("-5")
		return emptyID, fmt.Errorf("unauthenticated: %w", err)
	}

	insert := "INSERT INTO logins (id, user_id) VALUES($1, $2);"
	if _, err := s.db.Exec(insert, loginID, u.ID); err != nil {
		log.Println("-6")
		return emptyID, fmt.Errorf("unauthenticated: %w", err)
	}

	return loginID.String(), nil
}

// Logout login with ID.
func (s *Service) Logout(loginID string) error {
	query := "DELETE FROM logins WHERE id = $1"
	_, err := s.db.Exec(query, loginID)
	return err
}

// UserID result.
func (s *Service) UserID(loginID string) (string, error) {
	query := `
    SELECT u.id FROM logins AS l
      JOIN users AS u ON l.user_id = u.id
     WHERE l.id = $1`

	var userID string
	if err := s.db.Get(&userID, query, loginID); err != nil {
		return userID, err
	}

	update := "UPDATE logins SET updated_at = CURRENT_TIMESTAMP WHERE id = $1"
	_, err := s.db.Exec(update, loginID)
	return userID, err
}

// User with login ID.
func (s *Service) User(loginID string) (*User, error) {
	query := `
    SELECT u.id AS id,
           u.email AS email,
           u.status AS status
      FROM logins AS l
      JOIN users AS u
        ON l.user_id = u.id
     WHERE l.id = $1`

	var user User
	if err := s.db.Get(&user, query, loginID); err != nil {
		return nil, err
	}

	update := "UPDATE logins SET updated_at = CURRENT_TIMESTAMP WHERE id = $1"
	_, err := s.db.Exec(update, loginID)
	return &user, err
}

// Token model.
type Token struct {
	ID        string    `db:"id"`
	UserID    string    `db:"user_id"`
	UserEmail string    `db:"user_email"`
	Name      string    `db:"name"`
	Value     string    `db:"value"`
	Counter   int64     `db:"counter"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Valid if counter less and five and less than 24 hours old.
func (t *Token) Valid() bool {
	if t.Counter > 5 {
		return false
	}

	now := time.Now().UTC()
	dur := now.Sub(t.CreatedAt)
	return dur.Hours() <= 24
}

func (s *Service) addUser(firstname, lastname, email, phone, password string, ssn *string, smsOptIn bool) (string, error) {
	email = strings.ToLower(email)

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return emptyID, err
	}

	userID, err := uuid.NewV4()
	if err != nil {
		return emptyID, err
	}

	query := `
	INSERT INTO users ( id, first_name, last_name, email, phone, password, ssn, sms_opt_in) 
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err = s.db.Exec(query, userID, firstname, lastname, email, phone, hash, ssn, smsOptIn)
	if err != nil {
		return emptyID, err
	}

	return userID.String(), nil
}

var max = big.NewInt(9999)

// AddToken with name.
func (s *Service) AddToken(userID, name string) (*Token, error) {
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return nil, err
	}

	value := fmt.Sprintf("%04d", n.Int64())

	hash, err := bcrypt.GenerateFromPassword([]byte(value), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	tokenID, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	query := `
	INSERT INTO tokens (id, user_id, name, value)
	VALUES ($1, $2, $3, $4)`

	_, err = s.db.Exec(query, tokenID, userID, name, hash)
	if err != nil {
		return nil, err
	}

	t, err := s.TokenByID(tokenID.String())
	if err != nil {
		return nil, err
	}

	t.Value = value
	return t, nil
}

// TokenByID result.
func (s *Service) TokenByID(id string) (*Token, error) {
	query := `
    SELECT t.id AS id,
           t.name AS name,
           t.counter AS counter,
           t.value AS value,
           t.created_at AS created_at,
           t.updated_at AS updated_at,
           u.id AS user_id,
           u.email AS user_email
      FROM tokens AS t
      JOIN users as u
        ON u.id = t.user_id
      WHERE t.id = $1`

	var t Token
	return &t, s.db.Get(&t, query, id)
}

// AddUser to auth.
func (s *Service) AddUser(firstname, lastname, email, phone, password string, ssn *string, smsOptIn bool) (*Token, error) {
	userID, err := s.addUser(firstname, lastname, email, phone, password, ssn, smsOptIn)
	if err != nil {
		return nil, err
	}

	return s.AddToken(userID, "CONFIRMATION")
}

// RemoveUser with user ID.
func (s *Service) RemoveUser(userID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM tokens WHERE user_id = $1", userID)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec("DELETE FROM logins WHERE user_id = $1", userID)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec("DELETE FROM users WHERE id= $1", userID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (s *Service) incrementTokenCounter(tokenID string) error {
	query := `
    UPDATE tokens
       SET counter = counter + 1,
           updated_at = CURRENT_TIMESTAMP
     WHERE id = $1`

	_, err := s.db.Exec(query, tokenID)
	return err
}

func (s *Service) isConfirmed(userID string) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1 AND status = 'CONFIRMED')"
	var exists bool
	return exists, s.db.Get(&exists, query, userID)
}

var (
	// ErrTokenInvalid is returend if token counter exceeds limits.
	ErrTokenInvalid = errors.New("token has expired")
)

// Confirm user with id.
// No error is returned if the user is CONFIRMED.
func (s *Service) Confirm(t *Token, value string) error {
	confirmed, err := s.isConfirmed(t.UserID)
	if err != nil {
		return err
	}

	if confirmed {
		return nil
	}

	if !t.Valid() {
		return ErrTokenInvalid
	}

	if err := s.incrementTokenCounter(t.ID); err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(t.Value), []byte(value)); err != nil {
		return err
	}

	setConfirmed := `
	UPDATE users
	   SET status = 'CONFIRMED',
	       updated_at = CURRENT_TIMESTAMP
	 WHERE id = $1`

	_, err = s.db.Exec(setConfirmed, t.UserID)
	return err
}

// UpdatePassword for user with ID.
func (s *Service) UpdatePassword(id, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	query := `
    UPDATE users
       SET password = $1,
           updated_at = CURRENT_TIMESTAMP
     WHERE id = $2`

	_, err = s.db.Exec(query, string(hash), id)
	return err
}

// ResetPassword for user with token.
func (s *Service) ResetPassword(tokenID, password string) error {
	var userID string
	query := "SELECT t.user_id FROM tokens AS t WHERE t.id = $1"
	if err := s.db.Get(&userID, query, tokenID); err != nil {
		return err
	}

	return s.UpdatePassword(userID, password)
}

func (s *Service) UpdateEmail(id, email string) error {
	email = strings.TrimSpace(strings.ToLower(email))

	query := `
    UPDATE users
       SET email = $1,
           updated_at = CURRENT_TIMESTAMP
    WHERE id = $2`

	_, err := s.db.Exec(query, email, id)
	return err
}

func (s *Service) UpdateSSN(id, ssn string) error {

	ssn = strings.ReplaceAll(ssn, "-", "")

	_, err := strconv.Atoi(ssn)

	if len(ssn) > 12 || err != nil {
		return errors.New("bad SSN")
	}

	query := `
    UPDATE users
       SET ssn = $2,
           updated_at = CURRENT_TIMESTAMP
    WHERE id = $1`

	_, err = s.db.Exec(query, id, ssn)
	return err
}

// ValidResetID verifies that id exists.
func (s *Service) ValidResetID(id string) error {
	var t Token
	query := `
    SELECT t.id, t.user_id,
           t.name,
           t.value,
           t.counter,
           t.created_at,
           t.updated_at
	  FROM tokens AS t
	 WHERE t.id = $1
	   AND t.name = 'RESET'`

	if err := s.db.Get(&t, query, id); err != nil {
		return err
	}

	if !t.Valid() {
		return ErrTokenInvalid
	}

	return s.incrementTokenCounter(t.ID)
}

// UserByID result.
func (s *Service) UserByID(id string) (*User, error) {
	query := `
    SELECT u.id,
           u.email,
           u.status
      FROM users AS u
     WHERE u.id = $1
       AND u.status = 'CONFIRMED'`

	var u User
	return &u, s.db.Get(&u, query, id)
}

// UserByID result.
func (s *Service) UserByID_AnyStatus(id string) (*User, error) {
	query := `
    SELECT u.id,
           u.email,
           u.status
      FROM users AS u
     WHERE u.id = $1`

	var u User
	return &u, s.db.Get(&u, query, id)
}

// UsersByID result.
func (s *Service) UsersByID(ids ...string) ([]User, error) {
	query := `
    SELECT u.id,
           u.email,
           u.status
      FROM users AS u
     WHERE u.id IN (?)
       AND u.status = 'CONFIRMED'`

	query, args, err := sqlx.In(query, ids)
	if err != nil {
		return nil, err
	}
	query = s.db.Rebind(query)

	u := make([]User, 0)
	return u, s.db.Select(&u, query, args...)
}

// New service.
func New(db *sqlx.DB) *Service {
	return &Service{db: db}
}
