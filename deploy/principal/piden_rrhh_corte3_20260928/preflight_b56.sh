#!/usr/bin/env bash
# Solo lectura, inmediatamente antes de Bolsa56 y después de Bolsa48.
set -Eeuo pipefail
script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
fallar() { printf 'ERROR: preflight B56: %s\n' "$*" >&2; exit 1; }
[[ $# == 1 && ( $1 == --clon || $1 == --destino ) ]] || fallar 'uso: preflight_b56.sh --clon|--destino'
"$script_dir/preflight_no_go.sh" "$1" >/dev/null
if [[ $1 == --clon ]]; then esperada=${VEC_PIDEN_CLON_DB:-}; else esperada=${VEC_PIDEN_DESTINO_DB:-}; fi
[[ -n $esperada && -n ${PGSERVICE:-} ]] || fallar 'servicio o nombre de base ausente'
salida=$(psql -X --no-psqlrc --set=ON_ERROR_STOP=1 --no-align --tuples-only --field-separator='|' \
  --command "SELECT current_database(), current_setting('server_version_num'),
  (SELECT rolsuper FROM pg_roles WHERE rolname=current_user),
  (to_regclass('vec_bolsa_llamamientos.situacion_participacion') IS NOT NULL
   AND to_regclass('vec_bolsa_llamamientos.operacion_situacion_participacion') IS NOT NULL
   AND to_regclass('vec_bolsa_llamamientos.traza_valor_participacion') IS NOT NULL
   AND to_regclass('vec_bolsa_llamamientos.datos_contacto_participacion') IS NOT NULL
   AND to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_auditoria_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
   AND EXISTS (SELECT 1 FROM pg_proc p
       WHERE p.oid=to_regprocedure('vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
         AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole
         AND p.prosecdef IS TRUE
         AND p.proacl::text='{vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_llamamientos_propietario}'
         AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')='363c4dfa08690195a47ad8ea90478ddf8edd12b0dee8a64b48180571536c3d91'
         AND position('B56: motivo unido' in pg_get_functiondef(p.oid))=0));") \
  || fallar 'inventario SQL imposible'
IFS='|' read -r base version dba preimagen <<<"$salida"
[[ $base == "$esperada" && $version =~ ^18[0-9]{4}$ && $dba == t && $preimagen == t ]] \
  || fallar 'base, PG18, DBA o B48 exacta (definición/owner/ACL) incompatibles; conciliar antes de instalar'
printf 'B56_PREIMAGEN_OK: PG18, B16/B48 exacta y B56 ausente; base=%s\n' "$base"
