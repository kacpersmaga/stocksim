FROM golang:1.22-alpine AS builder

WORKDIR /app

# Download dependencies first for better layer caching.
# -mod=mod allows go to update go.sum if not present.
COPY go.mod ./
RUN go mod download -x

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -mod=mod \
    -ldflags="-s -w" \
    -trimpath \
    -o /stocksim ./main.go

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /stocksim /stocksim

EXPOSE 8080

ENTRYPOINT ["/stocksim"]
