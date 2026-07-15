# syntax=docker/dockerfile:1

FROM golang:1.26.5-bookworm@sha256:1ecb7edf62a0408027bd5729dfd6b1b8766e578e8df93995b225dfd0944eb651 AS build

WORKDIR /src

RUN useradd --create-home --uid 10001 app \
  && chown -R app:app /src /go

USER app

COPY --chown=app:app go.mod go.sum ./
RUN go mod download

# Copia deliberadamente cerrada: evita que documentación de trabajo, copias
# históricas o fuentes con datos personales entren en ninguna capa de imagen.
COPY --chown=app:app cmd ./cmd
COPY --chown=app:app config ./config
COPY --chown=app:app internal ./internal
COPY --chown=app:app locales ./locales
COPY --chown=app:app web ./web
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -trimpath \
  -ldflags="-s -w" \
  -o /src/bin/vec-server \
  ./cmd/vec-server

FROM debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818 AS runtime

RUN useradd --system --uid 10001 --home-dir /nonexistent --shell /usr/sbin/nologin app

COPY --from=build /src/bin/vec-server /usr/local/bin/vec-server
COPY --from=build /src/config /app/config
COPY --from=build /src/locales /app/locales
COPY --from=build /src/web /app/web

USER app
WORKDIR /app
ENV VEC_HTTP_ADDR=:8080
ENV VEC_AUTH_MODE=disabled
ENV VEC_BOLSA_STORAGE_MODE=local_durable
ENV VEC_BOLSA_DATA_DIR=/data/bolsa
# La imagen no acepta identidades hasta configurar el futuro adaptador de
# aserciones protegidas. Las cabeceras heredadas quedan solo para pruebas
# locales expresamente habilitadas y nunca son el valor de la imagen.
ENV VEC_TRUSTED_PROXY_CIDRS=127.0.0.1/32,::1/128
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/vec-server"]
