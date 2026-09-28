#!/usr/bin/env bash
# Reproduce literalmente la consulta de preflight del hash Go sobre CT136 real
# con un LOGIN sintético. Uso: probar_preflight_ct136_pg18.sh SQL_REF GO_REF
set -Eeuo pipefail

repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
sql_ref=${1:?falta hash con CT136 y delta DBA}
go_ref=${2:?falta hash con preflight bootstrap}
roles=deploy/postgresql/contratacion_temporal/roles_registrador_auditoria_up.sql
migracion=deploy/postgresql/contratacion_temporal/migraciones/000136_auditoria_frontera_auditoria_ruta_exacta.up.sql
fuente=internal/app/bootstrap/auditoria_consulta.go
fuente_asignacion=internal/app/bootstrap/autoridad_postgresql_desarrollo.go
for ruta in "$roles" "$migracion"; do git -C "$repo" cat-file -e "$sql_ref:$ruta"; done
git -C "$repo" cat-file -e "$go_ref:$fuente"
git -C "$repo" cat-file -e "$go_ref:$fuente_asignacion"
consulta=$(git -C "$repo" show "$go_ref:$fuente" | python3 "$repo/scripts/rrhh_auditoria/extraer_preflight_ct136.py")
lectura_asignacion=$(git -C "$repo" show "$go_ref:$fuente_asignacion" | python3 "$repo/scripts/rrhh_auditoria/extraer_asignacion_for_update.py")

nombre="vec-ct136-preflight-$$"
datos="/dev/shm/$nombre"
limpiar() {
  docker rm -f "$nombre" >/dev/null 2>&1 || true
  docker run --rm --network none -v /dev/shm:/limpiar postgres:18.4 \
    rm -rf "/limpiar/$nombre" >/dev/null 2>&1 || true
  rm -rf -- "$datos" 2>/dev/null || true
}
trap limpiar EXIT
mkdir -p "$datos"
docker run -d --rm --network none --name "$nombre" -e POSTGRES_HOST_AUTH_METHOD=trust \
  -v "$datos:/var/lib/postgresql" postgres:18.4 >/dev/null
for _ in $(seq 1 120); do
  if docker exec "$nombre" pg_isready -q -U postgres -d postgres; then break; fi
  sleep 0.5
done
docker exec "$nombre" pg_isready -q -U postgres -d postgres
super() { docker exec -i "$nombre" psql -X -q -A -t -v ON_ERROR_STOP=1 -U postgres -d ct136_preflight; }
login() { docker exec -i "$nombre" psql -X -q -A -t -v ON_ERROR_STOP=1 -U vec_ct136_preflight_login -d ct136_preflight; }
docker exec "$nombre" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres \
  -c 'CREATE ROLE vec_contratacion_temporal_propietario NOLOGIN' \
  -c 'CREATE DATABASE ct136_preflight' >/dev/null
printf '%s\n' 'CREATE SCHEMA vec_contratacion_temporal AUTHORIZATION vec_contratacion_temporal_propietario;' | super >/dev/null
git -C "$repo" show "$sql_ref:$roles" | super >/dev/null
git -C "$repo" show "$sql_ref:$migracion" | super >/dev/null
printf '%s\n' \
  'CREATE ROLE vec_ct136_preflight_login LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;' \
  'GRANT vec_contratacion_temporal_registrador_auditoria TO vec_ct136_preflight_login WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;' | super >/dev/null

sonda() { printf '%s\n' "$consulta" | login; }
igual() {
  local actual esperado etiqueta
  actual=$(sonda); esperado=$1; etiqueta=$2
  [[ $actual == "$esperado" ]] || { printf 'FALLO %s: %s, esperado %s\n' "$etiqueta" "$actual" "$esperado" >&2; exit 1; }
  printf 'OK %s\n' "$etiqueta"
}
igual t 'preflight CT136 con LOGIN nominal mínimo'
printf '%s\n' \
  'REVOKE USAGE ON SCHEMA vec_contratacion_temporal FROM vec_contratacion_temporal_registrador_auditoria;' \
  'GRANT USAGE ON SCHEMA vec_contratacion_temporal TO vec_ct136_preflight_login;' | super >/dev/null
