package data

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRecordNotFound    = errors.New("record not found")
	PgxErrRecordNotFound = pgx.ErrNoRows
	ErrUpdateConflict    = errors.New("update conflict")
)

type Store struct {
	Movies interface {
		Insert(ctx context.Context, movie *Movie) error
		Get(ctx context.Context, id int64) (*Movie, error)
		Update(ctx context.Context, movie *Movie) error
		Delete(ctx context.Context, id int64) error
	}
}

func NewStore(db *pgxpool.Pool) Store {
	return Store{
		Movies: MovieStore{DB: db},
	}
}

func NewMockStore() Store {
	return Store{
		Movies: MockMovieStore{},
	}
}
