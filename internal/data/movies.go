package data

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/s-devoe/greenlight-go/internal/validator"
)

type Movie struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"-"`
	Title     string    `json:"title"`
	Year      int32     `json:"year,omitempty"`
	// Runtime   int32     `json:"-"`
	Runtime Runtime  `json:"-"`
	Genres  []string `json:"genres,omitempty"`
	Version int32    `json:"version"`
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
