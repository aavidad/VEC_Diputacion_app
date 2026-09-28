#!/usr/bin/env bash
# Negativos locales del inventario, sin PostgreSQL ni exportación.
set -Eeuo pipefail
script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
tmp=$(mktemp -d /dev/shm/vec-piden-corte3-plan.XXXXXXXX)
trap 'rm -rf -- "$tmp"' EXIT
fallar() { printf 'ERROR: prueba plan c3: %s\n' "$*" >&2; exit 1; }
plan=$script_dir/migraciones.txt

"$script_dir/validar_plan.sh" --provisional >/dev/null
if "$script_dir/validar_plan.sh" >/dev/null 2>&1; then
  fallar 'se aceptó exportación sin hash final'
fi
for ruta in \
  deploy/postgresql/bolsa_llamamientos/roles_calculador_politica_up.sql \
  deploy/postgresql/contratacion_temporal/roles_registrador_auditoria_up.sql \
  deploy/postgresql/autorizacion_atestada_v3/migraciones/000100_documental_tres_ambitos_ct.up.sql \
  deploy/postgresql/autorizacion_atestada_v3/migraciones/000101_consumidor_consulta_reincorporacion_titular_bolsa.up.sql \
  deploy/postgresql/bolsa_llamamientos/migraciones/000056_motivo_traza_auditoria_participacion.up.sql
do
  grep -vFx -- "$ruta" "$plan" >"$tmp/omitido.txt"
  if "$script_dir/validar_plan.sh" --plan "$tmp/omitido.txt" --provisional >/dev/null 2>&1; then
    fallar "se admitió omitir $ruta"
  fi
done
python3 - "$plan" "$tmp/invertido.txt" <<'PY'
from pathlib import Path
import sys
source, target = map(Path, sys.argv[1:])
text = source.read_text()
a = 'deploy/postgresql/autorizacion_atestada_v3/migraciones/000100_documental_tres_ambitos_ct.up.sql\n'
b = 'deploy/postgresql/contratacion_temporal/migraciones/000137_documental_tres_ambitos.up.sql\n'
assert a + b in text
target.write_text(text.replace(a + b, b + a, 1))
PY
if "$script_dir/validar_plan.sh" --plan "$tmp/invertido.txt" --provisional >/dev/null 2>&1; then
  fallar 'se admitió CT137 antes de AD3-100'
fi
for ruta in \
  deploy/postgresql/bolsa_llamamientos/migraciones/000057_motivo_catalogado.up.sql \
  deploy/postgresql/autorizacion_atestada_v3/migraciones/000102_consumidor_motivo_catalogado.up.sql \
  deploy/postgresql/autorizacion_atestada_v3/migraciones/000103_consumidor_motivo_catalogado.up.sql \
  deploy/postgresql/autorizacion_atestada_v3/migraciones/000104_consumidor_motivo_catalogado.up.sql \
  deploy/postgresql/autorizacion_atestada_v3/migraciones/000105_consumidor_motivo_catalogado.up.sql
do
  cp -- "$plan" "$tmp/futuro.txt"
  printf '%s\n' "$ruta" >>"$tmp/futuro.txt"
  if "$script_dir/validar_plan.sh" --plan "$tmp/futuro.txt" --provisional >/dev/null 2>&1; then
    fallar "se admitió sin revisar: $ruta"
  fi
done
cat >"$tmp/psql" <<'SH'
#!/usr/bin/env bash
if [[ $* == *'B56: motivo unido'* ]]; then
  printf 'vec_clon_piden_20260928|180004|t|%s\n' "${SIM_B56_PREIMAGEN:-t}"
  exit 0
fi
case ${PGSERVICE:-} in
  principal) printf 'vec_clon_piden_20260928|180004|t|%s\n' "${SIM_B49:-f}" ;;
  publica) printf 'bolsa_publica_clon_piden_20260928|180004|t|%s\n' "${SIM_PUBLICA3:-f}" ;;
  *) exit 3 ;;
esac
SH
chmod 0700 "$tmp/psql"
export PATH="$tmp:$PATH" PGSERVICE=principal
export VEC_PIDEN_CLON_DB=vec_clon_piden_20260928
export VEC_PIDEN_BOLSA_PUBLICA_PGSERVICE=publica
export VEC_PIDEN_BOLSA_PUBLICA_CLON_DB=bolsa_publica_clon_piden_20260928
"$script_dir/preflight_no_go.sh" --clon >/dev/null
if SIM_B49=t "$script_dir/preflight_no_go.sh" --clon >/dev/null 2>&1; then
  fallar 'se admitió B49 instalada'
fi
if SIM_PUBLICA3=t "$script_dir/preflight_no_go.sh" --clon >/dev/null 2>&1; then
  fallar 'se admitió Pública3 instalada'
fi
"$script_dir/preflight_b56.sh" --clon >/dev/null
if SIM_B56_PREIMAGEN=f "$script_dir/preflight_b56.sh" --clon >/dev/null 2>&1; then
  fallar 'se admitió preimagen B56 incompatible'
fi
printf 'PLAN_CORTE3_NEGATIVOS_OK: pin, omisiones, orden, B56/preimagen, B57/AD3-102…105 excluidos y B49/Pública3\n'
