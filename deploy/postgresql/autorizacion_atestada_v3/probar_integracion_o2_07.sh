#!/usr/bin/env bash
set -Eeuo pipefail

raiz="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd -P)"

VEC_EJECUTAR_O207=1 \
    bash "${raiz}/deploy/postgresql/autorizacion_atestada_v3/probar_integracion_o2_06.sh"
