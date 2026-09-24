#!/usr/bin/env bash
# Ensayo B2 con migraciones y consumo V3 reales. Solo datos sintéticos.
# Requiere una base sintética preparada por VEC_B2_BASELINE_SQL con la cadena
# ContextoActor 001..008, Personal 010/016/017/018 y AD3 hasta 050a.
# VEC_B2_CASOS_SQL debe generar material V3 firmado por la infraestructura
# real de pruebas y comprobar alta, replay, denegaciones y lectura. Nunca se
# sustituye el consumidor ni se fabrican decisiones dentro de este runner.
# El material firmado se recibe desde un archivo privado externo a Git.
# Hasta que estos dos SQL sintéticos y AD3-54..58 estén disponibles, el ensayo
# termina en preflight (código 2) antes de crear un contenedor.
set -euo pipefail

base_dir=$(CDPATH='' cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repo_dir=$(CDPATH='' cd -- "$base_dir/../../../.." && pwd)
motor=${VEC_CONTENEDOR_MOTOR:-docker}
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-personal-b2-real-${BASHPID}"
base=vec_personal_b2_real_sintetica
ad3_dir=deploy/postgresql/autorizacion_atestada_v3/migraciones
personal_dir=deploy/postgresql/personal/migraciones
ref_53=${VEC_B2_AD3_53_REF:-trabajo/cronos-montaje-20260925}
ref_59=${VEC_B2_AD3_59_REF:-trabajo/dietas-montaje-20260925}

fallo() { printf 'BLOQUEO B2 real: %s\n' "$*" >&2; exit 2; }
error() { printf 'FALLO B2 real: %s\n' "$*" >&2; exit 1; }
limpiar() { "$motor" rm -f "$contenedor" >/dev/null 2>&1 || true; }
consulta() { "$motor" exec "$contenedor" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d "$base" -c "$1"; }
archivo() { "$motor" exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d "$base" -o /dev/null < "$1"; }
aplicar_ref() { git -C "$repo_dir" show "$1:$2" | "$motor" exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d "$base" -o /dev/null; }
esperar() {
  local _
  for _ in {1..120}; do
    if consulta 'SELECT 1' >/dev/null 2>&1; then return 0; fi
    sleep 0.25
  done
  error 'PostgreSQL 18 no quedó disponible'
}
resolver_ad3() {
  local numero=$1 ruta
  case "$numero" in
    53) ruta=$(git -C "$repo_dir" ls-tree -r --name-only "$ref_53" "$ad3_dir" 2>/dev/null | rg '/000053_[^/]+\.up\.sql$' || true) ;;
    59) ruta=$(git -C "$repo_dir" ls-tree -r --name-only "$ref_59" "$ad3_dir" 2>/dev/null | rg '/000059_[^/]+\.up\.sql$' || true) ;;
    *) ruta=$(rg --files "$repo_dir/$ad3_dir" | rg "/0000${numero}_[^/]+\.up\.sql$" || true)
       ruta=${ruta#"$repo_dir"/} ;;
  esac
  [[ -n "$ruta" && $(printf '%s\n' "$ruta" | wc -l) -eq 1 ]] || fallo "AD3-0000${numero}: falta una única migración UP real"
  if [[ "$numero" == 53 || "$numero" == 59 ]]; then
    git -C "$repo_dir" cat-file -e "$( [[ "$numero" == 53 ]] && printf %s "$ref_53" || printf %s "$ref_59"):$ruta" 2>/dev/null || fallo "AD3-0000${numero}: referencia Git ilegible"
  fi
  printf '%s\n' "$ruta"
}
archivo_privado() {
  local ruta=$1 ruta_real directorio
  [[ $ruta == /* && -f $ruta && ! -L $ruta && -O $ruta ]] ||
    fallo 'el material V3 debe ser un archivo regular privado y absoluto fuera del repositorio'
  ruta_real=$(realpath -e -- "$ruta") || fallo 'la ruta privada no se puede resolver'
  [[ $ruta_real != "$repo_dir"/* ]] ||
    fallo 'el material V3 no puede estar dentro del repositorio público'
  directorio=$(dirname -- "$ruta_real")
  [[ -d $directorio && ! -L $directorio && -O $directorio ]] ||
    fallo 'el directorio del material V3 debe ser privado y propio'
  [[ $(stat -c %a -- "$ruta") == 600 && $(stat -c %a -- "$directorio") == 700 ]] ||
    fallo 'el material V3 requiere archivo 0600 y directorio 0700'
}

[[ ${1:-} == '' || ${1:-} == '--preflight' ]] || fallo 'uso: registro_empleado_b2_cadena_real_pg18.sh [--preflight]'
declare -a rutas_ad3=()
for numero in 51 52 53 54 55 56 57 58 59 60; do
  rutas_ad3+=("$(resolver_ad3 "$numero")")
done
for ruta in \
  "$personal_dir/000019_escritura_registro_empleado_b2.up.sql" \
  "$personal_dir/000018_lectura_registro_empleado_b2.up.sql"; do
  [[ -f "$repo_dir/$ruta" ]] || fallo "falta $ruta"
done
[[ -n ${VEC_B2_BASELINE_SQL:-} && -f ${VEC_B2_BASELINE_SQL:-} ]] ||
  fallo 'VEC_B2_BASELINE_SQL debe señalar un SQL sintético que instale la preimagen real hasta CA-008 y AD3-050a'
[[ -n ${VEC_B2_CASOS_SQL:-} && -f ${VEC_B2_CASOS_SQL:-} ]] ||
  fallo 'VEC_B2_CASOS_SQL debe señalar casos con firma/atestación V3 real; no se admiten stubs'
if [[ ${VEC_B2_BASELINE_SQL} != "$base_dir"/* ]]; then
  archivo_privado "$VEC_B2_BASELINE_SQL"
fi
archivo_privado "$VEC_B2_CASOS_SQL"
[[ $imagen =~ @sha256:[0-9a-f]{64}$ ]] ||
  fallo 'VEC_POSTGRES_TEST_IMAGE debe fijarse por digest sha256'
printf 'Preflight B2 real: AD3-51..60 y material sintético presentes.\n'
[[ ${1:-} == '--preflight' ]] && exit 0

command -v "$motor" >/dev/null 2>&1 || fallo "motor de contenedor ausente: $motor"
trap limpiar EXIT
"$motor" run -d --rm --name "$contenedor" \
  -e POSTGRES_DB="$base" -e POSTGRES_HOST_AUTH_METHOD=trust \
  "$imagen" >/dev/null
esperar
[[ $(consulta "SELECT current_setting('server_version_num')") == 180004 ]] || error 'se requiere PostgreSQL 18.4'

# El SQL de preimagen se aplica solo en el contenedor propio. Un fixture que
# simule consumidores o altere migraciones incumple este contrato.
archivo "$VEC_B2_BASELINE_SQL"
[[ $(consulta "SELECT to_regprocedure('vec_contexto_actor_v1.acreditar_persona_tercero_v1(text)') IS NOT NULL AND to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_acceso_rutas_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL AND to_regclass('vec_personal.situacion_empleado_historia') IS NOT NULL") == t ]] ||
  error 'la preimagen real CA-008, Personal-017 o AD3-050a no está instalada'

for indice in "${!rutas_ad3[@]}"; do
  numero=$((indice+51))
  ruta=${rutas_ad3[$indice]}
  if [[ $numero == 60 ]]; then
    sed '$s/^COMMIT;[[:space:]]*$/ROLLBACK;/' "$repo_dir/$ruta" | \
      "$motor" exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d "$base" -o /dev/null
    [[ $(consulta "SELECT to_regprocedure('vec_autorizacion_atestada_v3.consumir_registro_empleado_b2_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL") == t ]] ||
      error 'ROLLBACK de AD3-60 dejó el consumidor instalado'
  fi
  if [[ $numero == 53 ]]; then aplicar_ref "$ref_53" "$ruta"
  elif [[ $numero == 59 ]]; then aplicar_ref "$ref_59" "$ruta"
  else archivo "$repo_dir/$ruta"; fi
done

# Las migraciones Personal se ensayan en ROLLBACK y luego COMMIT, sin DOWN.
for ruta in "$personal_dir/000018_lectura_registro_empleado_b2.up.sql" \
            "$personal_dir/000019_escritura_registro_empleado_b2.up.sql"; do
  sed '$s/^COMMIT;[[:space:]]*$/ROLLBACK;/' "$repo_dir/$ruta" | \
    "$motor" exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d "$base" -o /dev/null
  archivo "$repo_dir/$ruta"
done
[[ $(consulta "SELECT has_function_privilege('vec_personal_propietario','vec_autorizacion_atestada_v3.consumir_registro_empleado_b2_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') AND NOT has_function_privilege('vec_personal_ejecutor','vec_autorizacion_atestada_v3.consumir_registro_empleado_b2_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')") == t ]] ||
  error 'ACL de consumo AD3-60 divergente'
[[ $(consulta "SELECT NOT has_table_privilege('vec_personal_ejecutor','vec_personal.registro_empleado_b2_recibo','SELECT') AND NOT has_table_privilege('vec_personal_ejecutor','vec_personal.relacion_servicio_historia','SELECT') AND NOT has_table_privilege('vec_personal_ejecutor','vec_personal.ocupacion_empleado_historia','SELECT') AND NOT has_table_privilege('vec_personal_ejecutor','vec_personal.servicio_reconocido_historia','SELECT') AND NOT has_table_privilege('vec_personal_ejecutor','vec_personal.situacion_empleado_historia','SELECT')") == t ]] ||
  error 'ACL de historia Personal divergente'

# Los casos deben lanzar error SQL ante un negativo inesperadamente permitido.
# Su transacción positiva debe COMMIT y devolver un único recibo comprobable.
archivo "$VEC_B2_CASOS_SQL" >/dev/null 2>&1 ||
  error 'los casos V3 firmados fallaron; se ha ocultado su salida privada'
recibos=$(consulta 'SELECT count(*) FROM vec_personal.registro_empleado_b2_recibo')
[[ $recibos =~ ^[1-9][0-9]*$ ]] || error 'no existe recibo COMMIT de B2'
"$motor" restart "$contenedor" >/dev/null
esperar
[[ $(consulta 'SELECT count(*) FROM vec_personal.registro_empleado_b2_recibo') == "$recibos" ]] ||
  error 'los recibos cambiaron tras reinicio'
printf 'PG18.4 B2 real: cadena AD3-51..60, ACL, casos firmados y %s recibos persistentes tras reinicio.\n' "$recibos"
