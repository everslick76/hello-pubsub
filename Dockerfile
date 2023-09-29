FROM golang:1.20-alpine

ENV GO111MODULE=on

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o /hello-pubsub

EXPOSE 8081

CMD [ "/hello-pubsub" ]
