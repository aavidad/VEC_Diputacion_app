#!/usr/bin/env bash
# Clon sintético H1; dependencias H3/H4 por objetos, B73 y B76. Nunca DOWN.
set -euo pipefail
raiz=$(cd "$(dirname "$0")/../../../.." && pwd)
estado=${VEC_B76_CLON_ESTADO:-/home/alberto/.local/state/vec-clon/estado-cidonia-20260929-hito1.tgz}
[[ -r "$estado" ]] || { echo 'Falta snapshot sintético H1'; exit 2; }
[[ "$(basename "$estado")" == estado-cidonia-20260929-hito1.tgz ]] || { echo 'Snapshot H1 distinto'; exit 2; }
scratch=$(mktemp -d /dev/shm/vec-b76-XXXXXX)
contenedor="vec-clon-b76-${scratch##*-}"
trap 'docker rm -f "$contenedor" >/dev/null 2>&1 || true; docker run --rm --network none --pull never -v "$scratch":/scratch alpine:3.22 rm -rf /scratch/data; rmdir "$scratch"' EXIT
mkdir "$scratch/data"
docker run --rm --network none --pull never -v "$scratch/data":/d -v "$estado":/estado.tgz:ro alpine:3.22 tar -C /d -xzf /estado.tgz
docker run -d --rm --network none --pull never --name "$contenedor" \
 --user postgres --read-only --security-opt no-new-privileges --cap-drop ALL \
 --cpus 1 --memory 512m --pids-limit 128 --ulimit nofile=1024:1024 \
 --tmpfs /tmp:rw,size=32m --tmpfs /var/run/postgresql:rw,size=8m,mode=1777 \
 -e HOME=/tmp -v "$scratch/data":/var/lib/postgresql postgres:18.4 >/dev/null
for ((i=0;i<60;i++)); do
 if docker exec "$contenedor" pg_isready -U postgres >/dev/null 2>&1; then break; fi
 sleep 1
done
docker exec "$contenedor" pg_isready -U postgres >/dev/null
q() { docker exec -i "$contenedor" psql -X -q -t -A -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
sql() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres < "$raiz/$1"; }
# Solo se habilitan dependencias cuando falta su objeto (sin reaplicación).
while IFS='|' read -r objeto fichero; do
 [[ "$(q "SELECT to_regclass('$objeto') IS NOT NULL")" == t ]] || sql "$fichero"
done <<'DEPS'
vec_contratacion_temporal.numeracion_anual_asignada|deploy/postgresql/contratacion_temporal/migraciones/000142_numeracion_anual_expedientes_anteriores.up.sql
vec_bolsa_llamamientos.plazas_oferta|deploy/postgresql/bolsa_llamamientos/migraciones/000058_plazas_oferta.up.sql
DEPS
[[ "$(q "SELECT to_regproc('vec_contratacion_temporal.gobi_o404b_material_catalogo_v2') IS NOT NULL")" == t ]] || sql deploy/postgresql/contratacion_temporal/migraciones/000141_requisitos_via_expediente.up.sql
# H4 CT143 no crea tabla; la función acredita su presencia.
[[ "$(q "SELECT to_regproc('vec_contratacion_temporal.numero_visible_aviso_confirmado_v1') IS NOT NULL")" == t ]] || sql deploy/postgresql/contratacion_temporal/migraciones/000143_numero_visible_aviso_llamamiento.up.sql
[[ "$(q "SELECT to_regproc('vec_bolsa_llamamientos.registrar_situacion_participacion_interna_b73') IS NOT NULL")" == t ]] || sql deploy/postgresql/bolsa_llamamientos/migraciones/000073_reincorporacion_b8_justificada.up.sql
sql deploy/postgresql/bolsa_llamamientos/pruebas_sql/b76_preparar_historia.sql
# Ensayo de la migración exacta en ROLLBACK; no se ejecuta ningún DOWN.
sed 's/^COMMIT;$/ROLLBACK;/' "$raiz/deploy/postgresql/bolsa_llamamientos/migraciones/000076_revision_regularizacion_rrhh.up.sql" \
 | docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres
[[ "$(q "SELECT to_regproc('vec_bolsa_llamamientos.registrar_situacion_participacion_interna_b76') IS NULL")" == t ]] \
 || { echo 'ROLLBACK no revirtió B76'; exit 1; }
sql deploy/postgresql/bolsa_llamamientos/migraciones/000076_revision_regularizacion_rrhh.up.sql
sql deploy/postgresql/bolsa_llamamientos/pruebas_sql/b76_revision_regularizacion.sql
echo 'ENSAYO-OK B76 (doble AD3 transaccional; sin principal ni custodia real)'