igual f 'preflight deniega USAGE directo al LOGIN sin USAGE del rol CT136'
printf '%s\n' \
  'REVOKE USAGE ON SCHEMA vec_contratacion_temporal FROM vec_ct136_preflight_login;' \
  'GRANT USAGE ON SCHEMA vec_contratacion_temporal TO vec_contratacion_temporal_registrador_auditoria;' | super >/dev/null
igual t 'preflight recupera al restaurar USAGE del rol CT136'
printf '%s\n' 'GRANT SELECT (actor_ref) ON vec_contratacion_temporal.auditoria_frontera_auditoria_ruta_exacta TO vec_contratacion_temporal_registrador_auditoria;' | super >/dev/null
igual f 'preflight deniega SELECT(actor_ref) heredado'
printf '%s\n' 'REVOKE SELECT (actor_ref) ON vec_contratacion_temporal.auditoria_frontera_auditoria_ruta_exacta FROM vec_contratacion_temporal_registrador_auditoria;' | super >/dev/null
igual t 'preflight recupera al revocar SELECT(actor_ref)'
printf '%s\n' 'GRANT CREATE ON SCHEMA vec_contratacion_temporal TO vec_contratacion_temporal_registrador_auditoria;' | super >/dev/null
igual f 'preflight deniega CREATE heredado en esquema CT'
printf '%s\n' 'REVOKE CREATE ON SCHEMA vec_contratacion_temporal FROM vec_contratacion_temporal_registrador_auditoria;' | super >/dev/null
igual t 'preflight recupera al revocar CREATE'

# El helper de publicación usa el SELECT FOR UPDATE extraído del código Go.
# Su nueva transacción serializable ReadWrite permite la lectura bloqueante;
# ReadOnly produce el SQLSTATE 25006, incluso con una fila sintética válida.
cat <<'SQL' | super >/dev/null
CREATE ROLE vec_autorizacion_propietario NOLOGIN;
GRANT CREATE ON DATABASE ct136_preflight TO vec_autorizacion_propietario;
SET ROLE vec_autorizacion_propietario;
CREATE SCHEMA vec_autorizacion;
CREATE TABLE vec_autorizacion.asignacion_perfil_actual (
  perfil_activo_ref text PRIMARY KEY, asignacion_ref text NOT NULL
);
CREATE TABLE vec_autorizacion.asignacion_perfil (
  asignacion_ref text PRIMARY KEY, asignacion_id text NOT NULL,
  version integer NOT NULL, perfil_activo_ref text NOT NULL,
  principal_id text NOT NULL, version_rol_ref text NOT NULL,
  huella_sha256 text NOT NULL
);
INSERT INTO vec_autorizacion.asignacion_perfil_actual VALUES
  ('perfil:auditoria:sintetico','asignacion:sintetica');
INSERT INTO vec_autorizacion.asignacion_perfil VALUES
  ('asignacion:sintetica','asignacion-id-sintetica',1,
   'perfil:auditoria:sintetico','actor:sintetico',
   'rol:auditoria:sintetico',repeat('a',64));
RESET ROLE;
SQL
salida=$(printf 'BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;\nSET LOCAL ROLE vec_autorizacion_propietario;\n%s\nCOMMIT;\n' "$lectura_asignacion" | super)
[[ $salida == *'asignacion:sintetica'* ]] || { echo 'FALLO FOR UPDATE con ReadWrite' >&2; exit 1; }
echo 'OK lectura FOR UPDATE bajo rol propietario con Serializable ReadWrite'
if salida=$(printf '\\set VERBOSITY verbose\nBEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;\nSET LOCAL ROLE vec_autorizacion_propietario;\n%s\n' "$lectura_asignacion" | super 2>&1); then
  echo 'FALLO FOR UPDATE admitido en ReadOnly' >&2; exit 1
fi
[[ $salida == *'25006'* ]] || { echo 'FALLO ReadOnly no devolvió SQLSTATE 25006' >&2; exit 1; }
echo 'OK lectura FOR UPDATE denegada en ReadOnly con SQLSTATE 25006'
printf 'OK PG18 CT136 SQL=%s Go=%s\n' "$(git -C "$repo" rev-parse --short "$sql_ref")" "$(git -C "$repo" rev-parse --short "$go_ref")"
