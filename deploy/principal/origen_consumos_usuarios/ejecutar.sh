#!/usr/bin/env bash
# Instala en la principal las filas de origen AD172 de Usuarios.
# Uso: ejecutar.sh --inventario | --ensayo | --aplicar
#   --ensayo  ejecuta todo y termina en ROLLBACK; el inventario no debe cambiar.
#   --aplicar exige VEC_ORIGEN_USUARIOS_APLICAR=SI-REVISADO y termina en COMMIT.
# Transporte: psql dentro del contenedor de PostgreSQL por su socket local.
#   VEC_ORIGEN_PG_CONTENEDOR  nombre del contenedor (obligatorio)
#   VEC_ORIGEN_MOTOR          podman (por defecto) o docker
set -euo pipefail
umask 077

accion=${1:-}
case "$accion" in
  --inventario|--ensayo|--aplicar) ;;
  *) echo 'uso: ejecutar.sh --inventario|--ensayo|--aplicar' >&2; exit 2 ;;
esac
base_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)
motor=${VEC_ORIGEN_MOTOR:-podman}
[[ $motor == podman || $motor == docker ]] || { echo 'VEC_ORIGEN_MOTOR: podman o docker' >&2; exit 2; }
contenedor=${VEC_ORIGEN_PG_CONTENEDOR:?falta VEC_ORIGEN_PG_CONTENEDOR}
[[ $contenedor =~ ^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$ ]] || { echo 'nombre de contenedor no válido' >&2; exit 2; }
if [[ $accion == --aplicar && ${VEC_ORIGEN_USUARIOS_APLICAR:-} != SI-REVISADO ]]; then
  echo 'aplicar exige VEC_ORIGEN_USUARIOS_APLICAR=SI-REVISADO' >&2; exit 2
fi

psql_pg() {
  "$motor" exec -i "$contenedor" psql -XAtq -w -v ON_ERROR_STOP=1 -h /var/run/postgresql -U postgres -d postgres "$@"
}

evidencia=$(mktemp -d "${TMPDIR:-/tmp}/origen-usuarios-XXXXXX")
psql_pg < "$base_dir/inventario.sql" > "$evidencia/antes.json"
echo "inventario: $evidencia/antes.json"
[[ $accion != --inventario ]] || exit 0

finalizar=ROLLBACK
[[ $accion == --aplicar ]] && finalizar=COMMIT
if ! psql_pg -v "finalizar=$finalizar" < "$base_dir/operacion.sql" > "$evidencia/salida.txt" 2> "$evidencia/error.txt"; then
  cat "$evidencia/error.txt" >&2
  psql_pg < "$base_dir/inventario.sql" > "$evidencia/despues.json" || true
  echo "operación rechazada sin cambios; inventario posterior: $evidencia/despues.json" >&2
  exit 1
fi
cat "$evidencia/salida.txt"
psql_pg < "$base_dir/inventario.sql" > "$evidencia/despues.json" || {
  echo 'inventario posterior inaccesible; comprobar a mano antes de repetir' >&2; exit 1;
}
python3 - "$evidencia/antes.json" "$evidencia/despues.json" "$finalizar" <<'PY'
import json
import sys
antes, despues = (json.load(open(p, encoding='utf8')) for p in sys.argv[1:3])
finalizar = sys.argv[3]
if finalizar == 'ROLLBACK':
    assert antes == despues, 'el ensayo cambió el inventario'
else:
    assert antes['logins'] == despues['logins'], 'cambiaron los LOGIN'
    previas = {tuple(f) for f in antes['filas']}
    posteriores = {tuple(f) for f in despues['filas']}
    assert previas <= posteriores, 'desapareció alguna fila previa'
    nuevas = posteriores - previas
    assert all(f[0] in ('vec_pref508a_i_ue', 'vec_pref508a_e_ue') and f[3] == 'vec-usuarios' for f in nuevas), 'fila nueva fuera de lo previsto'
    propias = [f for f in posteriores if f[0] in ('vec_pref508a_i_ue', 'vec_pref508a_e_ue')]
    assert len(propias) == 20, 'no están las 20 ternas de Usuarios'
print('verificado:', finalizar)
PY
echo "inventario posterior: $evidencia/despues.json"
