package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
)

var (
	topic *pubsub.Topic

	// Messages received by this instance.
	messagesMu sync.Mutex
	messages   []string
)

const maxMessages = 10

func main() {

	ctx := context.Background()

	client, err := pubsub.NewClient(ctx, "cloud-core-376009")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	topic = client.Topic("hello")

	// check if the topic exists
	exists, err := topic.Exists(ctx)
	if err != nil || !exists {
		log.Fatal(err)
	}

	http.HandleFunc("/", hello)
	http.HandleFunc("/publish", publishHandler)
	http.HandleFunc("/push", pushHandler)

	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal(err)
	}
}

type pushRequest struct {
	Message struct {
		Attributes map[string]string
		Data       []byte
		ID         string `json:"message_id"`
	}
	Subscription string
}

func hello(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintf(w, "Hello from hello-pubsub!")
}

func pushHandler(w http.ResponseWriter, r *http.Request) {

	msg := &pushRequest{}
	if err := json.NewDecoder(r.Body).Decode(msg); err != nil {
		http.Error(w, fmt.Sprintf("Could not decode body: %v", err), http.StatusBadRequest)
		return
	}

	messagesMu.Lock()
	defer messagesMu.Unlock()
	// Limit to ten.
	messages = append(messages, string(msg.Message.Data))
	if len(messages) > maxMessages {
		messages = messages[len(messages)-maxMessages:]
	}
}

func publishHandler(w http.ResponseWriter, r *http.Request) {

	ctx := context.Background()

	currentTime := time.Now().String()

	msg := &pubsub.Message{
		Data: []byte(currentTime),
	}

	if _, err := topic.Publish(ctx, msg).Get(ctx); err != nil {
		http.Error(w, fmt.Sprintf("Could not publish message: %v", err), 500)
		return
	}

	fmt.Fprint(w, "Message published: "+ currentTime)
}
