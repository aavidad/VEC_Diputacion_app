#!/usr/bin/env bash
# Instala en la principal las filas de origen AD172 de vec-server.
# Uso: ejecutar.sh --inventario | --ensayo | --aplicar
#   --ensayo  ejecuta todo y termina en ROLLBACK; el inventario no debe cambiar.
#   --aplicar exige VEC_ORIGEN_AD172_APLICAR=SI-REVISADO y termina en COMMIT.
# Variables:
#   VEC_ORIGEN_PG_CONTENEDOR  contenedor de PostgreSQL (obligatorio)
#   VEC_ORIGEN_MOTOR          podman (por defecto) o docker
#   VEC_ORIGEN_BLOQUES        bloques de ternas.tsv separados por comas; por
#                             defecto los de RRHH. «cronos» va aparte y solo
#                             se instala si se nombra.
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
if [[ $accion == --aplicar && ${VEC_ORIGEN_AD172_APLICAR:-} != SI-REVISADO ]]; then
  echo 'aplicar exige VEC_ORIGEN_AD172_APLICAR=SI-REVISADO' >&2; exit 2
fi

# Selección de bloques: solo nombres que existen en la lista.
bloques=${VEC_ORIGEN_BLOQUES:-usuarios,contratacion,bolsa,documentos,incorporacion}
[[ $bloques =~ ^[a-z]+(,[a-z]+)*$ ]] || { echo 'VEC_ORIGEN_BLOQUES no válido' >&2; exit 2; }
conocidos=$(awk -F'\t' '!/^#/ && NF {print $1}' "$base_dir/ternas.tsv" | sort -u)
IFS=, read -r -a elegidos <<< "$bloques"
for b in "${elegidos[@]}"; do
  grep -qx -- "$b" <<< "$conocidos" || { echo "bloque desconocido: $b" >&2; exit 2; }
done
ternas=$(awk -F'\t' -v sel="$bloques" 'BEGIN{n=split(sel,a,","); for(i=1;i<=n;i++) ok[a[i]]=1}
  !/^#/ && NF && ($1 in ok) {print}' "$base_dir/ternas.tsv")
[[ -n $ternas ]] || { echo 'ninguna terna seleccionada' >&2; exit 2; }

psql_pg() {
  "$motor" exec -i "$contenedor" psql -XAtq -w -v ON_ERROR_STOP=1 -h /var/run/postgresql -U postgres -d postgres \
    -v "ternas=$ternas" "$@"
}

evidencia=$(mktemp -d "${TMPDIR:-/tmp}/origen-ad172-XXXXXX")
printf '%s\n' "$ternas" > "$evidencia/ternas.tsv"
psql_pg < "$base_dir/inventario.sql" > "$evidencia/antes.json"
echo "bloques: $bloques ($(wc -l < "$evidencia/ternas.tsv") ternas)"
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
python3 - "$evidencia/antes.json" "$evidencia/despues.json" "$evidencia/ternas.tsv" "$finalizar" <<'PY'
import json
import sys
antes, despues = (json.load(open(p, encoding='utf8')) for p in sys.argv[1:3])
finalizar = sys.argv[4]
esperadas = set()
for linea in open(sys.argv[3], encoding='utf8'):
    c = linea.rstrip('\n').split('\t')
    if len(c) == 8:
        esperadas.add((c[1], c[3], c[4], c[6], c[5]))


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
        falla('faltan ternas de la lista')
    if not (posteriores - previas) <= esperadas:
        falla('hay filas nuevas fuera de la lista')
print('verificado:', finalizar)
PY
echo "inventario posterior: $evidencia/despues.json"
echo "copiar $evidencia a la bitácora privada, fuera de Git: /tmp puede vaciarse al reiniciar"
