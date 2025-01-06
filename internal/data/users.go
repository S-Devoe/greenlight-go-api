package data

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/s-devoe/greenlight-go/internal/validator"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  password  `json:"-"`
	Activated bool      `json:"activated"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}

type MockUserStore struct{}

type password struct {
	plaintext *string
	hash      []byte
}

var (
	ErrDuplicateEmail = errors.New("duplicate email")
)

type UserStore struct {
	DB *pgxpool.Pool
}

func (s *UserStore) Insert(ctx context.Context, user *User) error {
	stmt := `
    INSERT INTO users (name, email, password_hash, activated)
    VALUES ($1, $2, $3, $4)
    RETURNING id, created_at, version
    `

	args := []interface{}{
		user.Name,
		user.Email,
		user.Password.hash,
		user.Activated,
	}

	c, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	err := s.DB.QueryRow(c, stmt, args...).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Version,
	)

	if err != nil {

		switch {
		case ErrorCode(err) == UniqueViolation:
			return ErrDuplicateEmail
		default:
			return err
		}
	}

	return nil
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (*User, error) {
	stmt := `
    SELECT id, name, email, password_hash, activated, version, created_at
    FROM users
    WHERE email = $1
    `
	var user User
	c, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	err := s.DB.QueryRow(c, stmt, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password.hash,
		&user.Activated,
		&user.Version,
		&user.CreatedAt,
	)

	if err != nil {
		switch {
		case errors.Is(err, PgxErrRecordNotFound):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &user, nil
}

func (s *UserStore) UpdateUser(ctx context.Context, user *User) error {
	stmt := `
    UPDATE users
    SET name = $1, email = $2, password_hash = $3, activated = $4, version = version + 1
    WHERE id = $5 AND version = $6
    RETURNING version
    `
	args := []interface{}{
		user.Name,
		user.Email,
		user.Password.hash,
		user.Activated,
		user.ID,
		user.Version,
	}
	c, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	err := s.DB.QueryRow(c, stmt, args...).Scan(&user.Version)

	if err != nil {

		switch {
		case ErrorCode(err) == UniqueViolation:
			return ErrDuplicateEmail
		case errors.Is(err, PgxErrRecordNotFound):
			return ErrUpdateConflict
		default:
			return err
		}
	}

	return nil
}

func ValidateEmail(v *validator.Validator, email string) {
	v.Check(email != "", "email", "please enter a valid email")
	v.Check(validator.Macthes(email, validator.EmailRegex), "email", "please enter a valid email address")

}

func ValidatePasswordPlaintext(v *validator.Validator, password string) {
	v.Check(password != "", "password", "password must be provided")
	v.Check(len(password) > 5, "password", "password must be at least 6 characters")
	v.Check(len(password) <= 20, "password", "password must be at most 20 characters")
}

func ValidateUser(v *validator.Validator, user *User) {
	v.Check(user.Name != "", "name", "name must be provided")
	v.Check(len(user.Name) <= 100, "name", "name must not be more than 100 characters")
	ValidateEmail(v, user.Email)

	if user.Password.plaintext != nil {
		ValidatePasswordPlaintext(v, *user.Password.plaintext)
	}

	if user.Password.hash == nil {
		panic("missing password hash for user")
	}
}

func (p *password) Set(plaintextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12)
	if err != nil {
		return err
	}

	p.plaintext = &plaintextPassword
	p.hash = hash

	return nil
}

func (p *password) Matches(plaintextPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(plaintextPassword))

	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}

	}
	return true, nil
}
