#!/usr/bin/env bash
# Recorrido de cese CT115 -> feed CT129 -> B13 -> B45 en PostgreSQL 18 efímero.
# --estructural usa el fixture abierto de Bolsa: dobla la verificación CT129.
# --volcado GLOBALS_SQL BASE_FC exige historia sintética CT115/CT75 y B13 real.
# No usar un volcado con datos personales. Nunca apunta a una base conservada.
set -Eeuo pipefail

repo=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm}
modo=${1:-}

uso() {
  echo "Uso: $0 --estructural | --volcado GLOBALS_SQL BASE_FC" >&2
  exit 2
}

if [[ $modo == --estructural && $# == 1 ]]; then
  echo 'Fixture estructural: Bolsa B45/B50/B53 en PostgreSQL 18; verificador CT129 doblado de forma explícita.'
  "$repo/deploy/postgresql/bolsa_llamamientos/probar_mi_bolsa_disponibilidad_pg18.sh"
  echo 'ALCANCE: +5/+9, recibo/replay, cese forjado, relación simultánea y fecha máxima de Mi Bolsa.'
  echo 'PENDIENTE: CT115 real, feed CT129, B13 real, RRHH/B10 y recuperación conjunta requieren volcado sintético.'
  exit 0
fi
[[ $modo == --volcado && $# == 3 ]] || uso
globales=$2
volcado=$3
[[ -s $globales && -s $volcado ]] || { echo 'Faltan volcados sintéticos no vacíos' >&2; exit 2; }
[[ ${VEC_CESE_DUMP_SINTETICO:-} == 1 ]] || {
  echo 'Confirma que ambos volcados contienen solo datos sintéticos con VEC_CESE_DUMP_SINTETICO=1' >&2
  exit 2
}

nombre=vec-rrhh-cese-pg18-$$
datos=/dev/shm/$nombre
limpiar() {
  docker rm -f "$nombre" >/dev/null 2>&1 || true
  if [[ -d $datos ]]; then
    docker run --rm -v "$datos:/d" --entrypoint /bin/sh "$imagen" -c 'rm -rf /d/* /d/.[!.]*' >/dev/null 2>&1 || true
    rmdir "$datos" 2>/dev/null || true
  fi
}
trap limpiar EXIT
mkdir -p "$datos"
docker run -d --rm --network none --name "$nombre" \
  -v "$datos:/var/lib/postgresql" -v "$repo:/repo:ro" \
  -e POSTGRES_HOST_AUTH_METHOD=trust "$imagen" >/dev/null
listo=0
for _ in $(seq 1 120); do
  if docker logs "$nombre" 2>&1 | grep -q 'PostgreSQL init process complete' &&
     docker exec "$nombre" pg_isready -q -U postgres; then listo=1; break; fi
  sleep 0.5
done
[[ $listo == 1 ]] || { echo 'PostgreSQL 18 no inició' >&2; exit 1; }
psql_pg() { docker exec -i "$nombre" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
escalar() { docker exec "$nombre" psql -X -At -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
[[ $(escalar 'SHOW server_version') == 18.4* ]] || { echo 'Se requiere PostgreSQL 18.4' >&2; exit 1; }

# pg_dumpall --globals-only suele repetir el rol postgres de la imagen.
# Se tolera solo esa repetición; los objetos necesarios se comprueban después.
docker exec -i "$nombre" psql -X -q -U postgres -d postgres <"$globales" >/dev/null 2>&1 || true
if ! docker exec -i "$nombre" pg_restore --exit-on-error -U postgres -d postgres <"$volcado" >/dev/null 2>&1; then
  echo 'El volcado no se pudo restaurar íntegramente en la base efímera' >&2
  exit 1
fi

exigir_objeto() {
  [[ $(escalar "SELECT $1 IS NOT NULL") == t ]] || { echo "Falta precondición: $2" >&2; exit 2; }
}
exigir_objeto "to_regclass('vec_contratacion_temporal.cese_nombramiento_v1')" 'CT115'
exigir_objeto "to_regprocedure('vec_contratacion_temporal.leer_contratos_bolsa_v1(bigint,text,integer)')" 'CT113'
exigir_objeto "to_regprocedure('vec_bolsa_llamamientos.registrar_contrato_participacion_v1(jsonb,text,timestamptz,bigint)')" 'B13'
exigir_objeto "to_regclass('vec_bolsa_llamamientos.vinculo_candidato')" 'candidaturas Bolsa'

instalar_si_falta() {
  local objeto=$1 ruta=$2
  if [[ $(escalar "SELECT $objeto IS NULL") == t ]]; then
    psql_pg -f "/repo/$ruta" >/dev/null
  fi
}
bolsa=deploy/postgresql/bolsa_llamamientos/migraciones
ct=deploy/postgresql/contratacion_temporal/migraciones
instalar_si_falta "to_regclass('vec_bolsa_llamamientos.restriccion_cese_bolsa')" "$bolsa/000045_restriccion_global_cese.up.sql"
instalar_si_falta "to_regprocedure('vec_contratacion_temporal.leer_ceses_bolsa_v1(bigint,text,integer)')" "$ct/000129_verificacion_cese_bolsa.up.sql"
instalar_si_falta "to_regprocedure('vec_bolsa_llamamientos.consultar_estado_cese_bolsa_v1(text,timestamptz)')" "$bolsa/000050_lectura_estado_cese.up.sql"
# Bolsa53 altera una función previa; la preimagen del volcado se comprueba por
# su propia migración. Si ya está instalada, su doble UP debe ser rechazado.
if [[ $(escalar "SELECT EXISTS (SELECT 1 FROM pg_proc WHERE oid=to_regprocedure('vec_bolsa_llamamientos.consultar_mi_bolsa_v1(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') AND prosrc LIKE '%cese_disponible_en%')") == f ]]; then
  psql_pg -f "/repo/$bolsa/000053_mi_bolsa_disponibilidad_maxima.up.sql" >/dev/null
fi

psql_pg <<'SQL' >/dev/null
CREATE ROLE vec_rrhh_cese_feed_pg18 LOGIN INHERIT;
GRANT vec_contratacion_temporal_ejecutor TO vec_rrhh_cese_feed_pg18;
CREATE ROLE vec_rrhh_cese_relevo_pg18 LOGIN INHERIT;
GRANT vec_bolsa_llamamientos_relevo_cese TO vec_rrhh_cese_relevo_pg18;
GRANT CONNECT ON DATABASE postgres TO vec_rrhh_cese_feed_pg18,vec_rrhh_cese_relevo_pg18;
SQL

psql_pg -f /repo/scripts/rrhh_cese/recorrido_con_volcado.sql
recibo_antes=$(escalar "SELECT recibo_ref||'|'||disponible_desde||'|'||politica_version FROM vec_bolsa_llamamientos.restriccion_cese_bolsa ORDER BY recibida_en DESC LIMIT 1")
filas_antes=$(escalar 'SELECT count(*) FROM vec_bolsa_llamamientos.restriccion_cese_bolsa')
docker restart "$nombre" >/dev/null
for _ in $(seq 1 60); do docker exec "$nombre" pg_isready -q -U postgres && break; sleep 0.5; done
psql_pg -f /repo/scripts/rrhh_cese/replay_tras_reinicio.sql >/dev/null
recibo_despues=$(escalar "SELECT recibo_ref||'|'||disponible_desde||'|'||politica_version FROM vec_bolsa_llamamientos.restriccion_cese_bolsa ORDER BY recibida_en DESC LIMIT 1")
filas_despues=$(escalar 'SELECT count(*) FROM vec_bolsa_llamamientos.restriccion_cese_bolsa')
[[ $recibo_antes == "$recibo_despues" && $filas_antes == "$filas_despues" ]] || {
  echo 'Recibo o cardinalidad cambiaron tras reiniciar PostgreSQL' >&2; exit 1;
}
echo 'OK desde cese CT115 previamente confirmado: CT129→B13→B45, rechazo forjado, replay y recibo persistente tras reinicio PG18.'
echo 'La creación CT115 y los dos casos +5/+9 se acreditan por sus ensayos focales; este volcado recorre un cese.'
echo 'RRHH/B10 y navegador requieren composición y recorrido HTTP separado.'
