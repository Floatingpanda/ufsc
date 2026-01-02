package database

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	// Database driver.
	_ "github.com/lib/pq"
)

// Rebind transforms a query from named parameters to the DB driver's bindvar type.
// This function also supports IN-queries.
func Rebind(db *sqlx.DB, query string, arg interface{}) (string, []interface{}, error) {
	query, args, err := sqlx.Named(query, arg)
	if err != nil {
		return query, args, err
	}
	query, args, err = sqlx.In(query, args...)
	if err != nil {
		return query, args, err
	}

	query = db.Rebind(query)
	return query, args, err
}

// Config options.
type Config struct {
	Host string `json:"host"`
	Port int    `json:"port"`
	Name string `json:"name"`
	User string `json:"user"`
	Pass string `json:"pass"`
}

func (c *Config) dataSource() string {
	format := "host=%s port=%d user=%s password=%s dbname=%s sslmode=disable"
	return fmt.Sprintf(format, c.Host, c.Port, c.User, c.Pass, c.Name)
}

// New DB.
func New(c *Config) (*sqlx.DB, error) {
	return sqlx.Connect("postgres", c.dataSource())
}

// NewLocal connects to localhost.
func NewLocal(dbname, user, password string) (*sqlx.DB, error) {
	c := &Config{
		Host: "127.0.0.1",
		Port: 5432,
		Name: dbname,
		User: user,
		Pass: password,
	}

	return New(c)
}
