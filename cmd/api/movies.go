package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/s-devoe/greenlight-go/internal/data"
	"github.com/s-devoe/greenlight-go/internal/validator"
)

type CreateMovieRequest struct {
	Title   string       `json:"title"`
	Year    int32        `json:"year"`
	Runtime data.Runtime `json:"runtime"`
	Genres  []string     `json:"genres"`
}

func (app *application) createMovieHandler(w http.ResponseWriter, r *http.Request) {

	var input CreateMovieRequest
	err := app.readJSON(w, r, &input)

	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	movie := &data.Movie{
		Title:   input.Title,
		Genres:  input.Genres,
		Runtime: input.Runtime,
		Year:    input.Year,
	}

	v := validator.New()
	if data.ValidateMovie(v, movie); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	ctx := r.Context()
	err = app.store.Movies.Insert(ctx, movie)

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/movies/%d", movie.ID))

	err = app.writeJSON(w, http.StatusCreated, envelope{"movies": movie}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
func (app *application) showMovieHandler(w http.ResponseWriter, r *http.Request) {

	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	ctx := r.Context()

	movie, err := app.store.Movies.Get(ctx, id)

	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"movie": movie}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}

type UpdateMovieRequest struct {
	Title   *string       `json:"title"`
	Year    *int32        `json:"year"`
	Runtime *data.Runtime `json:"runtime"`
	Genres  []string      `json:"genres"`
}

func (app *application) updateMovieHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	var input UpdateMovieRequest
	err = app.readJSON(w, r, &input)

	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	ctx := r.Context()
	movie, err := app.store.Movies.Get(ctx, id)

	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	if input.Title != nil {
		movie.Title = *input.Title
	}
	if input.Genres != nil {
		movie.Genres = input.Genres
	}
	if input.Runtime != nil {
		movie.Runtime = *input.Runtime
	}
	if input.Year != nil {
		movie.Year = *input.Year
	}

	v := validator.New()
	if data.ValidateMovie(v, movie); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	err = app.store.Movies.Update(ctx, movie)
	if err != nil {

		switch {
		case errors.Is(err, data.ErrUpdateConflict):
			app.updateConflictResponse(w, r)
		default:

			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"movie": movie}, nil)

	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) deleteMovieHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	ctx := r.Context()
	err = app.store.Movies.Delete(ctx, id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"message": "Movie deleted successfully"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

type listMovieParams struct {
	Title        string
	Genres       []string
	data.Filters // add the pagination types here
}

func (app *application) listMoviesHandler(w http.ResponseWriter, r *http.Request) {
	var params listMovieParams

	v := validator.New()
	qs := r.URL.Query()

	params.Title = app.readString(qs, "title", "")
	params.Genres = app.readCSV(qs, "genres", []string{})
	params.Filters.Page = app.readInt(qs, "page", 1, v)
	params.Filters.PageSize = app.readInt(qs, "page_size", 10, v)
	params.Filters.Sort = app.readString(qs, "sort", "id")
	params.Filters.SortSafeList = []string{"id", "title", "year", "runtime", "-id", "-title", "-runtime", "-year"}

	if data.ValidateFilters(v, params.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	ctx := r.Context()

	movies, metadata, err := app.store.Movies.GetAll(ctx, params.Title, params.Genres, params.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"movies": movies, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

}
