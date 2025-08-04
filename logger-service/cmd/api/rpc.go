package main

import (
	"context"
	"fmt"
	"log"
	"log-service/data"
	"net"
	"net/rpc"
	"time"
)

type RpcServer struct{}

type RpcPayload struct {
	Name string
	Data string
}

func (r *RpcServer) LogInfo(payload RpcPayload, resp *string) error {
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

	*resp = "Processed payload via RPC: " + payload.Name
	return nil
}

func (app *Config) rpcListen() error {
	log.Println("Starting RPC server on port ", rpcPort)
	listen, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%s", rpcPort))
	if err != nil {
		return err
	}
	defer listen.Close()

	for {
		rpcConn, err := listen.Accept()
		if err != nil {
			continue
		}

		go rpc.ServeConn(rpcConn)
	}
}
