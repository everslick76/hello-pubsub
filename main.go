package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/r3labs/sse/v2"
)

var (
	topic *pubsub.Topic
	server *sse.Server
)

func main() {

	// setup sse
	server = sse.New()
	server.CreateStream("messages")

	setupLogging()

	setupRest()

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

func setupRest() {

	http.HandleFunc("/", hello)
	http.HandleFunc("/publish", publishHandler)
	http.HandleFunc("/push", pushHandler)
	http.HandleFunc("/events", sseHandler)
	http.HandleFunc("/request", getPubsubMessage)
	http.HandleFunc("/concurrency1", concurrency1)
	http.HandleFunc("/concurrency2", concurrency2)
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

	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")

	msg := &pushRequest{}
	if err := json.NewDecoder(r.Body).Decode(msg); err != nil {
		http.Error(w, fmt.Sprintf("Could not decode body: %v", err), http.StatusBadRequest)
		return
	}

	message := fmt.Sprintf("Message received: %s", string(msg.Message.Data));
	server.Publish("messages", &sse.Event{
		Data: []byte(message),
	})

	log.Printf(message);
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


func getPubsubMessage(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	msg := &pubsub.Message{
		Data: []byte(time.Now().String()),
	}

	json.NewEncoder(w).Encode(msg)
}

 // testing goroutines, channels and wait groups options

func concurrency1(w http.ResponseWriter, r *http.Request) {

	// one goroutine per request

	n := 1
	requests := r.URL.Query().Get("requests")
	fmt.Sscan(requests, &n)

	ch := make(chan string)

	var wg sync.WaitGroup
	wg.Add(n)

	go print(ch, &wg)

	for i := 1; i <= n; i++ {
		go routine1(i, ch, &wg)
	}

	wg.Wait()

	log.Printf("%s Message(s) published", requests)
}

func routine1(i int, ch chan<- string, wg *sync.WaitGroup) {

	start := time.Now()
	ch <- fmt.Sprintf("Processing message number %d", i)
	time.Sleep(randomDuration(2, 10))
	ch <- fmt.Sprintf("Published message number %d after %s", i, time.Since(start))

	defer wg.Done()
}

func concurrency2(w http.ResponseWriter, r *http.Request) {

	// one goroutine per function

	var n int

	requests := r.URL.Query().Get("requests")
	if _, err := fmt.Sscan(requests, &n); err != nil {
		http.Error(w, fmt.Sprintf("Could not extract number of requests: %v", err), 500)
		return
	}

	ch := make(chan string)

	var wg sync.WaitGroup
	wg.Add(2)

	go print(ch, &wg)

	go routine2(n, ch, &wg)

	wg.Wait()

	log.Printf("%s Message(s) published", requests)
}

func routine2(n int, ch chan<- string, wg *sync.WaitGroup) {

	for i := 1; i <= n; i++ {
		start := time.Now()
		ch <- fmt.Sprintf("Processing message number %d", i)
		time.Sleep(randomDuration(2, 10))
		ch <- fmt.Sprintf("Published message number %d after %s", i, time.Since(start))
	}

	close(ch)
	defer wg.Done()
}

func randomDuration(min int, max int) time.Duration {
	return time.Duration(rand.Intn(max - min) + min) * time.Second
}

func print(ch <-chan string, wg *sync.WaitGroup) {

	for {
		n, open := <-ch
		if !open {
			break
		}
		log.Printf(n)
	}

    defer wg.Done()
}