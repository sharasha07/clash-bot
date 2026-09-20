package main

import "net/http"

func (app *application) health(w http.ResponseWriter, r *http.Request) {
	err := app.writeJSON(w, http.StatusOK, envelope{"status": "available"})
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}
