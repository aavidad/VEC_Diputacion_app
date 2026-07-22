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
  -o /src/bin/vec-publico \
  ./cmd/vec-publico \
  && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -trimpath \
  -ldflags="-s -w" \
  -o /src/bin/vec-interno \
  ./cmd/vec-interno \
  && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -trimpath \
  -ldflags="-s -w" \
  -o /src/bin/vec-presentacion \
  ./cmd/vec-presentacion \
  && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -trimpath \
  -ldflags="-s -w" \
  -o /src/bin/vec-cartografia-presentacion \
  ./cmd/vec-cartografia-presentacion \
  && cp -a /src/web /src/web-presentacion \
  && find /src/web-presentacion -type f \( -iname '*.test.js' -o -iname '*.test.mjs' -o -iname '*test-helper*' \) -delete \
  && find /src/web-presentacion -type f \( -iname 'README*' -o -iname '*INTEGRACION*' \) -delete \
  && rm -f \
       /src/web-presentacion/static/index.html \
       /src/web-presentacion/static/app.js \
       /src/web-presentacion/static/catalogo-categorias.js \
       /src/web-presentacion/static/catalogo-categorias.css \
       /src/web-presentacion/produccion.manifest \
  && rm -rf /src/web-presentacion/static/modulos \
  && install -d /src/web-produccion \
  && while IFS= read -r ruta; do \
       test -n "$ruta"; \
       test "${ruta#/}" = "$ruta"; \
       test "${ruta#*..}" = "$ruta"; \
       test -f "/src/web/$ruta"; \
       install -D -m 0644 "/src/web/$ruta" "/src/web-produccion/$ruta"; \
     done < /src/web/produccion.manifest \
  && test ! -e /src/web-produccion/static/presentacion \
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
  && test ! -e /src/web-publico/static/area-personal \
  && test ! -e /src/web-publico/static/presentacion \
  && test ! -e /src/web-interno/static/bolsa \
  && test ! -e /src/web-interno/static/verificar \
  && test ! -e /src/web-interno/static/area-personal \
  && ! find /src/web-publico /src/web-interno -type f \
       \( -iname '*.test.js' -o -iname '*.test.mjs' -o -iname '*demo*' -o -iname '*presentacion*' \) \
       -print -quit | grep -q .

# Artefacto deliberadamente distinto. Incluye exclusivamente datos sinteticos,
# no declara volumen durable y su composicion no crea conectores externos.
FROM debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818 AS runtime-presentacion

RUN useradd --system --uid 10001 --home-dir /nonexistent --shell /usr/sbin/nologin app \
  && install -d --owner=app --group=app /app

COPY --from=build /src/bin/vec-presentacion /usr/local/bin/vec-presentacion
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

# Mediador cartografico de la presentacion. Es un proceso distinto para que el
# servidor que entrega la web siga sin clientes de red ni identidad implicita.
FROM debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818 AS runtime-cartografia-presentacion

RUN useradd --system --uid 10001 --home-dir /nonexistent --shell /usr/sbin/nologin app \
  && install -d --owner=app --group=app /app

COPY --from=build /src/bin/vec-cartografia-presentacion /usr/local/bin/vec-cartografia-presentacion
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

USER app
WORKDIR /app
ENV VEC_HTTP_ADDR=127.0.0.1:8080
ENV VEC_HTTP_ALLOWED_CIDRS=127.0.0.1/32,::1/128
ENV VEC_EXECUTION_PROFILE=presentacion_rrhh
ENV VEC_RRHH_PRESENTATION_ENABLED=true
ENV VEC_RRHH_PRESENTATION_GUARD_ONE=ACEPTO_MODO_PRESENTACION_RRHH_NO_AUTORITATIVO
ENV VEC_RRHH_PRESENTATION_GUARD_TWO=CONFIRMO_DATOS_SINTETICOS_SIN_VALIDEZ_ADMINISTRATIVA
# El modo de autenticacion parte cerrado en Config. Si el despliegue intenta
# inyectar cualquier otro modo, la raiz de composicion rechaza el arranque.
ENV VEC_BOLSA_STORAGE_MODE=memory
ENV VEC_PERSONAL_CATALOG_PATH=memory
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/vec-cartografia-presentacion"]

# Superficie pública productiva: un único binario, recursos anónimos y
# certificados raíz. No contiene Portal del Empleado, fuentes DEMO, secretos,
# clientes internos, KMS ni configuración administrativa.
FROM debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818 AS runtime-publico

RUN useradd --system --uid 10001 --home-dir /nonexistent --shell /usr/sbin/nologin app \
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

RUN useradd --system --uid 10001 --home-dir /nonexistent --shell /usr/sbin/nologin app \
  && install -d --owner=app --group=app /app

COPY --from=build /src/bin/vec-interno /usr/local/bin/vec-interno
COPY --from=build /src/web-interno /app/web
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

USER app
WORKDIR /app
EXPOSE 8443

ENTRYPOINT ["/usr/local/bin/vec-interno"]

FROM debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818 AS runtime

RUN useradd --system --uid 10001 --home-dir /nonexistent --shell /usr/sbin/nologin app \
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

# Herramienta reproducible de revisión visual. Chromium y sus dependencias
# proceden de la imagen oficial fijada; el cliente Python y todas sus
# dependencias se instalan con versiones y huellas cerradas. Esta etapa nunca
# forma parte de los artefactos de ejecución de VEC.
FROM mcr.microsoft.com/playwright/python:v1.60.0-noble@sha256:8ff591d613b01c884cc488339ed4318b4513eaf0c57a164a878ba49e70e3f384 AS herramientas-revision-web

COPY scripts/revision_web/requirements.lock /tmp/requirements-revision-web.lock
RUN python3 -m pip install \
      --disable-pip-version-check \
      --no-cache-dir \
      --only-binary=:all: \
      --require-hashes \
      --requirement /tmp/requirements-revision-web.lock \
  && rm -f /tmp/requirements-revision-web.lock \
  && python3 -c 'import importlib.metadata as m; assert m.version("playwright") == "1.60.0"'

ENV PLAYWRIGHT_BROWSERS_PATH=/ms-playwright
WORKDIR /workspace
