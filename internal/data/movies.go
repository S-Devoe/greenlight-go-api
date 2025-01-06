package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/s-devoe/greenlight-go/internal/validator"
)

type Movie struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Year  int32  `json:"year,omitempty"`
	// Runtime   int32     `json:"-"`
	Runtime   Runtime   `json:"-"`
	Genres    []string  `json:"genres,omitempty"`
	Version   int32     `json:"version"`
	CreatedAt time.Time `json:"-"`
}

type MovieStore struct {
	DB *pgxpool.Pool
}

type MockMovieStore struct{}

func (m MovieStore) GetAll(ctx context.Context, title string, genres []string, filters Filters) ([]*Movie, Metadata, error) {
	stmt := fmt.Sprintf(`SELECT count(*) OVER(), id, title, year, runtime, genres, version 
	FROM movies 
	WHERE (STRPOS(LOWER(title), LOWER($1)) > 0 OR $1='') 
	AND (genres @>$2 OR $2 ='{}')
	ORDER BY %s %s, id  ASC
	LIMIT $3
	OFFSET $4`, filters.sortColumn(), filters.sortDirection())

	c, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	args := []interface{}{
		title,
		genres,
		filters.limit(),
		filters.offset(),
	}

	rows, err := m.DB.Query(c, stmt, args...)
	if err != nil {
		return nil, Metadata{}, err
	}

	defer rows.Close()

	totalRecords := 0
	movies := []*Movie{}

	for rows.Next() {
		var movie Movie

		err := rows.Scan(
			&totalRecords,
			&movie.ID,
			&movie.Title,
			&movie.Year,
			&movie.Runtime,
			&movie.Genres,
			&movie.Version,
		)
		if err != nil {
			return nil, Metadata{}, err
		}

		movies = append(movies, &movie)
	}
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return movies, metadata, nil
}
func (m MovieStore) Insert(ctx context.Context, movie *Movie) error {
	stmt := `INSERT INTO movies (title, year, runtime, genres)
	VALUES ($1, $2, $3, $4)
	RETURNING id, created_at, version`

	c, cancel := context.WithTimeout(ctx, 10*time.Second)

	defer cancel()

	args := []interface{}{
		movie.Title,
		movie.Year,
		movie.Runtime,
		movie.Genres,
	}
	return m.DB.QueryRow(c, stmt, args...).Scan(&movie.ID, &movie.CreatedAt, &movie.Version)

}

func (m MovieStore) Get(ctx context.Context, id int64) (*Movie, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}
	stmt := `SELECT id, title, year, runtime, genres, version, created_at 
	FROM movies
	WHERE id = $1`

	c, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()

	var movie Movie
	row := m.DB.QueryRow(c, stmt, id)

	err := row.Scan(
		&[]byte{},
		&movie.ID,
		&movie.Title,
		&movie.Year,
		&movie.Runtime,
		&movie.Genres,
		&movie.Version,
		&movie.CreatedAt,
	)

	if err != nil {
		switch {
		case errors.Is(err, PgxErrRecordNotFound):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &movie, nil
}

func (m MovieStore) Update(ctx context.Context, movie *Movie) error {
	stmt := `
	UPDATE movies
	SET title = $1, year = $2, runtime = $3, genres = $4, version = version + 1
	WHERE id = $5 AND version = $6
	RETURNING version`
	args := []interface{}{
		movie.Title,
		movie.Year,
		movie.Runtime,
		movie.Genres,
		movie.ID,
		movie.Version,
	}

	c, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err := m.DB.QueryRow(c, stmt, args...).Scan(&movie.Version)

	if err != nil {
		switch {
		case errors.Is(err, PgxErrRecordNotFound):
			return ErrUpdateConflict
		default:
			return err
		}

	}
	return nil
}

func (m MovieStore) Delete(ctx context.Context, id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}

	stmt := `DELETE FROM movies WHERE id = $1`
	// if  in the future i am wondering why i am using different contexts for the methods here, check Let's Go Further Chapter 8 last paragraph.
	c, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	result, err := m.DB.Exec(c, stmt, id)

	if err != nil {
		return err
	}
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrRecordNotFound
	}
	return nil
}

// mock methods for unit testing
func (m MockMovieStore) Insert(ctx context.Context, movie *Movie) error {
	return nil
}

func (m MockMovieStore) Get(ctx context.Context, id int64) (*Movie, error) {
	return nil, nil
}

func (m MockMovieStore) Update(ctx context.Context, movie *Movie) error {
	return nil
}

func (m MockMovieStore) Delete(ctx context.Context, id int64) error {
	return nil
}

func (m MockMovieStore) GetAll(ctx context.Context, title string, genres []string, filters Filters) ([]*Movie, Metadata, error) {
	return nil, Metadata{}, nil
}

// custom JSON Marshal for coverting runtime for the client JSON response
func (m Movie) MarshalJSON() ([]byte, error) {
	var runtime string
	if m.Runtime != 0 {
		runtime = fmt.Sprintf("%d mins", m.Runtime)
	}

	type MovieAlias Movie

	aux := struct {
		MovieAlias
		Runtime string `json:"runtime,omitempty"`
	}{
		MovieAlias: MovieAlias(m),
		Runtime:    runtime,
	}

	return json.Marshal(aux)
}

func ValidateMovie(v *validator.Validator, movie *Movie) {
	v.Check(movie.Title != "", "title", "title must be provided")
	v.Check(len(movie.Title) < 500, "title", "title must be less than 500 characters")
	v.Check(movie.Year >= 1880, "year", "year must be greater than 1879")
	v.Check(movie.Year <= int32(time.Now().Year()), "year", "year must not be in the future")
	v.Check(movie.Runtime != 0, "runtime", "runtime must be a positive number")
	v.Check(movie.Runtime > 0, "runtime", "runtime must be greater than 0")
	v.Check(len(movie.Genres) > 0, "genres", "at least one genre must be provided")
	v.Check(movie.Genres != nil, "genres", "genres must be provided")
	v.Check(len(movie.Genres) <= 5, "genre", "genre must not contain more than 5 genre")
	v.Check(validator.Unique(movie.Genres), "genre", "genre must not contain duplicate values")
}
