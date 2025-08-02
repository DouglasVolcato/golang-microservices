package main

import (
	"fmt"
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	rabbitConn, err := connect()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer rabbitConn.Close()
	log.Panicln("Connected to RabbitMQ")
}

func connect() (*amqp.Connection, error) {
	var counts = 0
	var backOff = time.Duration(counts) * time.Second
	var connection *amqp.Connection

	for {
		c, err := amqp.Dial("amqp://guest:guest@localhost")
		if err != nil {
			fmt.Println("RabbitMQ not yet ready...")
			counts++
		} else {
			connection = c
			break
		}

		if counts > 5 {
			fmt.Println(err)
			return nil, err
		}

		backOff = time.Duration(counts) * time.Second
		fmt.Println(backOff)
		log.Println("backing off")
		time.Sleep(backOff)
		continue
	}

	return connection, nil
}
