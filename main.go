package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"slices"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/r3labs/sse/v2"
)

var (
	topic  *pubsub.Topic
	server *sse.Server
)

func init() {

	setupLogging()
}

func main() {

	// setup sse
	server = sse.New()
	server.AutoReplay = false
	server.CreateStream("messages")

	http.HandleFunc("/", hello)
	http.HandleFunc("/publish", publishHandler)
	http.HandleFunc("/push", pushHandler)
	http.HandleFunc("/events", sseHandler)

	// setup pubsub
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

	// open up for business
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal(err)
	}
}

func sseHandler(w http.ResponseWriter, r *http.Request) {

	origin := r.Header.Get("Origin")
	allowed := []string{"http://localhost:3000", "https://storage.googleapis.com"}
	if slices.Contains(allowed, origin) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}

	go func() {
		// browser disconnect
		<-r.Context().Done()
		println("The client is disconnected here")
		return
	}()

	server.ServeHTTP(w, r)
}

func setupLogging() {

	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
}

func hello(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintf(w, "Hello from hello-pubsub!")
}

type pushRequest struct {
	Message struct {
		Attributes map[string]string
		Data       []byte
		ID         string `json:"message_id"`
	}
	Subscription string
}

type jsonResult struct {
	Message string `json:"message"`
}

func pushHandler(w http.ResponseWriter, r *http.Request) {

	time.Sleep(randomDuration(2, 5))

	msg := &pushRequest{}
	if err := json.NewDecoder(r.Body).Decode(msg); err != nil {
		http.Error(w, fmt.Sprintf("Could not decode body: %v", err), http.StatusBadRequest)
		return
	}

	name := string(msg.Message.Data)
	message := fmt.Sprintf("Message received: %s", name)
	server.Publish("messages", &sse.Event{
		Data: []byte(name),
	})

	log.Printf(message)
}

func publishHandler(w http.ResponseWriter, r *http.Request) {

	n := 1
	requests := r.URL.Query().Get("requests")
	fmt.Sscan(requests, &n)

	ctx := context.Background()

	for i := 1; i <= n; i++ {

		currentTime := time.Now().String()

		msg := &pubsub.Message{
			Data: []byte(currentTime),
		}

		if _, err := topic.Publish(ctx, msg).Get(ctx); err != nil {
			http.Error(w, fmt.Sprintf("Could not publish message: %v", err), http.StatusBadRequest)
			return
		}

		message := "Message published: " + string(msg.Data)
		server.Publish("messages", &sse.Event{
			Data: []byte(message),
		})

		log.Printf(message)
	}

	msg := &jsonResult{
		Message: fmt.Sprintf("Message(s) published: %v", n),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msg)
}

func randomDuration(min int, max int) time.Duration {
	return time.Duration(rand.Intn(max-min)+min) * time.Second
}
