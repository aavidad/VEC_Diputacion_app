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
  ./cmd/vec-server \
  && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -trimpath \
  -ldflags="-s -w" \
  -o /src/bin/vec-presentacion \
  ./cmd/vec-presentacion \
  && cp -a /src/web /src/web-presentacion \
  && cp -a /src/web /src/web-produccion \
  && find /src/web-produccion -depth -iname '*presentacion*' -exec rm -rf '{}' + \
  && find /src/web-produccion -type f -iname '*demo*' -delete \
  && test ! -e /src/web-produccion/static/presentacion

# Artefacto deliberadamente distinto. Incluye exclusivamente datos sinteticos,
# no declara volumen durable y su composicion no crea conectores externos.
FROM debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818 AS runtime-presentacion

RUN useradd --system --uid 10001 --home-dir /nonexistent --shell /usr/sbin/nologin app \
  && install -d --owner=app --group=app /app

COPY --from=build /src/bin/vec-presentacion /usr/local/bin/vec-presentacion
COPY --from=build /src/config /app/config
COPY --from=build /src/locales /app/locales
COPY --from=build /src/web-presentacion /app/web
COPY --chown=app:app data/demo/convocatorias_publicas.demo.json /app/data/demo/convocatorias_publicas.demo.json
COPY --chown=app:app data/catalogos/categorias-profesionales/v1.demo.json /app/data/catalogos/categorias-profesionales/v1.demo.json

USER app
WORKDIR /app
ENV VEC_HTTP_ADDR=127.0.0.1:8080
ENV VEC_HTTP_ALLOWED_CIDRS=127.0.0.1/32,::1/128
ENV VEC_EXECUTION_PROFILE=presentacion_rrhh
ENV VEC_RRHH_PRESENTATION_ENABLED=true
ENV VEC_RRHH_PRESENTATION_GUARD_ONE=ACEPTO_MODO_PRESENTACION_RRHH_NO_AUTORITATIVO
ENV VEC_RRHH_PRESENTATION_GUARD_TWO=CONFIRMO_DATOS_SINTETICOS_SIN_VALIDEZ_ADMINISTRATIVA
ENV VEC_BOLSA_STORAGE_MODE=memory
ENV VEC_PERSONAL_CATALOG_PATH=memory
ENV VEC_BOLSA_PUBLIC_SOURCE_PATH=/app/data/demo/convocatorias_publicas.demo.json
ENV VEC_BOLSA_CATEGORIES_SOURCE_PATH=/app/data/catalogos/categorias-profesionales/v1.demo.json
ENV VEC_BOLSA_CATEGORIES_CATALOG_ID=categorias-profesionales
ENV VEC_BOLSA_CATEGORIES_CATALOG_VERSION=1
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/vec-presentacion"]

FROM debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818 AS runtime

RUN useradd --system --uid 10001 --home-dir /nonexistent --shell /usr/sbin/nologin app \
  && install -d --owner=app --group=app /app /data/bolsa

COPY --from=build /src/bin/vec-server /usr/local/bin/vec-server
COPY --from=build /src/config /app/config
COPY --from=build /src/locales /app/locales
COPY --from=build /src/web-produccion /app/web

USER app
WORKDIR /app
ENV VEC_HTTP_ADDR=:8080
ENV VEC_EXECUTION_PROFILE=produccion
ENV VEC_BOLSA_STORAGE_MODE=local_durable
ENV VEC_BOLSA_DATA_DIR=/data/bolsa
ENV VEC_BOLSA_PUBLIC_SOURCE_PATH=/run/vec/convocatorias_publicas.json
ENV VEC_BOLSA_CATEGORIES_SOURCE_PATH=/run/vec/categorias_profesionales.json
ENV VEC_BOLSA_CATEGORIES_CATALOG_ID=categorias-profesionales
ENV VEC_BOLSA_CATEGORIES_CATALOG_VERSION=1
# La imagen no acepta identidades hasta configurar el futuro adaptador de
# aserciones protegidas. Las cabeceras heredadas quedan solo para pruebas
# locales expresamente habilitadas y nunca son el valor de la imagen.
ENV VEC_TRUSTED_PROXY_CIDRS=127.0.0.1/32,::1/128
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/vec-server"]
