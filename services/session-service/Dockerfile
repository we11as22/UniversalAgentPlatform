FROM golang:1.22-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/session-service ./services/session-service

FROM gcr.io/distroless/base-debian12
COPY --from=builder /out/session-service /session-service
EXPOSE 8080
ENTRYPOINT ["/session-service"]

