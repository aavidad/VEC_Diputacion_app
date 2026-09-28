#!/usr/bin/env bash
# Inventario de solo lectura: detiene la tanda si B49 o Pública3 ya están instaladas.
set -Eeuo pipefail

fallar() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }
[[ $# == 1 ]] || fallar 'uso: preflight_no_go.sh --clon|--destino'
case $1 in
  --clon)
    principal=${VEC_PIDEN_CLON_DB:-}
    publica=${VEC_PIDEN_BOLSA_PUBLICA_CLON_DB:-}
    [[ $principal == *_clon_piden_20260928 && $publica == *_clon_piden_20260928 ]] \
      || fallar 'nombres de clon no acreditados'
    ;;
  --destino)
    principal=${VEC_PIDEN_DESTINO_DB:-}
    publica=${VEC_PIDEN_BOLSA_PUBLICA_DESTINO_DB:-}
    [[ -n $principal && -n $publica ]] || fallar 'faltan nombres de bases destino'
    ;;
  *) fallar 'uso: preflight_no_go.sh --clon|--destino' ;;
esac
[[ -n ${PGSERVICE:-} && -n ${VEC_PIDEN_BOLSA_PUBLICA_PGSERVICE:-} ]] \
  || fallar 'faltan servicios privados PostgreSQL principal/pública'
command -v psql >/dev/null || fallar 'falta psql'

leer() {
  local servicio=$1 consulta=$2
  PGSERVICE="$servicio" psql -X --no-psqlrc --set=ON_ERROR_STOP=1 \
    --no-align --tuples-only --field-separator='|' --command "$consulta"
}

# La migración B49 crea la tabla, el LOGIN grupal y ambas funciones en COMMIT.
# Rechazamos también huellas parciales; nunca asumimos que falta por no figurar
# en el plan Git. La consulta requiere DBA para inventario completo de roles.
sql_principal="SELECT current_database(), current_setting('server_version_num'),
  (SELECT rolsuper FROM pg_roles WHERE rolname=current_user),
  (to_regclass('vec_bolsa_llamamientos.publicacion_cese_b10') IS NOT NULL
   OR to_regprocedure('vec_bolsa_llamamientos.siguiente_publicacion_cese_b10_v1()') IS NOT NULL
   OR to_regprocedure('vec_bolsa_llamamientos.confirmar_publicacion_cese_b10_v1(bigint,text,text,text)') IS NOT NULL
   OR EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_bolsa_llamamientos_publicador_cese'));"

# Pública3 renombra la función previa y crea tabla y confirmador de replay.
sql_publica="SELECT current_database(), current_setting('server_version_num'),
  (SELECT rolsuper FROM pg_roles WHERE rolname=current_user),
  (to_regclass('vec_bolsa_publica_publicacion.testigo_replay_v3') IS NOT NULL
   OR to_regprocedure('vec_bolsa_publica_publicacion.publicar_proyeccion_v3_original(jsonb,jsonb,text)') IS NOT NULL
   OR to_regprocedure('vec_bolsa_publica_publicacion.confirmar_replay_proyeccion_v3(jsonb,jsonb,text)') IS NOT NULL);"

comprobar() {
  local nombre=$1 servicio=$2 esperada=$3 consulta=$4 salida base version dba presente
  salida=$(leer "$servicio" "$consulta") || fallar "no se pudo inventariar $nombre"
  IFS='|' read -r base version dba presente <<<"$salida"
  [[ $base == "$esperada" ]] || fallar "$nombre apunta a base inesperada: $base"
  [[ $version =~ ^18[0-9]{4}$ && $dba == t ]] \
    || fallar "$nombre requiere PostgreSQL 18 y cuenta DBA"
  [[ $presente == f ]] || fallar "$nombre contiene huellas B49/Pública3: NO-GO"
  printf '%s: PG18, base y ausencia de huellas NO-GO comprobadas\n' "$nombre"
}

comprobar principal "$PGSERVICE" "$principal" "$sql_principal"
comprobar bolsa_publica "$VEC_PIDEN_BOLSA_PUBLICA_PGSERVICE" "$publica" "$sql_publica"
