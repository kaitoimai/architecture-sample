FROM golang:1.24-bookworm AS builder
WORKDIR /app

COPY go.* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
  -ldflags="-w -s" \
  -o ./server \
  ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot AS runner
WORKDIR /app

COPY --from=builder /app/server ./

ENTRYPOINT ["./server"]
CMD []
