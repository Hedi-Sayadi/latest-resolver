# Stage1: build the binary using the Go builder image.
FROM golang:latest as builder

WORKDIR /app

COPY . .

RUN go build -o tekton-resolver

# Stage2:  Copy the binary from the builder image into an alpine linux image.
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/tekton-resolver .


ENTRYPOINT ["./tekton-resolver"]
