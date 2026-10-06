#!/usr/bin/env bash
# Ensayo del guion sobre PostgreSQL 18.4 desechable, sin red ni datos reales.
# Uso: bash deploy/principal/origen_consumos_ad172/probar_pg18.sh
set -euo pipefail
umask 077
base_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)
repo_dir=$(cd "$base_dir/../../.." && pwd -P)
ad172="$repo_dir/deploy/postgresql/autorizacion_atestada_v3/migraciones/000172_origen_consumos_confirmados.up.sql"
contenedor=vec-origen-ad172-prueba-$$
TMPDIR=$(mktemp -d)
export TMPDIR
trap 'docker rm -f "$contenedor" >/dev/null 2>&1 || true; rm -rf "$TMPDIR"' EXIT
lista=$(grep -v '^#' "$base_dir/ternas.tsv")
# El guion solo acepta el núcleo cotejado. El de la prueba es un sustituto, así
# que se ensaya una copia del paquete cuya única huella es la del sustituto.
paquete="$TMPDIR/paquete"
mkdir "$paquete"
cp "$base_dir"/{ejecutar.sh,operacion.sql,inventario.sql,ternas.tsv} "$paquete/"
cuenta() { awk -F'\t' -v b="$1" '$1 ~ "^(" b ")$"' <<< "$lista" | wc -l | tr -d ' '; }
por_defecto=$(cuenta 'usuarios|contratacion|bolsa|documentos|incorporacion')
cronos=$(cuenta cronos)

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
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.probar_origen(text,text,text) TO PUBLIC;
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
  docker exec -i "$contenedor" psql -XAtq -v ON_ERROR_STOP=1 -U postgres -d postgres -v "ternas=$lista" \
    < "$base_dir/fixture_pg18.sql" >/dev/null
  ad172_objetos | docker exec -i "$contenedor" psql -XAtq -v ON_ERROR_STOP=1 -U postgres -d postgres >/dev/null
}
huella_nucleo() { sql -c "SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure"; }
sql() { docker exec -i "$contenedor" psql -XAtq -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
guion() { VEC_ORIGEN_MOTOR=docker VEC_ORIGEN_PG_CONTENEDOR=$contenedor bash "$paquete/ejecutar.sh" "$@"; }
aplicar() { env "$@" VEC_ORIGEN_AD172_APLICAR=SI-REVISADO VEC_ORIGEN_MOTOR=docker VEC_ORIGEN_PG_CONTENEDOR="$contenedor" bash "$paquete/ejecutar.sh" --aplicar; }
origen() { sql -U "$1" -c "SELECT coalesce(vec_autorizacion_atestada_v3.probar_origen('$2','$3','$4'),'NULO')"; }
espera() { [[ "$1" == "$2" ]] || { echo "FALLO: $3 (obtenido «$1», esperado «$2»)" >&2; exit 1; }; echo "OK $3"; }
rechaza() { # rechaza <motivo esperado> <descripción> [VAR=valor...]
  local motivo=$1 desc=$2; shift 2
  local err; if err=$(env "$@" VEC_ORIGEN_MOTOR=docker VEC_ORIGEN_PG_CONTENEDOR="$contenedor" bash "$paquete/ejecutar.sh" --aplicar 2>&1 >/dev/null); then
    echo "FALLO: $desc (aceptado)" >&2; exit 1; fi
  grep -q -- "$motivo" <<< "$err" || { echo "FALLO: $desc (motivo distinto: $err)" >&2; exit 1; }
  echo "OK $desc"
}
filas() { sql -c 'SELECT count(*) FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1'; }
ap=VEC_ORIGEN_AD172_APLICAR=SI-REVISADO

echo "== A: bloques por defecto ($por_defecto ternas): ensayo, aplicación, resolutor y repetición"
nuevo_pg
huella_nucleo > "$paquete/nucleos_cotejados.txt"
espera "$(origen vec_pref508a_i_ue vec_usuarios.preferencias.consultar.interna_corporativa.v1 vec.preferencias.consultar interna_corporativa)" NULO 'sin fila el resolutor deniega (causa del 403)'
salida=$(guion --ensayo)
grep -q "ternas_nuevas=$por_defecto" <<< "$salida"; grep -q 'verificado: ROLLBACK' <<< "$salida"
espera "$(filas)" 1 'el ensayo no deja filas'
if guion --aplicar >/dev/null 2>&1; then echo 'FALLO: aplicó sin confirmación' >&2; exit 1; fi
espera "$(filas)" 1 'aplicar sin confirmación no cambia nada'
salida=$(aplicar)
grep -q "ternas_nuevas=$por_defecto" <<< "$salida"; grep -q 'verificado: COMMIT' <<< "$salida"
espera "$(filas)" "$((por_defecto + 1))" 'aplicar añade las ternas y conserva la previa'
espera "$(origen vec_pref508a_i_ue vec_usuarios.preferencias.consultar.interna_corporativa.v1 vec.preferencias.consultar interna_corporativa)" vec-usuarios 'preferencias acreditadas'
espera "$(origen vec_ct_o207_runtime vec_contratacion_temporal.confirmar_alta_atestada.v1 contratacion_temporal.peticion_centro.consultar interna_corporativa)" vec-contratacion-temporal 'peticiones del centro acreditadas'
espera "$(origen vec_bolsa_llamamientos_desarrollo vec_bolsa_llamamientos.contacto_participacion.consultar.v1 bolsa.contacto_participacion.consultar interna_corporativa)" vec-bolsa 'contacto de Bolsa acreditado'
espera "$(origen vec_documentos_rrhh_ejecutor_desarrollo vec_documentos.operacion.v1 documentos.expediente.listar interna_corporativa)" vec-documentos 'documentos acreditados'
espera "$(origen vec_inc_v2_alta_personal_20260910 vec_personal.alta_ejercicio.v1 personal.alta_ejercicio.registrar interna_corporativa)" vec-incorporacion 'alta en Personal acreditada'
espera "$(origen vec_cronos_emp_ejecutor_desarrollo vec_cronos_v1.saldo_propio.consultar.v1 cronos.saldo.propio.consultar interna_corporativa)" NULO 'Cronos no se instala por defecto'
espera "$(origen vec_ct_o207_runtime vec_contratacion_temporal.confirmar_alta_atestada.v1 contratacion_temporal.peticion_centro.consultar externa_personal)" NULO 'canal cruzado sigue denegado'
espera "$(origen vec_bolsa_llamamientos_desarrollo vec_contratacion_temporal.confirmar_alta_atestada.v1 contratacion_temporal.peticion_centro.consultar interna_corporativa)" NULO 'LOGIN cruzado sigue denegado'
espera "$(origen vec_inc_v2_registro_ct_20260910 vec_contratacion_temporal.confirmar_alta_atestada.v1 contratacion_temporal.peticion_centro.consultar interna_corporativa)" NULO 'otro LOGIN del mismo grupo sigue denegado'
salida=$(aplicar)
grep -q 'ternas_nuevas=0' <<< "$salida"
espera "$(filas)" "$((por_defecto + 1))" 'repetir es idempotente'
salida=$(aplicar VEC_ORIGEN_BLOQUES=cronos)
grep -q "ternas_nuevas=$cronos" <<< "$salida"
espera "$(origen vec_cronos_emp_ejecutor_desarrollo vec_cronos_v1.saldo_propio.consultar.v1 cronos.saldo.propio.consultar interna_corporativa)" vec-cronos 'Cronos se instala solo si se pide'

echo '== B: terna previa con otro proceso'
nuevo_pg
sql -c "SET ROLE vec_autorizacion_atestada_v3_propietario; INSERT INTO vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1 VALUES ('vec_ct_o207_runtime','vec_contratacion_temporal.confirmar_alta_atestada.v1','contratacion_temporal.solicitud.crear','otro-proceso','interna_corporativa')" >/dev/null
rechaza 'otro proceso o canal' 'conflicto rechazado' "$ap"
espera "$(filas)" 2 'sin cambios tras el conflicto'

echo '== C: LOGIN con una segunda membresía'
nuevo_pg
sql -c 'CREATE ROLE vec_extra NOLOGIN; GRANT vec_extra TO vec_bolsa_llamamientos_desarrollo' >/dev/null
rechaza 'LOGIN o grupo ejecutor incompatible' 'membresía extra rechazada' "$ap"

echo '== D: LOGIN en un grupo distinto del que exige su perfil'
nuevo_pg
sql -c 'REVOKE vec_documentos_ejecutor FROM vec_documentos_rrhh_ejecutor_desarrollo; GRANT vec_contratacion_temporal_ejecutor TO vec_documentos_rrhh_ejecutor_desarrollo' >/dev/null
rechaza 'LOGIN o grupo ejecutor incompatible' 'grupo equivocado rechazado' "$ap"

echo '== E: tabla con permisos de más'
nuevo_pg
sql -c 'GRANT SELECT ON vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1 TO vec_ct_o207_runtime' >/dev/null
rechaza 'AD172 ausente o con permisos' 'ACL ampliada rechazada' "$ap"

echo '== F: núcleo sin AD172'
nuevo_pg
sql -c "DROP FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)" >/dev/null
rechaza 'núcleo sin AD172' 'núcleo ausente rechazado' "$ap"

echo '== G: bloque desconocido y LOGIN sin permiso de conexión'
rechaza 'bloque desconocido' 'bloque desconocido rechazado' "$ap" VEC_ORIGEN_BLOQUES=usuarios,dietas
nuevo_pg
sql -c 'ALTER ROLE vec_cronos_emp_ejecutor_desarrollo NOLOGIN' >/dev/null
rechaza 'LOGIN o grupo ejecutor incompatible' 'LOGIN sin conexión rechazado' "$ap" VEC_ORIGEN_BLOQUES=cronos
espera "$(filas)" 1 'sin cambios tras el rechazo'

echo '== H: núcleo distinto del cotejado'
nuevo_pg
cp "$base_dir/nucleos_cotejados.txt" "$paquete/nucleos_cotejados.txt"
rechaza 'núcleo distinto del cotejado' 'núcleo no cotejado rechazado' "$ap"
espera "$(filas)" 1 'sin cambios con núcleo no cotejado'
echo 'PRUEBA-OK'
