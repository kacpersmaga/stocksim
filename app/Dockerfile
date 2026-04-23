FROM golang:1.22-alpine AS builder

WORKDIR /app

# Download dependencies first for better layer caching.
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -trimpath \
    -o /stocksim ./main.go

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /stocksim /stocksim

EXPOSE 8080

ENTRYPOINT ["/stocksim"]
