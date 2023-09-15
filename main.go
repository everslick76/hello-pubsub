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
)

func main() {

	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	// log.SetFlags(log.LstdFlags | log.Lmicroseconds)

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
	http.HandleFunc("/request", getPushRequest)
	http.HandleFunc("/publish", publishHandler)
	http.HandleFunc("/concurrency1", concurrency1)
	http.HandleFunc("/concurrency2", concurrency2)

	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal(err)
	}
}

func getPushRequest(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	msg := &pubsub.Message{
		Data: []byte(time.Now().String()),
	}

	json.NewEncoder(w).Encode(msg)
}

func hello(w http.ResponseWriter, r *http.Request) {

	log.Println("Call to hello received")

	fmt.Fprintf(w, "Hello from hello-pubsub!")
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
	
		fmt.Fprint(w, "Message published: " + currentTime)
	}
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
		n, open :=  <-ch
		if !open {
			break
		}
		log.Printf(n)
	}

    defer wg.Done()
}