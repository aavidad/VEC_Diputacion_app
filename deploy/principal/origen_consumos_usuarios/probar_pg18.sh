#!/usr/bin/env bash
# Ensayo del guion sobre PostgreSQL 18.4 desechable, sin red ni datos reales.
# Uso: bash deploy/principal/origen_consumos_usuarios/probar_pg18.sh
set -euo pipefail
umask 077
base_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)
repo_dir=$(cd "$base_dir/../../.." && pwd -P)
ad172="$repo_dir/deploy/postgresql/autorizacion_atestada_v3/migraciones/000172_origen_consumos_confirmados.up.sql"
contenedor=vec-origen-usuarios-prueba-$$
trap 'docker rm -f "$contenedor" >/dev/null 2>&1 || true' EXIT

# Tabla, disparadores, política y resolutor de AD172, copiados literalmente.
ad172_objetos() {
  echo 'SET ROLE vec_autorizacion_atestada_v3_propietario;'
  awk '/^CREATE TABLE vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1/{p=1}
       p{print}
       /^REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.resolver_origen_consumo_v1/{exit}' "$ad172"
  cat <<'SQL'
-- Envoltorio de prueba con la misma forma que el núcleo: definidor
-- propietario y sin SET ROLE, para que el resolutor vea al LOGIN real.
CREATE FUNCTION vec_autorizacion_atestada_v3.probar_origen(a text, o text, c text) RETURNS text
LANGUAGE sql SECURITY DEFINER SET search_path = pg_catalog
AS $$ SELECT vec_autorizacion_atestada_v3.resolver_origen_consumo_v1(a, o, c) $$;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.probar_origen(text,text,text) TO vec_pref508a_i_ue, vec_pref508a_e_ue;
INSERT INTO vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
 VALUES ('vec_adm_lector_sintetico','vec.admin.usuarios.consultar.v1','administracion.usuarios.consultar',
         'vec-admin-usuarios','administracion_privilegiada');
RESET ROLE;
SQL
}

