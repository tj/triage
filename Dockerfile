
# Build container
FROM golang:1.13 as build
WORKDIR /go/src/app
COPY . .
ENV CGO_ENABLED=0
RUN go get ./...
RUN go build -o /triage ./cmd/triage/main.go

# Runtime container
FROM alpine:3.10
RUN apk --no-cache add ca-certificates
COPY --from=build /triage /triage
CMD ["/triage"]