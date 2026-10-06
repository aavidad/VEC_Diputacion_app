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
command -v python3 >/dev/null || { echo 'falta python3 para cotejar el inventario' >&2; exit 2; }
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
  if psql_pg < "$base_dir/inventario.sql" > "$evidencia/despues.json" &&
     cmp -s "$evidencia/antes.json" "$evidencia/despues.json"; then
    echo "operación rechazada; inventario sin cambios: $evidencia/despues.json" >&2
  else
    echo "operación rechazada; el inventario cambió o no se pudo leer: revisar $evidencia antes de repetir" >&2
  fi
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
familias = [('preferencias', 'consultar'), ('preferencias', 'actualizar'), ('correos', 'consultar'),
            ('correos', 'anadir'), ('correos', 'reenviar'), ('correos', 'verificar'),
            ('correos', 'activar'), ('correos', 'retirar'), ('imagen', 'consultar'), ('imagen', 'actualizar')]
superficies = [('vec_pref508a_i_ue', 'interna_corporativa'), ('vec_pref508a_e_ue', 'externa_personal')]
esperadas = {(login, f'vec_usuarios.{f}.{a}.{canal}.v1', f'vec.{f}.{a}', 'vec-usuarios', canal)
             for login, canal in superficies for f, a in familias}
esperadas.add(('vec_pref508a_i_ue', 'vec_usuarios.correos.avisos_llamamiento.interna_corporativa.v1',
               'llamamiento.emitir.v1', 'vec-usuarios', 'interna_corporativa'))


def falla(motivo):
    sys.exit('cotejo fallido: ' + motivo)


if finalizar == 'ROLLBACK':
    if antes != despues:
        falla('el ensayo cambió el inventario')
else:
    previas = {tuple(f) for f in antes['filas']}
    posteriores = {tuple(f) for f in despues['filas']}
    if antes['logins'] != despues['logins']:
        falla('cambiaron los LOGIN')
    if not previas <= posteriores:
        falla('desapareció alguna fila previa')
    if not esperadas <= posteriores:
        falla('faltan ternas de Usuarios')
    if not (posteriores - previas) <= esperadas:
        falla('hay filas nuevas fuera de lo previsto')
print('verificado:', finalizar)
PY
echo "inventario posterior: $evidencia/despues.json"
echo "copiar $evidencia a la bitácora privada, fuera de Git: /tmp puede vaciarse al reiniciar"
