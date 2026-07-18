#!/bin/sh
set -eu

cd "$(dirname "$0")/.."

export VEC_HTTP_ADDR="${VEC_PRESENTACION_ADDR:-127.0.0.1:8081}"
export VEC_HTTP_ALLOWED_CIDRS="127.0.0.1/32,::1/128"
export VEC_EXECUTION_PROFILE="presentacion_rrhh"
export VEC_RRHH_PRESENTATION_ENABLED="true"
export VEC_RRHH_PRESENTATION_GUARD_ONE="ACEPTO_MODO_PRESENTACION_RRHH_NO_AUTORITATIVO"
export VEC_RRHH_PRESENTATION_GUARD_TWO="CONFIRMO_DATOS_SINTETICOS_SIN_VALIDEZ_ADMINISTRATIVA"
export VEC_AUTH_MODE="disabled"
export VEC_BOLSA_STORAGE_MODE="memory"
export VEC_PERSONAL_CATALOG_PATH="memory"
export VEC_BOLSA_PUBLIC_SOURCE_PATH="data/demo/convocatorias_publicas.demo.json"
export VEC_BOLSA_CATEGORIES_SOURCE_PATH="data/catalogos/categorias-profesionales/v1.demo.json"

exec go run ./cmd/vec-presentacion
