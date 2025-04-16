FROM golang:1.23.4-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main .

EXPOSE 8080

ENV ELASTICSEARCH_URL=http://elasticsearch:9200

CMD ["./main"]