FROM golang:1.20-alpine

EXPOSE 8081

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go get ./...

RUN go mod download

COPY . ./

RUN go build -o /hello-pubsub

CMD [ "/hello-pubsub" ]
