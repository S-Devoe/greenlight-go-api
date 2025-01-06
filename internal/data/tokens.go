package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/s-devoe/greenlight-go/internal/validator"
)

const (
	ScopeActivation = "activation"
)

type Token struct {
	Plaintext string
	Hash      []byte
	UserId    int64
	Expiry    time.Time
	Scope     string
}
type TokenStore struct {
	DB *pgxpool.Pool
}

func (s *TokenStore) New(ctx context.Context, userID int64, ttl time.Duration, scope string) (*Token, error) {
	token, err := generateToken(userID, ttl, scope)
	if err != nil {
		return nil, err
	}
	err = s.Insert(ctx, token)
	return token, err
}

func (s *TokenStore) Insert(ctx context.Context, token *Token) error {
	stmt := `INSERT INTO tokens (hash, user_id, expiry, scope)
    VALUES ($1, $2, $3, $4)`

	args := []interface{}{
		token.Hash,
		token.UserId,
		token.Expiry,
		token.Scope,
	}

	c, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := s.DB.Exec(c, stmt, args...)
	return err

}

func (s *TokenStore) DeleteAllForUser(ctx context.Context, scope string, userID int64) error {
	stmt := `DELETE FROM tokens WHERE scope = $1 AND user_id = $2`

	c, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := s.DB.Exec(c, stmt, scope, userID)
	return err

}

func ValidateToken(v *validator.Validator, tokenPlaintext string) {
	v.Check(tokenPlaintext != "", "token", "must be provided")
	v.Check(len(tokenPlaintext) == 26, "token", "must be 26 characters long")
}

func generateToken(userId int64, ttl time.Duration, scope string) (*Token, error) {

	token := &Token{
		UserId: userId,
		Expiry: time.Now().Add(ttl),
		Scope:  scope,
	}

	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)

	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]

	return token, nil
}
