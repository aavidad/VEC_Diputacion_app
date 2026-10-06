#!/usr/bin/env bash
# Prueba de AD208 en PostgreSQL 18.4 desechable, sin red ni datos.
# Uso: bash deploy/postgresql/autorizacion_atestada_v3/pruebas/000208_origen_no_acreditado_sqlstate.sh [definicion.sql]
#   Sin argumento usa un núcleo sustituto con la misma firma, metadatos y
#   sentencia de AD172. Con argumento (salida de pg_get_functiondef del núcleo
#   de una base con AD172) repite además el cambio sobre ese texto real.
set -euo pipefail
umask 077
base=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
migracion="$base/migraciones/000208_origen_no_acreditado_sqlstate.up.sql"
definicion=${1:-}
contenedor=vec-ad208-prueba-$$
trap 'docker rm -f "$contenedor" >/dev/null 2>&1 || true' EXIT
espera() { [[ "$1" == "$2" ]] || { echo "FALLO: $3 (obtenido «$1», esperado «$2»)" >&2; exit 1; }; echo "OK $3"; }
sql() { docker exec -i "$contenedor" psql -XAtq -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
migrar() { docker exec -i "$contenedor" psql -XAtq -v ON_ERROR_STOP=1 -U postgres -d postgres < "$migracion"; }
huella() { sql -c "SELECT encode(sha256(convert_to(pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'UTF8')),'hex')"; }
meta() { sql -c "SELECT md5((to_jsonb(p)-'prosrc')::text) FROM pg_proc p WHERE oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure"; }

base_pg() {
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
  docker run -d --rm --name "$contenedor" --network none --memory 2g \
    -e POSTGRES_HOST_AUTH_METHOD=trust postgres:18.4 >/dev/null
  for _ in $(seq 60); do
    docker exec "$contenedor" pg_isready -U postgres -h /var/run/postgresql >/dev/null 2>&1 && break; sleep 1
  done
  sleep 1
  sql <<'SQL' >/dev/null
CREATE ROLE vec_autorizacion_atestada_v3_propietario NOLOGIN;
CREATE ROLE vec_consumidor_sintetico LOGIN;
CREATE SCHEMA vec_autorizacion_atestada_v3 AUTHORIZATION vec_autorizacion_atestada_v3_propietario;
SET ROLE vec_autorizacion_atestada_v3_propietario;
CREATE TABLE vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1 (login_nombre name);
CREATE FUNCTION vec_autorizacion_atestada_v3.resolver_origen_consumo_v1(text,text,text) RETURNS text
 LANGUAGE sql STABLE AS $$ SELECT NULL::text $$;
RESET ROLE;
SQL
}

nucleo_sustituto() {
  sql <<'SQL' >/dev/null
SET ROLE vec_autorizacion_atestada_v3_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(p_perfil_mutacion text, p_capacidad_canonica bytea, p_decision_canonica bytea, p_motivo_canonico bytea, p_contexto_actor_canonico bytea, p_persona_version numeric, p_perfil_version numeric, p_payload_vec_ad_3 bytea, p_sobre_cose_sign1 bytea, p_evidencia_verificacion bytea, p_raiz_publica_spki bytea)
 RETURNS TABLE(decision_ref text, efecto_ref text, huella_efecto_sha256 text, consumo_huella_sha256 text, auditoria_ref text, consumida_en timestamp with time zone, consumo_nuevo boolean)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET lock_timeout TO '2s'
AS $function$
DECLARE
    c jsonb := pg_catalog.convert_from(p_capacidad_canonica, 'UTF8')::jsonb;
    d jsonb := pg_catalog.convert_from(p_decision_canonica, 'UTF8')::jsonb;
    v_proceso_origen text;
    v_canal_origen text;
BEGIN
    v_canal_origen := d #>> '{vinculo_autenticacion_actor,superficie}';
    v_proceso_origen := vec_autorizacion_atestada_v3.resolver_origen_consumo_v1(
        c ->> 'audiencia_consumo', c ->> 'operacion', v_canal_origen);
    IF v_proceso_origen IS NULL THEN
        RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='origen de consumo no acreditado';
    END IF;
END
$function$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
RESET ROLE;
SQL
}

# Invoca el núcleo como lo haría un consumidor y devuelve SQLSTATE|mensaje|detalle.
invocar() {
  sql <<'SQL'
DO $$
DECLARE e text; m text; t text;
BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna('x',
    convert_to('{"audiencia_consumo":"vec_prueba.audiencia.v1","operacion":"prueba.operacion"}','UTF8'),
    convert_to('{"vinculo_autenticacion_actor":{"superficie":"interna_corporativa"}}','UTF8'),
    ''::bytea,''::bytea,1,1,''::bytea,''::bytea,''::bytea,''::bytea);
EXCEPTION WHEN OTHERS THEN
  GET STACKED DIAGNOSTICS e = RETURNED_SQLSTATE, m = MESSAGE_TEXT, t = PG_EXCEPTION_DETAIL;
  RAISE NOTICE 'R|%|%|%', e, m, t;
END $$;
SQL
}
resultado() { invocar 2>&1 | sed -n 's/.*NOTICE:  R|//p'; }

echo '== A: núcleo sustituto'
base_pg
nucleo_sustituto
espera "$(resultado)" '42501|origen de consumo no acreditado|' 'antes: 42501 sin detalle (el 403 mudo)'
antes_meta=$(meta)
migrar >/dev/null
espera "$(resultado)" 'VA172|origen de consumo no acreditado|audiencia=vec_prueba.audiencia.v1 operacion=prueba.operacion canal=interna_corporativa' 'después: VA172 con la terna y sin LOGIN'
espera "$(meta)" "$antes_meta" 'firma, propietario, configuración y ACL intactos'
espera "$(sql -c "SELECT has_function_privilege('vec_consumidor_sintetico','vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')")" f 'sigue sin EXECUTE para otros'
despues=$(huella)
if migrar >/dev/null 2>&1; then echo 'FALLO: aceptó reejecutar' >&2; exit 1; fi
espera "$(huella)" "$despues" 'la reejecución se rechaza sin cambios'

echo '== B: preimagen con la sentencia repetida'
base_pg
nucleo_sustituto
sql -c "SET ROLE vec_autorizacion_atestada_v3_propietario; DO \$\$ BEGIN EXECUTE replace(pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure), 'END IF;', 'END IF;' || E'\n' || '    IF false THEN' || E'\n' || '        RAISE EXCEPTION USING ERRCODE=''42501'', MESSAGE=''origen de consumo no acreditado'';' || E'\n' || '    END IF;'); END \$\$" >/dev/null
antes=$(huella)
if migrar >/dev/null 2>&1; then echo 'FALLO: aceptó marca repetida' >&2; exit 1; fi
espera "$(huella)" "$antes" 'marca repetida rechazada sin cambios'

echo '== C: núcleo con permisos de más'
base_pg
nucleo_sustituto
sql -c "GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_consumidor_sintetico" >/dev/null
antes=$(huella)
if migrar >/dev/null 2>&1; then echo 'FALLO: aceptó ACL ampliada' >&2; exit 1; fi
espera "$(huella)" "$antes" 'ACL ampliada rechazada sin cambios'

if [[ -n $definicion ]]; then
  echo '== D: texto real del núcleo'
  base_pg
  { echo 'SET ROLE vec_autorizacion_atestada_v3_propietario;'; cat "$definicion"; echo ';'
    echo "REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;"; } | sql >/dev/null
  antes=$(huella); antes_meta=$(meta)
  migrar >/dev/null
  espera "$(meta)" "$antes_meta" 'metadatos del núcleo real intactos'
  espera "$(sql -c "SELECT encode(sha256(convert_to(replace(pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure), E'        -- AD208: falta de configuración técnica, no denegación. Sin LOGIN.\n        RAISE EXCEPTION USING ERRCODE=''VA172'', MESSAGE=''origen de consumo no acreditado'',\n            DETAIL=pg_catalog.format(''audiencia=%s operacion=%s canal=%s'',\n                c ->> ''audiencia_consumo'', c ->> ''operacion'', v_canal_origen);', '        RAISE EXCEPTION USING ERRCODE=''42501'', MESSAGE=''origen de consumo no acreditado'';'),'UTF8')),'hex')")" "$antes" 'el núcleo real solo cambia esa sentencia'
  echo "huella real antes=$antes después=$(huella)"
fi
echo 'PRUEBA-OK'
