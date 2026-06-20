# syntax=docker/dockerfile:1

FROM golang:1.22-bookworm AS build

WORKDIR /src

RUN useradd --create-home --uid 10001 app \
  && chown -R app:app /src /go

USER app

COPY --chown=app:app go.mod ./
RUN go mod download

COPY --chown=app:app . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -trimpath \
  -ldflags="-s -w" \
  -o /src/bin/vec-server \
  ./cmd/vec-server

FROM debian:bookworm-slim AS runtime

RUN useradd --system --uid 10001 --home-dir /nonexistent --shell /usr/sbin/nologin app

COPY --from=build /src/bin/vec-server /usr/local/bin/vec-server

USER app
ENV VEC_HTTP_ADDR=:8080
ENV VEC_AUTH_MODE=trusted_headers
ENV VEC_BOLSA_STORAGE_MODE=local_durable
ENV VEC_BOLSA_DATA_DIR=/data/bolsa
ENV VEC_TRUSTED_PROXY_CIDRS=127.0.0.1/32,::1/128,172.16.0.0/12
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/vec-server"]
