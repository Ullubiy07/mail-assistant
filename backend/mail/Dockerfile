# Build stage
FROM golang:alpine3.22 AS builder

WORKDIR /code

COPY go.mod go.sum .
RUN go mod download

COPY mail ./mail
COPY pkg ./pkg

RUN go build -o app mail/cmd/main.go

# Run stage
FROM alpine:3.20

WORKDIR /code

COPY --from=builder /code/app .

CMD ["./app"]