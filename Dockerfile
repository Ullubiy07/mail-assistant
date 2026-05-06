# Build stage
FROM golang:alpine3.22 AS builder

WORKDIR /code

COPY go.mod go.sum .
RUN go mod download

COPY . .

RUN go build -o ./out/app ./cmd/main.go

# Run stage
FROM alpine:3.9

WORKDIR /code

COPY --from=builder /code /code

CMD ["/code/out/app"]