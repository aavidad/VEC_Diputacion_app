#!/usr/bin/env bash
# Exclusivo para una base PG18 desechable cuyo nombre termina en _clon_piden_20260928.
set -Eeuo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
repo=$(git -C "$script_dir" rev-parse --show-toplevel)
base=da48a409b7e75249fa1b2378d612a90998aa25bd
fallar() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }
[[ ${1:-} == --aplicar-en-clon && $# == 1 ]] || fallar 'uso: ensayar_clon.sh --aplicar-en-clon'
[[ -n ${PGSERVICE:-} && -n ${VEC_PIDEN_CLON_DB:-} ]] || fallar 'faltan PGSERVICE o VEC_PIDEN_CLON_DB'
[[ $VEC_PIDEN_CLON_DB == *_clon_piden_20260928 ]] || fallar 'nombre de clon no autorizado'
command -v psql >/dev/null || fallar 'falta psql'
git -C "$repo" merge-base --is-ancestor "$base" HEAD || fallar 'checkout ajeno al candidato'
git -C "$repo" diff --quiet "$base" HEAD -- . \
  ':(exclude)deploy/principal/piden_rrhh_corte2_20260928/**' \
  || fallar 'fuente distinta del corte 2 fijado: revisar el plan'
[[ -z $(git -C "$repo" status --porcelain) ]] || fallar 'checkout sucio'
"$script_dir/validar_plan.sh"
"$script_dir/preflight_roles_calculador.sh" --clon
"$script_dir/preflight_roles_ct136.sh" --clon

# PGSERVICE/PGPASSFILE se resuelven fuera de Git. No pasar DSN ni clave en argumentos.
conexion=(psql -X --no-psqlrc --set=ON_ERROR_STOP=1 --no-align --tuples-only)
actual=$("${conexion[@]}" --command 'SELECT current_database()')
[[ $actual == "$VEC_PIDEN_CLON_DB" ]] || fallar "la conexión apunta a $actual"
version=$("${conexion[@]}" --command 'SHOW server_version_num')
[[ $version =~ ^18[0-9]{4}$ ]] || fallar "se requiere PG18; recibido $version"

# El publicador B10 separado es NO-GO: ACK sin prueba de publicación y fuente
# V2/documental sin componer. Validar toda la lista antes del primer UP.
anterior=''
rol_calculador='deploy/postgresql/bolsa_llamamientos/roles_calculador_politica_up.sql'
b51='deploy/postgresql/bolsa_llamamientos/migraciones/000051_consulta_politica_ofertas_v3.up.sql'
rol='deploy/postgresql/contratacion_temporal/roles_registrador_auditoria_up.sql'
ct136='deploy/postgresql/contratacion_temporal/migraciones/000136_auditoria_frontera_auditoria_ruta_exacta.up.sql'
visto_calculador=false
visto_b51=false
visto_rol=false
visto_ct136=false
while IFS= read -r ruta; do
  [[ -n $ruta && $ruta != \#* ]] || continue
  case $ruta in
    */000049_publicacion_cese_b10.up.sql|*/000003_publicacion_cese_replay.up.sql)
      fallar "migración NO-GO en plan: $ruta" ;;
  esac
  if [[ $ruta == "$rol_calculador" ]]; then
    [[ $visto_calculador == false ]] || fallar 'delta de rol calculador duplicado'
    visto_calculador=true
  elif [[ $ruta == "$b51" ]]; then
    [[ $anterior == "$rol_calculador" && $visto_b51 == false ]] \
      || fallar 'B51 debe seguir inmediatamente al delta DBA de rol calculador'
    visto_b51=true
  elif [[ $ruta == "$rol" ]]; then
    [[ $visto_rol == false ]] || fallar 'delta de rol CT136 duplicado'
    visto_rol=true
  elif [[ $ruta == "$ct136" ]]; then
    [[ $anterior == "$rol" && $visto_ct136 == false ]] \
      || fallar 'CT136 debe seguir inmediatamente al delta DBA de rol'
    visto_ct136=true
  elif [[ $ruta != deploy/postgresql/*/*.up.sql ]]; then
    fallar "ruta no válida: $ruta"
  fi
  [[ -f $repo/$ruta ]] || fallar "falta SQL: $ruta"
  anterior=$ruta
done <"$script_dir/migraciones.txt"
[[ $visto_calculador == true && $visto_b51 == true \
   && $visto_rol == true && $visto_ct136 == true ]] \
  || fallar 'falta B51/CT136 o alguno de sus deltas DBA'

while IFS= read -r ruta; do
  [[ -n $ruta && $ruta != \#* ]] || continue
  printf 'Aplicando en clon: %s\n' "$ruta"
  "${conexion[@]}" --file "$repo/$ruta" >/dev/null
done <"$script_dir/migraciones.txt"
printf 'CLON_PG18_UP_OK fuente=%s paquete=%s base=%s\n' \
  "$base" "$(git -C "$repo" rev-parse HEAD)" "$actual"
