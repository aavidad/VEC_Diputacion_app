#!/usr/bin/env bash
set -euo pipefail
umask 077
# El entorno privado administrativo lo carga systemd. No contiene permisos
# web ni una clave base: sólo rutas y la configuración del preparador actual.
: "${VEC_RENOVACION_BINARIO:?}"
: "${VEC_RENOVACION_MATERIAL_EXTERNO:?}"
: "${VEC_RENOVACION_PUBLICADOR_DSN:?}"
: "${VEC_RENOVACION_CONSUMIDORES:?}"
exec "$VEC_RENOVACION_BINARIO" preparar-portal-externo \
  --material-externo "$VEC_RENOVACION_MATERIAL_EXTERNO" \
  --publicador-confianza-dsn "$VEC_RENOVACION_PUBLICADOR_DSN" \
  --consumidores "$VEC_RENOVACION_CONSUMIDORES"
