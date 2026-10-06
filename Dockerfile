# syntax=docker/dockerfile:1

FROM golang:1.26.6-bookworm@sha256:116d58cbd88c1297624acc6e967a060012422bacf9930927e23fb719189c6f36 AS build

WORKDIR /src

RUN groupadd --gid 10001 app \
  && useradd --create-home --uid 10001 --gid 10001 app \
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
# Revisión para el registro técnico: --build-arg VEC_REVISION=$(git rev-parse --short=12 HEAD)
ARG VEC_REVISION=
# Solo vec-interno enlaza PKCS#11 para HMAC con clave no exportable; las otras
# superficies conservan sus binarios sin CGO.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -trimpath \
  -ldflags="-s -w -X vec-diputacion-granada/internal/shared/telemetria.Revision=${VEC_REVISION}" \
  -o /src/bin/vec-server \
  ./cmd/vec-server \
  && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -trimpath \
  -ldflags="-s -w -X vec-diputacion-granada/internal/shared/telemetria.Revision=${VEC_REVISION}" \
  -o /src/bin/vec-publico \
  ./cmd/vec-publico \
  && CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
  -trimpath \
  -ldflags="-s -w -X vec-diputacion-granada/internal/shared/telemetria.Revision=${VEC_REVISION}" \
  -o /src/bin/vec-interno \
  ./cmd/vec-interno \
  && install -d /src/web-produccion \
  && while IFS= read -r ruta; do \
       test -n "$ruta"; \
       test "${ruta#/}" = "$ruta"; \
       test "${ruta#*..}" = "$ruta"; \
       test -f "/src/web/$ruta"; \
       install -D -m 0644 "/src/web/$ruta" "/src/web-produccion/$ruta"; \
     done < /src/web/produccion.manifest \
  && test ! -e /src/web-produccion/static/index.html \
  && test ! -e /src/web-produccion/static/app.js \
  && test ! -e /src/web-produccion/static/modulos \
  && test ! -e /src/web-produccion/static/catalogo-categorias.js \
  && test ! -e /src/web-produccion/static/catalogo-categorias.css

# Cada superficie recibe únicamente los recursos enumerados por su manifiesto.
# El manifiesto se renombra dentro del artefacto porque el servidor lo usa como
# lista positiva HTTP, pero el inventario fuente permanece separado y revisable.
RUN for superficie in publico interno; do \
      destino="/src/web-${superficie}"; \
      manifiesto="/src/web/${superficie}.manifest"; \
      install -d "${destino}"; \
      install -m 0644 "${manifiesto}" "${destino}/produccion.manifest"; \
      while IFS= read -r ruta; do \
        test -n "${ruta}"; \
        test "${ruta#/}" = "${ruta}"; \
        test "${ruta#*..}" = "${ruta}"; \
        test -f "/src/web/${ruta}"; \
        install -D -m 0644 "/src/web/${ruta}" "${destino}/${ruta}"; \
      done <"${manifiesto}"; \
    done \
  && test ! -e /src/web-publico/static/portal-empleado \
  && test ! -e /src/web-publico/cartografia \
  && test -f /src/web-interno/cartografia/granada-base-20260719-z8-z12.zip \
  && test -f /src/web-interno/cartografia/granada-base-20260719-z8-z12.json \
  && test ! -e /src/web-publico/static/area-personal \
  && test ! -e /src/web-interno/static/bolsa \
  && test ! -e /src/web-interno/static/verificar \
  && test ! -e /src/web-interno/static/area-personal \
  && ! find /src/web-publico /src/web-interno -type f \
       \( -iname '*.test.js' -o -iname '*.test.mjs' -o -iname '*demo*' -o -iname '*presentacion*' \) \
       -print -quit | grep -q . \
  && install -d /src/locales-interno \
  && while IFS= read -r ruta; do \
       test -n "${ruta}"; \
       test "${ruta#/}" = "${ruta}"; \
       test "${ruta#*..}" = "${ruta}"; \
       test -f "/src/locales/${ruta}"; \
       install -D -m 0644 "/src/locales/${ruta}" "/src/locales-interno/${ruta}"; \
     done </src/web/interno.locales.manifest

# Superficie pública productiva: un único binario, recursos anónimos y
# certificados raíz. No contiene Portal del Empleado, fuentes DEMO, secretos,
# clientes internos, KMS ni configuración administrativa.
FROM debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818 AS runtime-publico

RUN groupadd --system --gid 10001 app \
  && useradd --system --uid 10001 --gid 10001 --home-dir /nonexistent --shell /usr/sbin/nologin app \
  && install -d --owner=app --group=app /app

COPY --from=build /src/bin/vec-publico /usr/local/bin/vec-publico
COPY --from=build /src/web-publico /app/web
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

USER app
WORKDIR /app
ENV VEC_HTTP_ADDR=:8080
ENV VEC_EXECUTION_PROFILE=produccion
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/vec-publico"]

# Superficie corporativa productiva: binario y recursos distintos de los del
# portal anonimo. No incorpora certificados, claves, DSN ni selectores de
# desarrollo; Sistemas debe inyectar todos los proveedores y secretos en
# tiempo de ejecucion. Mientras falte uno, vec-interno falla antes de escuchar.
FROM debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818 AS runtime-interno

RUN groupadd --system --gid 10001 app \
  && useradd --system --uid 10001 --gid 10001 --home-dir /nonexistent --shell /usr/sbin/nologin app \
  && install -d --owner=app --group=app /app

COPY --from=build /src/bin/vec-interno /usr/local/bin/vec-interno
COPY --from=build /src/web-interno /app/web
COPY --from=build /src/locales-interno /app/locales
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

USER app
WORKDIR /app
EXPOSE 8443

ENTRYPOINT ["/usr/local/bin/vec-interno"]

FROM debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818 AS runtime

RUN groupadd --system --gid 10001 app \
  && useradd --system --uid 10001 --gid 10001 --home-dir /nonexistent --shell /usr/sbin/nologin app \
  && install -d --owner=app --group=app /app /data/bolsa

COPY --from=build /src/bin/vec-server /usr/local/bin/vec-server
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
