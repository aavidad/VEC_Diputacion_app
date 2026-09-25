#!/usr/bin/env bash
set -euo pipefail
umask 077

accion=${1:-}
case "$accion" in
  --preparar-rollback|--preparar-commit|--activar-rollback|--activar-commit|--inventario) ;;
  *) echo 'uso: ejecutar.sh --inventario|--preparar-rollback|--preparar-commit|--activar-rollback|--activar-commit' >&2; exit 2 ;;
esac
base_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)
repo_dir=$(cd "$base_dir/../../.." && pwd -P)
: "${VEC_D7_EVIDENCIA_DIR:?falta directorio de evidencia privado}"
evidencia=$(realpath -e "$VEC_D7_EVIDENCIA_DIR")
[[ -d "$evidencia" && ! -L "$VEC_D7_EVIDENCIA_DIR" && $(stat -c %a "$evidencia") == 700 && $(stat -c %u "$evidencia") == "$(id -u)" ]] || {
  echo 'evidencia debe ser directorio privado 0700 propio' >&2; exit 2;
}
case "$evidencia/" in "$repo_dir/"*) echo 'evidencia dentro del repositorio' >&2; exit 2;; esac

if [[ "$accion" == *-commit && ${VEC_D7_APLICAR:-} != SI-D7-REVISADO ]]; then
  echo 'COMMIT exige VEC_D7_APLICAR=SI-D7-REVISADO' >&2; exit 2
fi

if [[ -n ${VEC_D7_POSTGRES_CONTAINER:-} ]]; then
  [[ $VEC_D7_POSTGRES_CONTAINER =~ ^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$ ]] || exit 2
  for nombre in PGSERVICE PGSERVICEFILE PGPASSFILE PGHOST PGHOSTADDR PGPORT PGDATABASE PGUSER PGPASSWORD PGOPTIONS; do
    [[ ! -v $nombre ]] || { echo "transporte contenedor incompatible con $nombre" >&2; exit 2; }
  done
  psql_d7() {
    docker exec -i "$VEC_D7_POSTGRES_CONTAINER" psql -XAtq -w -v ON_ERROR_STOP=1 \
      -h /var/run/postgresql -U postgres -d postgres "$@"
  }
else
  : "${PGSERVICE:?falta PGSERVICE privado}"
  : "${PGSERVICEFILE:?falta PGSERVICEFILE privado}"
  : "${PGPASSFILE:?falta PGPASSFILE privado}"
  [[ $PGSERVICE =~ ^[A-Za-z0-9_-]+$ ]] || exit 2
  for nombre in PGHOST PGHOSTADDR PGPORT PGDATABASE PGUSER PGPASSWORD PGOPTIONS; do
    [[ ! -v $nombre ]] || { echo "servicio privado incompatible con $nombre" >&2; exit 2; }
  done
  for fichero in "$PGSERVICEFILE" "$PGPASSFILE"; do
    [[ -f $fichero && ! -L $fichero && $(stat -c %a "$fichero") == 600 && $(stat -c %u "$fichero") == "$(id -u)" ]] || {
      echo 'fichero PG ausente o no privado 0600' >&2; exit 2;
    }
    case "$(realpath -e "$fichero")" in "$repo_dir/"*) echo 'fichero PG dentro del repositorio' >&2; exit 2;; esac
  done
  python3 - "$PGSERVICEFILE" "$PGSERVICE" <<'PY'
import configparser
import sys
p = configparser.ConfigParser(interpolation=None)
with open(sys.argv[1], encoding='utf8') as f:
    p.read_file(f)
if sys.argv[2] not in p:
    raise SystemExit('servicio PG ausente')
s = p[sys.argv[2]]
if 'password' in s or s.get('dbname') != 'postgres' or not s.get('host', '').startswith('/'):
    raise SystemExit('servicio PG requiere socket local de postgres y PGPASSFILE')
if any(x in ('.', '..') for x in s['host'].split('/')) or not s.get('port', '').isdigit() or not 1 <= int(s['port']) <= 65535 or not s.get('user'):
    raise SystemExit('servicio PG incompleto o invalido')
PY
  psql_d7() { psql -XAtq -w -v ON_ERROR_STOP=1 "$@"; }
fi

puerta=$(printf '%s\n' "SELECT current_setting('server_version_num')||'|'||" \
  "(SELECT rolsuper::text FROM pg_roles WHERE rolname=session_user)||'|'||" \
  "(session_user=current_user)::text||'|'||" \
  "(inet_server_addr() IS NULL)::text||'|'||current_database();" | psql_d7)
IFS='|' read -r version super mismo local base <<< "$puerta"
[[ $version =~ ^[0-9]{6}$ && $version -ge 180000 && $version -lt 190000 && $super == true && $mismo == true && $local == true && $base == postgres ]] || {
  echo 'D7 exige PostgreSQL 18, DBA, socket local y base postgres' >&2; exit 2;
}

marca=$(date -u +%Y%m%dT%H%M%SZ)-$$
antes="$evidencia/d7-antes-$marca.json"
despues="$evidencia/d7-despues-$marca.json"
psql_d7 < "$base_dir/inventario.sql" > "$antes"
echo "inventario privado: $antes"
[[ $accion != --inventario ]] || exit 0

modo=preparar
[[ $accion == --activar-* ]] && modo=activar
finalizar=ROLLBACK
[[ $accion == *-commit ]] && finalizar=COMMIT
if ! psql_d7 -v "d7_modo=$modo" -v "d7_finalizar=$finalizar" < "$base_dir/operacion.sql" >/dev/null; then
  psql_d7 < "$base_dir/inventario.sql" > "$despues" || true
  echo "operacion D7 rechazada; revisar inventario privado $despues" >&2
  exit 1
fi
psql_d7 < "$base_dir/inventario.sql" > "$despues" || {
  echo 'postinventario inaccesible; resultado transaccional incierto' >&2; exit 1;
}
python3 - "$antes" "$despues" "$modo" "$finalizar" <<'PY'
import json
import sys
before, after = (json.load(open(p, encoding='utf8')) for p in sys.argv[1:3])
mode, finish = sys.argv[3:]
if finish == 'ROLLBACK':
    assert before == after, 'ROLLBACK cambio el inventario'
else:
    expected = {"vec_personal_d7_asignacion": mode == 'activar',
                "vec_personal_d7_auditoria_frontera": mode == 'activar'}
    for name, can_login in expected.items():
        assert after['cuentas'][name]['login'] == can_login
        assert after['cuentas'][name]['miembros'] == 1
    assert before['f4'] == after['f4']
print('D7 verificado:', mode, finish)
PY
echo "postinventario privado: $despues"
