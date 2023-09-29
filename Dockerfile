FROM golang:1.20-alpine

ENV GO111MODULE=on

EXPOSE 8081

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY . ./

RUN go build -o /hello-pubsub

CMD [ "/hello-pubsub" ]
