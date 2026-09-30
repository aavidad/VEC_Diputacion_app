#!/usr/bin/env bash
set -euo pipefail

# Usa únicamente un contenedor PostgreSQL 18 desechable del operador.
# No instala la migración: cada conexión termina en ROLLBACK o aborta.
contenedor=${1:?Indique el contenedor PostgreSQL 18 desechable}
base=${2:-postgres}
raiz=$(cd "$(dirname "$0")/../../../.." && pwd)
temporal=$(mktemp -d /dev/shm/vec-ctx17-prueba.XXXXXX)
trap 'rm -rf -- "$temporal"' EXIT
chmod 700 "$temporal"
umask 077

version=$(docker exec "$contenedor" psql -U postgres -d "$base" -AtX -c 'SHOW server_version_num')
[[ "$version" =~ ^18[0-9]{4}$ ]] || { echo 'Se requiere PostgreSQL 18' >&2; exit 1; }

python3 - "$raiz" "$temporal" <<'PY'
from pathlib import Path
import sys

raiz, destino = map(Path, sys.argv[1:])
sql = (raiz / 'deploy/postgresql/contexto_actor_v1/migraciones/000017_contexto_externo_tipos_temporales.up.sql').read_text()
prueba = (raiz / 'deploy/postgresql/contexto_actor_v1/pruebas_sql/contexto_externo_tipos_temporales_000017.sql').read_text()
assert sql.endswith('COMMIT;\n')
(destino / 'adversarial.sql').write_text(sql[:-len('COMMIT;\n')] + prueba.replace('BEGIN;\n', '', 1))
firma = 'vec_contexto_actor_v1.snapshot_contexto_externo_vigente_v1(text,text,text,timestamptz)'
venenos = {
    'path_temporal': f'ALTER FUNCTION {firma} SET search_path=pg_temp,pg_catalog;',
    'path_publico': f'ALTER FUNCTION {firma} SET search_path=pg_catalog,public;',
    'acl_publica': f'GRANT EXECUTE ON FUNCTION {firma} TO PUBLIC;',
    'definidor': f'ALTER FUNCTION {firma} SECURITY INVOKER;',
}
for nombre, veneno in venenos.items():
    (destino / f'{nombre}.sql').write_text('\\set ON_ERROR_STOP on\nBEGIN;\n' + veneno + '\n' + sql.replace('BEGIN;\n', '', 1).replace('COMMIT;\n', 'ROLLBACK;\n'))
PY

docker exec -i "$contenedor" psql -U postgres -d "$base" -X -f - < "$temporal/adversarial.sql" > "$temporal/adversarial.log" 2>&1
grep -q '^CTX17-TIPOS-TEMPORALES-OK$' "$temporal/adversarial.log"
for caso in path_temporal path_publico acl_publica definidor; do
  if docker exec -i "$contenedor" psql -U postgres -d "$base" -X -f - < "$temporal/$caso.sql" > "$temporal/$caso.log" 2>&1; then
    echo "CTX17: aceptó preimagen alterada ($caso)" >&2
    exit 1
  fi
  grep -q 'CTX17: preimagen incompatible' "$temporal/$caso.log"
done
echo 'CTX17-ENSAYO-ROLLBACK-OK: dominios temporales y cuatro preimágenes incompatibles'
