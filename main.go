package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"

	"cloud.google.com/go/pubsub"
)

var (
	topic *pubsub.Topic

	// Messages received by this instance.
	messagesMu sync.Mutex
	messages   []string

	// token is used to verify push requests.
	// token = mustGetenv("PUBSUB_VERIFICATION_TOKEN")
)

const maxMessages = 10

func main() {
	ctx := context.Background()

	client, err := pubsub.NewClient(ctx, "cloud-core-376009")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	topicName := "hello"
	topic = client.Topic(topicName)

	log.Printf("Topic is %s", topic.String())

	// Create the topic if it doesn't exist.
	exists, err := topic.Exists(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if !exists {
		log.Printf("Topic %v doesn't exist - creating it", topicName)
		_, err = client.CreateTopic(ctx, topicName)
		if err != nil {
			log.Fatal(err)
		}
	}

	http.HandleFunc("/", listHandler)
	http.HandleFunc("/pubsub/publish", publishHandler)
	http.HandleFunc("/pubsub/push", pushHandler)

	port := "8081"
	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

// func mustGetenv(k string) string {
// 	v := os.Getenv(k)
// 	if v == "" {
// 		log.Fatalf("%s environment variable not set.", k)
// 	}
// 	return v
// }

type pushRequest struct {
	Message struct {
		Attributes map[string]string
		Data       []byte
		ID         string `json:"message_id"`
	}
	Subscription string
}

func pushHandler(w http.ResponseWriter, r *http.Request) {
	// Verify the token.
	// if r.URL.Query().Get("token") != token {
	// 	http.Error(w, "Bad token", http.StatusBadRequest)
	// 	return
	// }
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

func listHandler(w http.ResponseWriter, r *http.Request) {
	messagesMu.Lock()
	defer messagesMu.Unlock()

	if err := tmpl.Execute(w, messages); err != nil {
		log.Printf("Could not execute template: %v", err)
	}
}

func publishHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	msg := &pubsub.Message{
		Data: []byte(r.FormValue("payload")),
	}

	if _, err := topic.Publish(ctx, msg).Get(ctx); err != nil {
		http.Error(w, fmt.Sprintf("Could not publish message: %v", err), 500)
		return
	}

	fmt.Fprint(w, "Message published.")
}

var tmpl = template.Must(template.New("").Parse(`<!DOCTYPE html>
<html>
  <head>
    <title>Pub/Sub</title>
  </head>
  <body>
    <div>
      <p>Last ten messages received by this instance:</p>
      <ul>
      {{ range . }}
          <li>{{ . }}</li>
      {{ end }}
      </ul>
    </div>
    <form method="post" action="/pubsub/publish">
      <textarea name="payload" placeholder="Enter message here"></textarea>
      <input type="submit">
    </form>
    <p>Note: if the application is running across multiple instances, each
      instance will have its own list of messages.</p>
  </body>
</html>`))

// package main

// import (
// 	"context"
// 	"fmt"
// 	"log"
// 	"net/http"

// 	"cloud.google.com/go/pubsub"
// 	"github.com/gin-gonic/gin"
// )

// var (
// 	topic *pubsub.Topic
// )

// func main() {

// 	// rest
// 	router := gin.Default()

// 	router.GET("/", hello)
// 	router.GET("/publish/:msg", publish)
	
// 	err := router.Run(":8081")
// 	if err != nil {
// 		panic("[Error] failed to start Gin server due to: " + err.Error())
// 	}

// 	//pubsub
// 	ctx := context.Background()

// 	client, err := pubsub.NewClient(ctx, "cloud-core-376009")
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer client.Close()

// 	topicName := "projects/cloud-core-376009/topics/hello"
// 	topic = client.Topic(topicName)

// 	// check if the topic exists
// 	exists, err := topic.Exists(ctx)
// 	if err != nil || !exists {
// 		log.Fatal(err)
// 	} else {
// 		fmt.Println("Topic exists: " + topic.String())
// 	}
// }

// func hello(c *gin.Context) {
// 	c.IndentedJSON(http.StatusOK, "Hello from hello-pubsub!")
// }

// func publish(c *gin.Context) {

// 	msg := c.Param("msg")

// 	fmt.Println("Publishing " + msg)

// 	ctx := context.Background()

// 	pubsubMsg := &pubsub.Message{
// 		Data: []byte(msg),
// 	}

// 	if ctx == nil {
// 		fmt.Println("context is nil")
// 		c.IndentedJSON(http.StatusBadRequest, msg)
// 	} else {
// 		if _, err := topic.Publish(ctx, pubsubMsg).Get(ctx); err != nil {
// 			c.IndentedJSON(http.StatusInternalServerError, msg)
// 			return
// 		}
	
// 		fmt.Println("Published " + msg)
	
// 		c.IndentedJSON(http.StatusOK, msg)
// 	}
// }
