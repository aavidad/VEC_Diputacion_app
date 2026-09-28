#!/usr/bin/env bash
# CT139 en PostgreSQL 18 efímero. AD3 es un doble sintético: no acredita V3 real.
set -Eeuo pipefail
raiz=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
imagen=postgres:18.4
contenedor="vec-ct139-${PPID}-${RANDOM}"
datos="/dev/shm/$contenedor"
limpiar() {
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
  docker run --rm -v /dev/shm:/limpiar --entrypoint rm "$imagen" -rf "/limpiar/$contenedor" >/dev/null 2>&1 || true
}
trap limpiar EXIT
mkdir -p "$datos"
docker run -d --rm --network none --name "$contenedor" \
  -e POSTGRES_HOST_AUTH_METHOD=trust -v "$datos:/var/lib/postgresql" \
  -v "$raiz:/repo:ro" "$imagen" >/dev/null
for _ in $(seq 1 90); do
  if docker exec "$contenedor" pg_isready -q -U postgres -d postgres; then break; fi
  sleep 1
done
docker exec "$contenedor" pg_isready -q -U postgres -d postgres
psql_super() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
psql_login() { docker exec -i "$contenedor" psql -X -q -At -v ON_ERROR_STOP=1 -U vec_ct138_login -d postgres "$@"; }
ct=/repo/deploy/postgresql/contratacion_temporal
pruebas=$ct/pruebas_sql
psql_super -f "$pruebas/ct138_fixture_minimo_pg18.sql" >/dev/null
psql_super -f "$ct/migraciones/000056_respuesta_recibida_rrhh.up.sql" >/dev/null
psql_super -f "$pruebas/ct138_preimagen_sintetica_pg18.sql" >/dev/null
psql_super -f "$pruebas/ct138_sondas_minimas_pg18.sql" >/dev/null
psql_super -f "$ct/migraciones/000138_respuesta_recibida_semantica.up.sql" >/dev/null
psql_login -c "BEGIN ISOLATION LEVEL SERIALIZABLE; SELECT vec_contratacion_temporal.ct138_registrar_sintetico(vec_contratacion_temporal.ct138_material_sintetico('llamamiento:nuevo','comunicacion:nueva','',repeat('b',64)),2); COMMIT;" >/dev/null
psql_super -f "$pruebas/ct139_compartido_fixture_pg18.sql" >/dev/null
psql_super -f "$ct/migraciones/000139_consulta_recibo_respuesta.up.sql" >/dev/null
respuesta=$(psql_login -c "BEGIN ISOLATION LEVEL SERIALIZABLE; SELECT vec_contratacion_temporal.ct139_consultar_sintetico('org:ct138','exp:ct138','comunicacion:nueva','actor:otro')::text; COMMIT;" | rg '^\{')
python3 - "$respuesta" <<'PY'
import json,sys
r=json.loads(sys.argv[1]); assert r['Encontrado'] is True, r
assert r['Recibo']['ExpedienteRef']=='exp:ct138',r
assert r['Recibo']['ComunicacionRef']=='comunicacion:nueva',r
PY
ausencia=$(psql_login -c "BEGIN ISOLATION LEVEL SERIALIZABLE; SELECT vec_contratacion_temporal.ct139_consultar_sintetico('org:ct138','exp:ajeno','comunicacion:nueva','actor:otro')::text; COMMIT;" | rg '^\{')
python3 - "$ausencia" <<'PY'
import json,sys
r=json.loads(sys.argv[1]); assert r=={'Encontrado':False,'Recibo':{}},r
PY
ausencia=$(psql_login -c "BEGIN ISOLATION LEVEL SERIALIZABLE; SELECT vec_contratacion_temporal.ct139_consultar_sintetico('org:ct138','exp:ct138','comunicacion:inexistente','actor:otro')::text; COMMIT;" | rg '^\{')
python3 - "$ausencia" <<'PY'
import json,sys
r=json.loads(sys.argv[1]); assert r=={'Encontrado':False,'Recibo':{}},r
PY
[[ $(psql_super -At -c "SELECT count(*) FROM vec_autorizacion_atestada_v3.ct139_lecturas_sinteticas") == 3 ]]
psql_super -c "SET ROLE vec_contratacion_temporal_propietario; UPDATE vec_contratacion_temporal.comunicacion_llamamiento_local SET recibo_json=jsonb_set(recibo_json,'{Solicitud,ExpedienteRef}','\"exp:inconsistente\"') WHERE comunicacion_ref='comunicacion:nueva';" >/dev/null
inconsistente=$(psql_login -c "BEGIN ISOLATION LEVEL SERIALIZABLE; SELECT vec_contratacion_temporal.ct139_consultar_sintetico('org:ct138','exp:ct138','comunicacion:nueva','actor:otro')::text; COMMIT;" | rg '^\{')
python3 - "$inconsistente" <<'PY'
import json,sys
r=json.loads(sys.argv[1]); assert r=={'Encontrado':False,'Recibo':{}},r
PY
[[ $(psql_super -At -c "SELECT count(*) FROM vec_autorizacion_atestada_v3.ct139_lecturas_sinteticas") == 4 ]]
printf 'CT139 PG18 sintético: lector distinto, expediente ajeno, ausencia, recibo inconsistente y cuatro auditorías OK\n'
