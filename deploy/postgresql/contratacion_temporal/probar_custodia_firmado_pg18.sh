#!/usr/bin/env bash
# Recorrido de extremo a extremo de 5.06 (custodia del PDF firmado) sobre un
# clon desechable del estado de la principal (PostgreSQL 18.4): instala
# AD3-113, Documentos 000009 y CT145, sustituye las fachadas AD3 por dobles
# sin COSE (pruebas_sql/custodia_firmado_e2e_dobles.sql) y ejecuta
# TestCustodiaFirmadoRecorridoPG18: firma por HTTP de CT, verificación,
# custodia en Documentos, registro con enlace, consulta, descarga por HTTP de
# Documentos y reintento sin duplicar.
# Uso: probar_custodia_firmado_pg18.sh [estado-del-clon.tgz]
# Por defecto usa VEC_CLON_ESTADO. Los datos viven en /dev/shm y se borran al
# terminar; el contenedor usa --rm y publica solo en 127.0.0.1.
set -Eeuo pipefail

repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
estado=${1:-${VEC_CLON_ESTADO:?falta el estado del clon (VEC_CLON_ESTADO o primer argumento)}}
[[ -r $estado ]] || { echo "No se puede leer $estado" >&2; exit 2; }
nombre="vec-pg-custodia506-$$"
datos=$(mktemp -d "/dev/shm/$nombre-XXXX")
limpiar() {
  docker rm -f "$nombre" >/dev/null 2>&1 || true
  docker run --rm -v /dev/shm:/limpiar alpine rm -rf "/limpiar/$(basename "$datos")" >/dev/null 2>&1 || rm -rf "$datos" 2>/dev/null || true
}
trap limpiar EXIT
docker run --rm -v "$datos:/d" -v "$(dirname "$estado"):/o:ro" alpine tar -C /d -xzf "/o/$(basename "$estado")"
docker run -d --rm --name "$nombre" -p 127.0.0.1::5432 -v "$datos:/var/lib/postgresql" postgres:18.4 >/dev/null
for _ in $(seq 1 120); do docker exec "$nombre" pg_isready -q -U postgres && break; sleep 0.5; done
sleep 1
[[ $(docker exec "$nombre" psql -X -At -U postgres -c 'SHOW server_version') == 18.4* ]] || { echo 'PostgreSQL 18.4 no disponible' >&2; exit 2; }
run() { docker exec -i "$nombre" psql -X -q -o /dev/null -v ON_ERROR_STOP=1 -U postgres -d postgres < "$1"; }
base=$repo/deploy/postgresql
for f in autorizacion_atestada_v3/migraciones/000113_custodia_documento_firmado.up.sql \
         documentos/migraciones/000009_custodia_documento_firmado.up.sql \
         contratacion_temporal/migraciones/000145_enlace_firma_documento_custodiado.up.sql \
         contratacion_temporal/pruebas_sql/custodia_firmado_e2e_dobles.sql; do
  run "$base/$f"; echo "OK $f"
done
puerto=$(docker port "$nombre" 5432/tcp | head -1 | sed 's/.*://')
b="127.0.0.1:$puerto/postgres?sslmode=disable"
(cd "$repo" && VEC_CUSTODIA_E2E_ADMIN_DSN="postgres://postgres@$b" \
  VEC_CUSTODIA_E2E_CT_DSN="postgres://vec_e2e_custodia_ct@$b" \
  VEC_CUSTODIA_E2E_DOCUMENTOS_DSN="postgres://vec_e2e_custodia_documentos@$b" \
  go test -count=1 -v -run 'TestCustodiaFirmadoRecorridoPG18' ./internal/app/bootstrap/)
# Una sola custodia, un solo enlace y un solo documento; el enlace lleva la
# huella del documento custodiado.
[[ $(docker exec "$nombre" psql -X -At -U postgres -c "SELECT (SELECT count(*) FROM vec_documentos.documento_firmado)=1
  AND (SELECT count(*) FROM vec_contratacion_temporal.firma_documento_custodia_v1)=1
  AND EXISTS (SELECT 1 FROM vec_contratacion_temporal.firma_documento_custodia_v1 c JOIN vec_documentos.documento d
               ON d.id=c.documento_ref AND d.version=c.documento_version AND d.huella_sha256=c.documento_huella_sha256)") == t ]] \
  || { echo 'FALLO: filas de custodia y enlace' >&2; exit 1; }
printf 'PG18.4 (clon de la principal): firma, custodia, enlace, consulta, descarga y reintento sin duplicados. Fachadas AD3 dobles: NO acredita COSE real.\n'
