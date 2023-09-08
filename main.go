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
)

var (
	topic *pubsub.Topic

	// Messages received by this instance.
	messagesMu sync.Mutex
	messages   []string
)

const maxMessages = 10

func main() {

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	// ctx := context.Background()

	// client, err := pubsub.NewClient(ctx, "cloud-core-376009")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer client.Close()

	// topic = client.Topic("hello")

	// // check if the topic exists
	// exists, err := topic.Exists(ctx)
	// if err != nil || !exists {
	// 	log.Fatal(err)
	// }

	http.HandleFunc("/", hello)
	http.HandleFunc("/publish", publishHandler)
	http.HandleFunc("/concurrency1", concurrency1)
	http.HandleFunc("/concurrency2", concurrency2)
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

	log.Println("Call to hello received")

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
		http.Error(w, fmt.Sprintf("Could not publish message: %v", err), http.StatusBadRequest)
		return
	}

	fmt.Fprint(w, "Message published: "+ currentTime)
}

 // testing goroutines, channels and wait groups options

func concurrency1(w http.ResponseWriter, r *http.Request) {

	// one goroutine per request

	var n int

	requests := r.URL.Query().Get("requests")
	if _, err := fmt.Sscan(requests, &n); err != nil {
		http.Error(w, fmt.Sprintf("Could not extract number of requests: %v", err), 500)
		return
	}

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
		n, open :=  <-ch
		if !open {
			break
		}
		log.Printf(n)
	}

    defer wg.Done()
}