package main

import (
	"log-service/data"
	"net/http"
)

type jsonPayload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

func (app *Config) WriteLog(w http.ResponseWriter, r *http.Request) {
	var requestPayoad jsonPayload
	err := app.readJSON(w, r, &requestPayoad)
	if err != nil {
		app.errorJSON(w, err)
	}

	event := data.LogEntry{
		Name: requestPayoad.Name,
		Data: requestPayoad.Data,
	}

	err = app.Models.LogEntry.Insert(event)
	if err != nil {
		app.errorJSON(w, err)
	}

	resp := jsonResponse{
		Error:   false,
		Message: "Logged",
	}

	app.writeJSON(w, http.StatusAccepted, resp)
}
