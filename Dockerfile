FROM golang:alpine AS builder
WORKDIR /go/src/github.com/filetrust/icap-adaptation-service
COPY . .
RUN cd cmd \
    && env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o  adaptation-service .

FROM scratch
COPY --from=builder /go/src/github.com/filetrust/icap-adaptation-service/cmd/adaptation-service /bin/adaptation-service

ENTRYPOINT ["/bin/adaptation-service"]
