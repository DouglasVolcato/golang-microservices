package main

import (
	"context"
	"log"
	"log-service/data"
	"time"
)

type RpcServer struct{}

type RpcPayload struct {
	Name string
	Data string
}

func (r *RpcServer) LongInfo(payload RpcPayload, resp *string) error {
	collection := client.Database("logs").Collection("logs")

	_, err := collection.InsertOne(context.TODO(), data.LogEntry{
		Name:      payload.Name,
		Data:      payload.Data,
		CreatedAt: time.Now(),
	})

	if err != nil {
		log.Println("error writing to mongo", err)
		return err
	}

	*resp = "Processd payload via RPC: " + payload.Name
	return nil
}
