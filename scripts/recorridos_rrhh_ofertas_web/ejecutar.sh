#!/usr/bin/env bash
# Ensayo focal reproducible de RRHH 3.06/3.07 en el corte indicado por Git.
# La prueba PostgreSQL existente crea un contenedor --rm, sin red, en /dev/shm.
set -Eeuo pipefail

raiz=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
cd "$raiz"

case "${1:-}" in
  pg18)
    ./deploy/postgresql/bolsa_llamamientos/probar_politica_ofertas_48h_pg18.sh
    ./deploy/postgresql/bolsa_llamamientos/probar_disposicion_oferta_pg18.sh
    ./deploy/postgresql/bolsa_llamamientos/probar_revision_bolsa_pg18.sh
    ;;
  focales)
    export GOCACHE="${GOCACHE:-/dev/shm/vec-ofertas-go-cache}"
    export GOMAXPROCS="${GOMAXPROCS:-2}"
    go test -count=1 \
      ./internal/modules/bolsa/domain \
      ./internal/modules/bolsa/application/reglasadjudicacion \
      ./internal/modules/bolsa/application \
      ./internal/modules/bolsa/application/mibolsa \
      ./internal/modules/bolsa/adapters/httpinterno \
      ./internal/modules/bolsa/adapters/httppersonal \
      ./internal/app/bootstrap \
      -run 'Test(PoliticaOfertas|PlazoOferta|PlazoHorasNaturales|SinPolitica|CalendariosAusente|ConsultaRRHHExigeMaterial|PublicarOferta|ResolverOferta|ConsultarOfertas|HandlerOfertas|AuditoriaAnotaLasRutasDeOfertas|FronterasOfertas|RecursoPortalPropioSeparaOferta|Disposicion|RespuestaIncluyeOfertas)'
    node --test \
      web/static/portal-empleado/portal-bolsas-ofertas.test.mjs \
      web/static/area-personal/mi-bolsa-ofertas.test.mjs
    ;;
  *)
    printf 'Uso: %s {pg18|focales}\n' "${0##*/}" >&2
    exit 2
    ;;
esac
