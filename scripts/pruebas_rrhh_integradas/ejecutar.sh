#!/usr/bin/env bash
# Ensayo RRHH sintético: PostgreSQL efímero y navegador contra un servidor local ya autorizado.
set -Eeuo pipefail

raiz=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
modo=${1:-}

uso() {
  cat <<'EOF'
Uso:
  ejecutar.sh pg18
  ejecutar.sh navegador ORIGEN CERTIFICADO CLAVE

pg18 ejecuta sólo los ensayos existentes de cese, disponibilidad +5/+9,
reincorporación y política de ofertas de 48 h. Cada uno crea su PostgreSQL 18
en /dev/shm y lo elimina. navegador no inicia ni reinicia VEC: exige un origen
HTTPS loopback y material mTLS sintético ya preparado por dirección.
EOF
}

case "$modo" in
  pg18)
    "$raiz/deploy/postgresql/bolsa_llamamientos/probar_cese_pg18.sh"
    "$raiz/deploy/postgresql/bolsa_llamamientos/probar_mi_bolsa_disponibilidad_pg18.sh"
    "$raiz/deploy/postgresql/bolsa_llamamientos/probar_reincorporacion_titular_pg18.sh"
    "$raiz/deploy/postgresql/bolsa_llamamientos/probar_politica_ofertas_48h_pg18.sh"
    ;;
  navegador)
    (( $# == 4 )) || { uso >&2; exit 2; }
    exec python3 "$raiz/scripts/pruebas_rrhh_integradas/recorrido_mi_bolsa.py" \
      --origen "$2" --certificado "$3" --clave "$4"
    ;;
  *) uso >&2; exit 2 ;;
esac
