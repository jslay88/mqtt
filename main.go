package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Configuration constants
const (
	broker = "tcp://mqtt-service:1883" // Update this to mqtt-service:1883 if running in Kubernetes
	topic  = "test/topic"
	qos    = 1
)

var clientID = os.Getenv("HOSTNAME") + "go-mqtt-client"

// MessageHandler handles incoming MQTT messages
func onMessageReceived(client mqtt.Client, message mqtt.Message) {
	fmt.Printf("Received message: %s\n", string(message.Payload()))
	fmt.Printf("Topic: %s\n", message.Topic())
}

func main() {
	// WaitGroup to keep the program running
	var wg sync.WaitGroup
	wg.Add(2) // For subscriber and publisher goroutines

	// Channel to handle OS signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// MQTT client options
	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(clientID).
		SetDefaultPublishHandler(onMessageReceived).
		SetConnectionLostHandler(func(client mqtt.Client, err error) {
			log.Printf("Connection lost: %v", err)
		})

	// Create MQTT client
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Error connecting to MQTT broker: %v", token.Error())
	}
	defer client.Disconnect(250)

	// Subscriber goroutine
	go func() {
		defer wg.Done()
		if token := client.Subscribe(topic, qos, nil); token.Wait() && token.Error() != nil {
			log.Printf("Error subscribing to topic: %v", token.Error())
			return
		}
		log.Printf("Subscribed to topic: %s", topic)

		// Keep goroutine alive until signal
		<-sigs
		if token := client.Unsubscribe(topic); token.Wait() && token.Error() != nil {
			log.Printf("Error unsubscribing: %v", token.Error())
		}
	}()

	// Publisher goroutine
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		counter := 0
		for {
			select {
			case <-ticker.C:
				message := fmt.Sprintf("Message %d from Go client: %s", counter, clientID)
				token := client.Publish(topic, qos, false, message)
				token.Wait()
				if token.Error() != nil {
					log.Printf("Error publishing message: %v", token.Error())
				} else {
					log.Printf("Published: %s", message)
				}
				counter++
			case <-sigs:
				return
			}
		}
	}()

	// Wait for signal to exit
	<-sigs
	log.Println("Shutting down...")
	wg.Wait()
}

// To build and run:
// 1. go get github.com/eclipse/paho.mqtt.golang
// 2. go build -o mqtt-client
// 3. ./mqtt-client