nuevo_pg() {
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
  docker run -d --rm --name "$contenedor" --network none --memory 2g \
    -e POSTGRES_HOST_AUTH_METHOD=trust postgres:18.4 >/dev/null
  for _ in $(seq 60); do
    docker exec "$contenedor" pg_isready -U postgres -h /var/run/postgresql >/dev/null 2>&1 && break; sleep 1
  done
  sleep 1
  docker exec -i "$contenedor" psql -XAtq -v ON_ERROR_STOP=1 -U postgres -d postgres < "$base_dir/fixture_pg18.sql" >/dev/null
  ad172_objetos | docker exec -i "$contenedor" psql -XAtq -v ON_ERROR_STOP=1 -U postgres -d postgres >/dev/null
}
sql() { docker exec -i "$contenedor" psql -XAtq -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
guion() { VEC_ORIGEN_MOTOR=docker VEC_ORIGEN_PG_CONTENEDOR=$contenedor bash "$base_dir/ejecutar.sh" "$@"; }
origen() { sql -U "$1" -c "SELECT coalesce(vec_autorizacion_atestada_v3.probar_origen('$2','$3','$4'),'NULO')"; }
espera() { [[ "$1" == "$2" ]] || { echo "FALLO: $3 (obtenido «$1», esperado «$2»)" >&2; exit 1; }; echo "OK $3"; }
rechaza() { # rechaza <motivo esperado> <descripción> [VAR=valor...]
  local motivo=$1 desc=$2; shift 2
  local err; if err=$(env "$@" VEC_ORIGEN_MOTOR=docker VEC_ORIGEN_PG_CONTENEDOR="$contenedor" bash "$base_dir/ejecutar.sh" --aplicar 2>&1 >/dev/null); then
    echo "FALLO: $desc (aceptado)" >&2; exit 1; fi
  grep -q -- "$motivo" <<< "$err" || { echo "FALLO: $desc (motivo distinto: $err)" >&2; exit 1; }
  echo "OK $desc"
}
filas() { sql -c 'SELECT count(*) FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1'; }

echo '== A: ensayo, aplicación, resolutor y repetición'
nuevo_pg
espera "$(origen vec_pref508a_i_ue vec_usuarios.preferencias.consultar.interna_corporativa.v1 vec.preferencias.consultar interna_corporativa)" NULO 'sin fila el resolutor deniega (causa del 403)'
salida=$(guion --ensayo)
grep -q 'ternas_nuevas=20' <<< "$salida"; grep -q 'verificado: ROLLBACK' <<< "$salida"
espera "$(filas)" 1 'el ensayo no deja filas'
if guion --aplicar >/dev/null 2>&1; then echo 'FALLO: aplicó sin confirmación' >&2; exit 1; fi
espera "$(filas)" 1 'aplicar sin confirmación no cambia nada'
salida=$(VEC_ORIGEN_USUARIOS_APLICAR=SI-REVISADO guion --aplicar)
grep -q 'ternas_nuevas=20' <<< "$salida"; grep -q 'verificado: COMMIT' <<< "$salida"
espera "$(filas)" 21 'aplicar añade 20 ternas y conserva la previa'
espera "$(origen vec_pref508a_i_ue vec_usuarios.preferencias.consultar.interna_corporativa.v1 vec.preferencias.consultar interna_corporativa)" vec-usuarios 'preferencias interna acreditada'
espera "$(origen vec_pref508a_i_ue vec_usuarios.imagen.actualizar.interna_corporativa.v1 vec.imagen.actualizar interna_corporativa)" vec-usuarios 'imagen interna acreditada'
espera "$(origen vec_pref508a_i_ue vec_usuarios.correos.retirar.interna_corporativa.v1 vec.correos.retirar interna_corporativa)" vec-usuarios 'correos interna acreditada'
espera "$(origen vec_pref508a_e_ue vec_usuarios.preferencias.actualizar.externa_personal.v1 vec.preferencias.actualizar externa_personal)" vec-usuarios 'preferencias externa acreditada'
espera "$(origen vec_pref508a_i_ue vec_usuarios.preferencias.consultar.interna_corporativa.v1 vec.preferencias.consultar externa_personal)" NULO 'canal cruzado sigue denegado'
espera "$(origen vec_pref508a_e_ue vec_usuarios.preferencias.consultar.interna_corporativa.v1 vec.preferencias.consultar interna_corporativa)" NULO 'LOGIN cruzado sigue denegado'
espera "$(origen vec_pref508a_i_ue vec_usuarios.preferencias.consultar.interna_corporativa.v1 vec.preferencias.actualizar interna_corporativa)" NULO 'operación cruzada sigue denegada'
salida=$(VEC_ORIGEN_USUARIOS_APLICAR=SI-REVISADO guion --aplicar)
grep -q 'ternas_nuevas=0' <<< "$salida"
espera "$(filas)" 21 'repetir es idempotente'
espera "$(sql -c "SELECT count(*) FROM pg_auth_members m JOIN pg_roles r ON r.oid=m.member WHERE r.rolname LIKE 'vec_pref508a_%'")" 2 'no cambia membresías'

echo '== B: terna previa con otro proceso'
nuevo_pg
sql -c "SET ROLE vec_autorizacion_atestada_v3_propietario; INSERT INTO vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1 VALUES ('vec_pref508a_i_ue','vec_usuarios.preferencias.consultar.interna_corporativa.v1','vec.preferencias.consultar','otro-proceso','interna_corporativa')" >/dev/null
rechaza 'otro proceso o canal' 'conflicto rechazado' VEC_ORIGEN_USUARIOS_APLICAR=SI-REVISADO
espera "$(filas)" 2 'sin cambios tras el conflicto'

echo '== C: LOGIN con una segunda membresía'
nuevo_pg
sql -c 'CREATE ROLE vec_extra NOLOGIN; GRANT vec_extra TO vec_pref508a_i_ue' >/dev/null
rechaza 'LOGIN o grupo ejecutor incompatible' 'membresía extra rechazada' VEC_ORIGEN_USUARIOS_APLICAR=SI-REVISADO

echo '== D: segundo miembro del grupo ejecutor'
nuevo_pg
sql -c 'CREATE ROLE vec_intruso LOGIN; GRANT vec_usuarios_ejecutor_interno TO vec_intruso' >/dev/null
rechaza 'LOGIN o grupo ejecutor incompatible' 'grupo compartido rechazado' VEC_ORIGEN_USUARIOS_APLICAR=SI-REVISADO

echo '== E: tabla con permisos de más'
nuevo_pg
sql -c 'GRANT SELECT ON vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1 TO vec_pref508a_i_ue' >/dev/null
rechaza 'AD172 ausente o con permisos' 'ACL ampliada rechazada' VEC_ORIGEN_USUARIOS_APLICAR=SI-REVISADO

echo '== F: núcleo sin AD172'
nuevo_pg
sql -c "DROP FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)" >/dev/null
rechaza 'núcleo sin AD172' 'núcleo ausente rechazado' VEC_ORIGEN_USUARIOS_APLICAR=SI-REVISADO
echo 'PRUEBA-OK'
