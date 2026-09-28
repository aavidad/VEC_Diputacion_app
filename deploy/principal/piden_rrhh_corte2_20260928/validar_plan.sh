#!/usr/bin/env bash
# Contrasta el inventario SQL exacto y sus dependencias antes de cualquier UP.
set -Eeuo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
repo=$(git -C "$script_dir" rev-parse --show-toplevel)
fuente=da48a409b7e75249fa1b2378d612a90998aa25bd
base_publicada=7247682cbd1e6e630c86c290e3ddeca281456a94
fallar() { printf 'ERROR: plan SQL corte2: %s\n' "$*" >&2; exit 1; }
if [[ $# == 0 ]]; then
  plan=$script_dir/migraciones.txt
elif [[ $# == 2 && $1 == --plan && -f $2 ]]; then
  plan=$2
else
  fallar 'uso: validar_plan.sh [--plan FICHERO para ensayo aislado]'
fi

mapfile -t rutas < <(grep -vE '^[[:space:]]*(#|$)' "$plan")
[[ ${#rutas[@]} == 29 ]] || fallar 'se esperaban 27 UP y dos deltas DBA'
declare -A posicion=()
indice=0
for ruta in "${rutas[@]}"; do
  [[ $ruta == deploy/postgresql/* && -f $repo/$ruta ]] \
    || fallar "ruta ausente o fuera de deploy/postgresql: $ruta"
  case $ruta in
    */000049_publicacion_cese_b10.up.sql|*/000003_publicacion_cese_replay.up.sql)
      fallar "migración NO-GO incluida: $ruta" ;;
  esac
  nombre=${ruta##*/}
  [[ -z ${posicion[$nombre]+existe} ]] || fallar "nombre repetido: $nombre"
  posicion[$nombre]=$indice
  indice=$((indice + 1))
done

esperados=$(git -C "$repo" diff --name-only "$base_publicada" "$fuente" \
  -- deploy/postgresql | grep -E '(/migraciones/.*\.up\.sql$|/roles_[^/]*_up\.sql$)' | LC_ALL=C sort)
actuales=$(printf '%s\n' "${rutas[@]}" \
  | grep -E '(/migraciones/.*\.up\.sql$|/roles_[^/]*_up\.sql$)' | LC_ALL=C sort)
[[ $actuales == "$esperados" ]] \
  || fallar 'el conjunto UP/deltas DBA difiere del delta exacto entre base publicada y fuente fijada'

antes() {
  local anterior=$1 posterior=$2 a b
  a=${posicion[$anterior]:-}
  b=${posicion[$posterior]:-}
  [[ $a =~ ^[0-9]+$ && $b =~ ^[0-9]+$ && $a -lt $b ]] \
    || fallar "dependencia invertida o ausente: $anterior → $posterior"
}

# Secuencia del núcleo V3: cada consumidor modifica la preimagen del siguiente.
antes 000090_consumidor_historial_propio_bolsa.up.sql 000091_consumidor_consulta_auditoria_rrhh.up.sql
antes 000091_consumidor_consulta_auditoria_rrhh.up.sql 000092_consumidor_reincorporacion_titular_ct.up.sql
antes 000092_consumidor_reincorporacion_titular_ct.up.sql 000093_consumidor_politica_ofertas_bolsa.up.sql
antes 000093_consumidor_politica_ofertas_bolsa.up.sql 000094_consumidor_catalogo_plantillas_ct.up.sql
antes 000094_consumidor_catalogo_plantillas_ct.up.sql 000095_consumidor_politica_cese_bolsa.up.sql
antes 000095_consumidor_politica_cese_bolsa.up.sql 000096_consumidor_catalogo_plantillas_documental_ct.up.sql
antes 000096_consumidor_catalogo_plantillas_documental_ct.up.sql 000097_consumidor_consulta_politica_ofertas_bolsa.up.sql
antes 000097_consumidor_consulta_politica_ofertas_bolsa.up.sql 000098_consumidor_lectura_reincorporacion_ct.up.sql
antes 000098_consumidor_lectura_reincorporacion_ct.up.sql 000099_ambito_organizacion_plantillas_ct.up.sql

# Dependencias entre módulos y funciones consultadas por las precondiciones UP.
antes 000090_consumidor_historial_propio_bolsa.up.sql 000044_historial_propio_candidato.up.sql
antes 000091_consumidor_consulta_auditoria_rrhh.up.sql 000132_consulta_auditoria_ct.up.sql
antes 000091_consumidor_consulta_auditoria_rrhh.up.sql 000048_consulta_auditoria_participacion.up.sql
antes 000092_consumidor_reincorporacion_titular_ct.up.sql 000130_reincorporacion_titular.up.sql
antes 000093_consumidor_politica_ofertas_bolsa.up.sql 000047_politica_ofertas_ejemplo.up.sql
antes 000045_restriccion_global_cese.up.sql 000129_verificacion_cese_bolsa.up.sql
antes 000045_restriccion_global_cese.up.sql 000046_reincorporacion_titular_ct.up.sql
antes 000130_reincorporacion_titular.up.sql 000046_reincorporacion_titular_ct.up.sql
antes 000094_consumidor_catalogo_plantillas_ct.up.sql 000131_catalogo_plantillas_documentos.up.sql
antes 000045_restriccion_global_cese.up.sql 000050_lectura_estado_cese.up.sql
antes 000045_restriccion_global_cese.up.sql 000053_mi_bolsa_disponibilidad_maxima.up.sql
antes 000131_catalogo_plantillas_documentos.up.sql 000133_obtener_catalogo_plantillas_publicado_documental.up.sql
antes 000096_consumidor_catalogo_plantillas_documental_ct.up.sql 000133_obtener_catalogo_plantillas_publicado_documental.up.sql
antes 000093_consumidor_politica_ofertas_bolsa.up.sql 000097_consumidor_consulta_politica_ofertas_bolsa.up.sql
antes 000047_politica_ofertas_ejemplo.up.sql 000051_consulta_politica_ofertas_v3.up.sql
antes 000097_consumidor_consulta_politica_ofertas_bolsa.up.sql 000051_consulta_politica_ofertas_v3.up.sql
antes roles_calculador_politica_up.sql 000051_consulta_politica_ofertas_v3.up.sql
antes 000098_consumidor_lectura_reincorporacion_ct.up.sql 000134_lectura_reincorporacion_titular.up.sql
antes 000130_reincorporacion_titular.up.sql 000134_lectura_reincorporacion_titular.up.sql
antes 000047_politica_ofertas_ejemplo.up.sql 000054_plazo_ofertas_48_horas.up.sql
antes 000051_consulta_politica_ofertas_v3.up.sql 000054_plazo_ofertas_48_horas.up.sql
antes 000099_ambito_organizacion_plantillas_ct.up.sql 000135_ambito_organizacion_plantillas.up.sql
antes 000131_catalogo_plantillas_documentos.up.sql 000135_ambito_organizacion_plantillas.up.sql
antes roles_registrador_auditoria_up.sql 000136_auditoria_frontera_auditoria_ruta_exacta.up.sql
calculador_pos=${posicion[roles_calculador_politica_up.sql]}
b51_pos=${posicion[000051_consulta_politica_ofertas_v3.up.sql]}
[[ $b51_pos -eq $((calculador_pos + 1)) ]] \
  || fallar 'B51 debe seguir inmediatamente al delta DBA del calculador'
rol_pos=${posicion[roles_registrador_auditoria_up.sql]}
ct136_pos=${posicion[000136_auditoria_frontera_auditoria_ruta_exacta.up.sql]}
[[ $ct136_pos -eq $((rol_pos + 1)) ]] \
  || fallar 'CT136 debe seguir inmediatamente al delta DBA del rol'

printf 'PLAN_SQL_CORTE2_OK: 27 UP, dos roles DBA y dependencias causales\n'
