package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	router.GET("/publish/:msg", publish)
	router.Run("localhost:8081")
}

func publish(c *gin.Context) {

	msg := c.Param("msg")

	fmt.Println("Publishing: " + msg)

	agent := NewAgent()

	go agent.Publish("projects/cloud-core-376009/topics/hello", msg)

	agent.Close()

	c.IndentedJSON(http.StatusOK, msg)
}

type Agent struct {
	mu    sync.Mutex
	subs  map[string][]chan string
	quit  chan struct{}
	closed bool
}

func NewAgent() *Agent {
	return &Agent{
		subs: make(map[string][]chan string),
		quit: make(chan struct{}),
	}
}

func (b *Agent) Publish(topic string, msg string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}

	for _, ch := range b.subs[topic] {
		ch <- msg
	}
}

func (b *Agent) Subscribe(topic string) <-chan string {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}

	ch := make(chan string)
	b.subs[topic] = append(b.subs[topic], ch)
	return ch
}

func (b *Agent) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}

	b.closed = true
	close(b.quit)

	for _, ch := range b.subs {
		for _, sub := range ch {
			close(sub)
		}
	}
}