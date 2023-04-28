package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"cloud.google.com/go/pubsub"
	"github.com/gin-gonic/gin"
)

var (
	topic *pubsub.Topic
)

func main() {

	// rest
	router := gin.Default()

	router.GET("/", hello)
	router.GET("/publish/:msg", publish)
	
	err := router.Run(":8081")
	if err != nil {
		panic("[Error] failed to start Gin server due to: " + err.Error())
	}

	//pubsub
	ctx := context.Background()

	client, err := pubsub.NewClient(ctx, "cloud-core-376009")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	topicName := "projects/cloud-core-376009/topics/hello"
	topic = client.Topic(topicName)

	// Create the topic if it doesn't exist.
	exists, err := topic.Exists(ctx)
	if err != nil || !exists {
		log.Fatal(err)
	}
}

func hello(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, "Hello from hello-pubsub!")
}

func publish(c *gin.Context) {

	ctx := context.Background()
	msg := c.Param("msg")

	pubsubMsg := &pubsub.Message{
		Data: []byte(c.Param("msg")),
	}

	if _, err := topic.Publish(ctx, pubsubMsg).Get(ctx); err != nil {
		c.IndentedJSON(http.StatusInternalServerError, msg)
		return
	}

	fmt.Println("Published " + msg)

	c.IndentedJSON(http.StatusOK, msg)
}
