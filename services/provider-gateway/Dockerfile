FROM golang:1.22-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/provider-gateway ./services/provider-gateway

FROM gcr.io/distroless/base-debian12
COPY --from=builder /out/provider-gateway /provider-gateway
EXPOSE 8080
ENTRYPOINT ["/provider-gateway"]

