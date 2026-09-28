#!/usr/bin/env bash
# Inventario SQL sin efectos. El modo provisional no habilita exportación.
set -Eeuo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
repo=$(git -C "$script_dir" rev-parse --show-toplevel)
base=7247682cbd1e6e630c86c290e3ddeca281456a94
provisional=8f9196fc6988561a8c46e61d1984fb8551b0ee9e
plan=$script_dir/migraciones.txt
modo=final
fallar() { printf 'ERROR: plan SQL corte3: %s\n' "$*" >&2; exit 1; }
case ${1:-} in
  '') ;;
  --provisional) [[ $# == 1 ]] || fallar 'argumentos adicionales'; modo=provisional ;;
  --plan) [[ $# == 3 && -f $2 && $3 == --provisional ]] || fallar 'uso: validar_plan.sh [--provisional] o --plan FICHERO --provisional'; plan=$2; modo=provisional ;;
  *) fallar 'uso: validar_plan.sh [--provisional] o --plan FICHERO --provisional' ;;
esac
if [[ $modo == final ]]; then
  [[ -f $script_dir/fuente_final.txt ]] || fallar 'falta fuente_final.txt ratificado por Dirección; NO EXPORTAR'
  fuente=$(cat "$script_dir/fuente_final.txt")
  [[ $fuente =~ ^[0-9a-f]{40}$ ]] || fallar 'hash final inválido'
else
  fuente=$provisional
fi
[[ $(git -C "$repo" rev-parse "$fuente^{commit}") == "$fuente" ]] || fallar 'fuente no encontrada'
git -C "$repo" merge-base --is-ancestor "$fuente" HEAD || fallar 'checkout no desciende de la fuente'
git -C "$repo" diff --quiet "$fuente" HEAD -- . \
  ':(exclude)deploy/principal/piden_rrhh_corte3_20260928/**' \
  || fallar 'cambios de producto posteriores al hash fuente'

mapfile -t rutas < <(grep -vE '^[[:space:]]*(#|$)' "$plan")
[[ ${#rutas[@]} -ge 34 ]] || fallar 'faltan UP, B56 o deltas DBA previos'
declare -A posicion=()
for i in "${!rutas[@]}"; do
  ruta=${rutas[$i]}
  [[ $ruta == deploy/postgresql/* && -f $repo/$ruta ]] || fallar "ruta ausente: $ruta"
  [[ $ruta == */migraciones/*.up.sql || $ruta == */roles_*_up.sql ]] || fallar "ruta SQL impropia: $ruta"
  case $ruta in
    */000049_publicacion_cese_b10.up.sql|*/000003_publicacion_cese_replay.up.sql)
      fallar "B49/Pública3 NO-GO: $ruta" ;;
    deploy/postgresql/bolsa_llamamientos/migraciones/000057_*.up.sql|deploy/postgresql/autorizacion_atestada_v3/migraciones/00010[2345]_*.up.sql)
      fallar "B57/AD3-102…105 pertenecen al corte 4: $ruta" ;;
  esac
  [[ -z ${posicion[$ruta]+existe} ]] || fallar "ruta repetida: $ruta"
  posicion[$ruta]=$i
done
esperados=$(git -C "$repo" diff --name-only "$base" "$fuente" -- deploy/postgresql \
  | grep -E '(/migraciones/.*\.up\.sql$|/roles_[^/]*_up\.sql$)' \
  | grep -Ev '^deploy/postgresql/(bolsa_llamamientos/migraciones/(000049_publicacion_cese_b10|000057_[^/]*)|bolsa_publica/migraciones/000003_publicacion_cese_replay|autorizacion_atestada_v3/migraciones/00010[2345]_[^/]*)\.up\.sql$' \
  | LC_ALL=C sort)
actuales=$(printf '%s\n' "${rutas[@]}" | LC_ALL=C sort)
[[ $actuales == "$esperados" ]] || fallar 'plan distinto del delta Git completo UP + roles DBA; revisar B56 y toda dependencia nueva'
antes() {
  local a=$1 b=$2
  [[ -n ${posicion[$a]+existe} && -n ${posicion[$b]+existe} && ${posicion[$a]} -lt ${posicion[$b]} ]] \
    || fallar "dependencia ausente o invertida: $a → $b"
}
ad3=deploy/postgresql/autorizacion_atestada_v3/migraciones
ct=deploy/postgresql/contratacion_temporal/migraciones
bolsa=deploy/postgresql/bolsa_llamamientos/migraciones
antes "$ad3/000096_consumidor_catalogo_plantillas_documental_ct.up.sql" "$ct/000133_obtener_catalogo_plantillas_publicado_documental.up.sql"
antes deploy/postgresql/bolsa_llamamientos/roles_calculador_politica_up.sql "$bolsa/000051_consulta_politica_ofertas_v3.up.sql"
antes deploy/postgresql/contratacion_temporal/roles_registrador_auditoria_up.sql "$ct/000136_auditoria_frontera_auditoria_ruta_exacta.up.sql"
for n in 090 091 092 093 094 095 096 097 098 099 100; do
  siguiente=$(printf '%03d' "$((10#$n + 1))")
  a=$(printf '%s\n' "${rutas[@]}" | grep -m1 "^$ad3/000$n" || true)
  b=$(printf '%s\n' "${rutas[@]}" | grep -m1 "^$ad3/000$siguiente" || true)
  [[ -n $a && -n $b ]] || fallar "falta secuencia AD3-$n/$siguiente"
  antes "$a" "$b"
done
antes "$ad3/000100_documental_tres_ambitos_ct.up.sql" "$ct/000137_documental_tres_ambitos.up.sql"
antes "$ct/000135_ambito_organizacion_plantillas.up.sql" "$ct/000137_documental_tres_ambitos.up.sql"
antes "$ad3/000101_consumidor_consulta_reincorporacion_titular_bolsa.up.sql" "$bolsa/000055_lectura_reincorporacion_titular_v3.up.sql"
antes "$bolsa/000046_reincorporacion_titular_ct.up.sql" "$bolsa/000055_lectura_reincorporacion_titular_v3.up.sql"
antes "$bolsa/000048_consulta_auditoria_participacion.up.sql" "$bolsa/000056_motivo_traza_auditoria_participacion.up.sql"
antes "$bolsa/000055_lectura_reincorporacion_titular_v3.up.sql" "$bolsa/000056_motivo_traza_auditoria_participacion.up.sql"
for par in \
  'deploy/postgresql/bolsa_llamamientos/roles_calculador_politica_up.sql|deploy/postgresql/bolsa_llamamientos/migraciones/000051_consulta_politica_ofertas_v3.up.sql' \
  'deploy/postgresql/contratacion_temporal/roles_registrador_auditoria_up.sql|deploy/postgresql/contratacion_temporal/migraciones/000136_auditoria_frontera_auditoria_ruta_exacta.up.sql'; do
  IFS='|' read -r a b <<<"$par"
  [[ ${posicion[$b]} -eq $((${posicion[$a]} + 1)) ]] || fallar "rol DBA debe preceder inmediatamente a $b"
done
printf 'PLAN_SQL_CORTE3_%s_OK: %d entradas; fuente=%s\n' "${modo^^}" "${#rutas[@]}" "$fuente"
