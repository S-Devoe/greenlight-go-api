package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/s-devoe/greenlight-go/internal/data"
	"github.com/s-devoe/greenlight-go/internal/validator"
)

type CreateUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var input CreateUserRequest

	err := app.readJSON(w, r, &input)

	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	ctx := r.Context()

	user := &data.User{
		Name:      input.Name,
		Email:     input.Email,
		Activated: false,
	}
	err = user.Password.Set(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	v := validator.New()

	if data.ValidateUser(v, user); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	err = app.store.Users.Insert(ctx, user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "email address already exists")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	token, err := app.store.Tokens.New(ctx, user.ID, 3*time.Minute, data.ScopeActivation)

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.background(func() {
		data := map[string]interface{}{
			"activationToken": token.Plaintext,
			"userID":          user.ID,
		}

		err = app.mailer.SendMail(user.Email, "user_welcome.tmpl", data)
		if err != nil {
			app.logger.PrintError(err, nil)
		}
	})

	err = app.writeJSON(w, http.StatusAccepted, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

type ResendTokenRequest struct {
	Email string `json:"email"`
}

func (app *application) resendActivationTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input ResendTokenRequest

	err := app.readJSON(w, r, &input)

	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	ctx := r.Context()

	user, err := app.store.Users.GetByEmail(ctx, input.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.badRequestResponse(w, r, errors.New("no user found with this email address"))
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	token, err := app.store.Tokens.New(ctx, user.ID, 3*time.Minute, data.ScopeActivation)

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.background(func() {
		data := map[string]interface{}{
			"activationToken": token.Plaintext,
			"userID":          user.ID,
		}

		err = app.mailer.SendMail(user.Email, "user_welcome.tmpl", data)
		if err != nil {
			app.logger.PrintError(err, nil)
		}
	})
	err = app.writeJSON(w, http.StatusAccepted, envelope{"message": fmt.Sprintf("token sent to your email %s", user.Email)}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}
