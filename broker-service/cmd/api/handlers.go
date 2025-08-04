package main

import (
	"broker/event"
	"broker/logs"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/rpc"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type RequestPayload struct {
	Action string      `json:"action"`
	Auth   AuthPayload `json:"auth"`
	Log    LogPayload  `json:"log"`
	Mail   MailPayload `json:"mail"`
}

type AuthPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LogPayload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

type MailPayload struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

func (app *Config) Broker(w http.ResponseWriter, r *http.Request) {
	payload := jsonResponse{
		Error:   false,
		Message: "Hello from the broker",
		Data:    nil,
	}

	_ = app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) HandleSubmission(w http.ResponseWriter, r *http.Request) {
	var requestPayload RequestPayload

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	switch requestPayload.Action {
	case "auth":
		app.authenticate(w, requestPayload.Auth)
		return
	case "log":
		app.logItem(w, requestPayload.Log)
		return
	case "log-rpc":
		app.logItemViaRpc(w, requestPayload.Log)
		return
	case "log-grpc":
		app.LogViaGrpc(w, requestPayload.Log)
		return
	case "log-rabbitmq":
		app.logEventViaRabbit(w, requestPayload.Log)
		return
	case "mail":
		app.sendMail(w, requestPayload.Mail)
		return
	default:
		app.errorJSON(w, errors.New("unknown action"), http.StatusBadRequest)
		return
	}
}

func (app *Config) authenticate(w http.ResponseWriter, a AuthPayload) {
	jsonData, err := json.Marshal(a)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	request, err := http.NewRequest("POST", "http://authentication-service/auth", bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusUnauthorized {
		app.errorJSON(w, errors.New("invalid credentials"))
		return
	} else if response.StatusCode != http.StatusAccepted {
		app.errorJSON(w, errors.New("error calling auth service"))
		return
	}

	var jsonFromService jsonResponse
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	if jsonFromService.Error {
		app.errorJSON(w, errors.New(jsonFromService.Message), http.StatusUnauthorized)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "Authenticated"
	payload.Data = jsonFromService.Data
	app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *Config) logItem(w http.ResponseWriter, entry LogPayload) {
	jsonData, err := json.Marshal(entry)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	logServiceUrl := "http://logger-service/log"
	request, err := http.NewRequest("POST", logServiceUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusUnauthorized {
		app.errorJSON(w, errors.New("invalid credentials"))
		return
	} else if response.StatusCode != http.StatusAccepted {
		app.errorJSON(w, errors.New("error calling logger service"))
		return
	}
	var payload jsonResponse
	payload.Error = false
	payload.Message = "logged"
	app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *Config) sendMail(w http.ResponseWriter, msg MailPayload) {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	mailServiceUrl := "http://mail-service/send"
	request, err := http.NewRequest("POST", mailServiceUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted {
		app.errorJSON(w, errors.New("error calling mail service"))
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "Message sent to " + msg.To
	app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *Config) logEventViaRabbit(w http.ResponseWriter, l LogPayload) {
	err := app.PushToQueue(l.Name, l.Data)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "logged via RabbitMQ"

	app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *Config) PushToQueue(name string, msg string) error {
	emitter, err := event.NewEventEmitter(app.Rabbit)
	if err != nil {
		return err
	}

	payload := LogPayload{
		Name: name,
		Data: msg,
	}

	j, err := json.Marshal(&payload)
	if err != nil {
		return err
	}

	err = emitter.Push(string(j), "log.INFO")
	if err != nil {
		return err
	}

	return nil
}

func (app *Config) logItemViaRpc(w http.ResponseWriter, l LogPayload) {
	client, err := rpc.Dial("tcp", "logger-service:5001")
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	var result string
	err = client.Call("RpcServer.LogInfo", l, &result)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: result,
	}
	app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *Config) LogViaGrpc(w http.ResponseWriter, l LogPayload) {
	conn, err := grpc.NewClient("logger-service:50001", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer conn.Close()

	c := logs.NewLogServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = c.WriteLog(ctx, &logs.LogRequest{
		LogEntry: &logs.Log{
			Name: l.Name,
			Data: l.Data,
		},
	})
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "logged"
	app.writeJSON(w, http.StatusAccepted, payload)
}
