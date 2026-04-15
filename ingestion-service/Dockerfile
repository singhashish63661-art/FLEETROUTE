FROM golang:1.22-alpine AS builder
WORKDIR /build

# Copy workspace files
COPY go.work go.work.sum* ./
COPY pkg/ pkg/
COPY protocols/ protocols/
COPY api-service/ api-service/
COPY ingestion-service/ ingestion-service/
COPY stream-processor/ stream-processor/
COPY websocket-service/ websocket-service/

RUN cd ingestion-service && go build -o /ingestion-service ./cmd/...

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /ingestion-service /ingestion-service
EXPOSE 5001 5002 5003 5004 5027 5028
ENTRYPOINT ["/ingestion-service"]
