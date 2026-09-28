#!/usr/bin/env bash
# Negativos locales reproducibles; no abre conexiones ni modifica bases.
set -Eeuo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
plan=$script_dir/migraciones.txt
temporal=$(mktemp -d /dev/shm/vec-piden-corte1-pruebas.XXXXXXXX)
trap 'rm -rf -- "$temporal"' EXIT
fallar() { printf 'ERROR: prueba del paquete: %s\n' "$*" >&2; exit 1; }

"$script_dir/validar_plan.sh" >/dev/null
for rol in \
  deploy/postgresql/bolsa_llamamientos/roles_calculador_politica_up.sql \
  deploy/postgresql/contratacion_temporal/roles_registrador_auditoria_up.sql
do
  grep -vFx -- "$rol" "$plan" >"$temporal/sin-rol.txt"
  if "$script_dir/validar_plan.sh" --plan "$temporal/sin-rol.txt" >/dev/null 2>&1; then
    fallar "se admitió la omisión de $rol"
  fi
done

python3 - "$plan" "$temporal/invertido.txt" <<'PY'
from pathlib import Path
import sys
source, target = map(Path, sys.argv[1:])
text = source.read_text()
a = 'deploy/postgresql/autorizacion_atestada_v3/migraciones/000096_consumidor_catalogo_plantillas_documental_ct.up.sql\n'
b = 'deploy/postgresql/contratacion_temporal/migraciones/000133_obtener_catalogo_plantillas_publicado_documental.up.sql\n'
assert a + b in text
target.write_text(text.replace(a + b, b + a, 1))
PY
if "$script_dir/validar_plan.sh" --plan "$temporal/invertido.txt" >/dev/null 2>&1; then
  fallar 'se admitió CT133 antes de AD3-96'
fi

cat >"$temporal/psql" <<'SH'
#!/usr/bin/env bash
case " $* " in
  *' --command '*)
    case ${PGSERVICE:-} in
      principal) printf 'vec_clon_piden_20260928|180004|t|%s\n' "${SIM_B49:-f}" ;;
      publica) printf 'bolsa_publica_clon_piden_20260928|180004|t|%s\n' "${SIM_PUBLICA3:-f}" ;;
      *) exit 3 ;;
    esac ;;
  *)
    sql=$(cat)
    [[ $sql == *$'\nROLLBACK;' && $sql != *$'\nCOMMIT;' ]] || exit 4
    [[ ${SIM_ACL_FAIL:-0} == 0 ]] || exit 5 ;;
esac
SH
chmod 0700 "$temporal/psql"
export PATH="$temporal:$PATH" PGSERVICE=principal
export VEC_PIDEN_CLON_DB=vec_clon_piden_20260928
export VEC_PIDEN_BOLSA_PUBLICA_PGSERVICE=publica
export VEC_PIDEN_BOLSA_PUBLICA_CLON_DB=bolsa_publica_clon_piden_20260928
"$script_dir/preflight_roles_calculador.sh" --clon >/dev/null
"$script_dir/preflight_roles_ct136.sh" --clon >/dev/null
if SIM_B49=t "$script_dir/preflight_no_go.sh" --clon >/dev/null 2>&1; then
  fallar 'B49 instalada fue admitida'
fi
if SIM_PUBLICA3=t "$script_dir/preflight_no_go.sh" --clon >/dev/null 2>&1; then
  fallar 'Pública3 instalada fue admitida'
fi
if SIM_ACL_FAIL=1 "$script_dir/preflight_roles_calculador.sh" --clon >/dev/null 2>&1; then
  fallar 'rol calculador incompatible fue admitido'
fi
if SIM_ACL_FAIL=1 "$script_dir/preflight_roles_ct136.sh" --clon >/dev/null 2>&1; then
  fallar 'rol CT136 incompatible fue admitido'
fi
printf 'PAQUETE_CORTE1_NEGATIVOS_OK: roles, AD3-96/CT133, B49, Pública3 y ACL\n'
