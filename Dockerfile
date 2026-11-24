FROM golang:1.27-bookworm AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /bin/api ./cmd/api \
    && CGO_ENABLED=0 go build -o /bin/worker ./cmd/worker

FROM debian:bookworm-slim AS runtime

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app

FROM runtime AS api

COPY --from=build /bin/api /app/api

EXPOSE 8080

ENTRYPOINT ["/app/api"]

FROM runtime AS worker

COPY --from=build /bin/worker /app/worker

ENTRYPOINT ["/app/worker"]
