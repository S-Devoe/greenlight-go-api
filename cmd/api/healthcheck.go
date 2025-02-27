package main

import (
	"net/http"
)

func (app *application) healthcheckHandler(w http.ResponseWriter, r *http.Request) {

	env := envelope{
		"status": "available",
		"system_info": map[string]string{
			"version":     version,
			"environment": app.config.Env,
		},
	}

	err := app.writeJSON(w, http.StatusOK, envelope{"health": env}, nil)

	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}
