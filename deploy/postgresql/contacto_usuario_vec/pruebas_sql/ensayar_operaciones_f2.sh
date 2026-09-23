#!/usr/bin/env bash
set -euo pipefail

# Extiende el runner estructural F2 con ContextoActor, PDP, gobierno/COSE y
# LOGIN nominal en el mismo contenedor PG18 sin red. La prueba Go conserva
# NO-GO hasta que todos sus efectos y recibos se acrediten de verdad.
VEC_F2_POSITIVO=1 exec "$(dirname "${BASH_SOURCE[0]}")/ensayar_contacto_f2.sh"
