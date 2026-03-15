FROM golang:1.22-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/quota-service ./services/quota-service

FROM gcr.io/distroless/base-debian12
COPY --from=builder /out/quota-service /quota-service
EXPOSE 8080
ENTRYPOINT ["/quota-service"]

