#!/usr/bin/env bash
set -euo pipefail
# Clon PG18 previo a correctivas; ningún caso confirma la transacción.
contenedor=${1:?Indique un contenedor desechable}
base=${2:-postgres}
estado=${3:-usuarios}
raiz=$(cd "$(dirname "$0")/../../../.." && pwd)
raiz_candidato=${4:-$raiz}
[[ "$estado" == candidato || "$estado" == usuarios ]] || { echo 'Estado esperado: candidato o usuarios' >&2; exit 1; }
temporal=$(mktemp -d /dev/shm/vec-usuarios-tipos.XXXXXX)
trap 'rm -rf -- "$temporal"' EXIT
chmod 700 "$temporal"
umask 077
version=$(docker exec "$contenedor" psql -U postgres -d "$base" -AtX -c 'SHOW server_version_num')
[[ "$version" =~ ^18[0-9]{4}$ ]] || { echo 'Se requiere PostgreSQL 18' >&2; exit 1; }
python3 - "$raiz" "$raiz_candidato" "$temporal" "$estado" <<'PY'
from pathlib import Path
import sys
raiz,candidato,destino=map(Path,sys.argv[1:4]);estado=sys.argv[4]
def contenido(root,path):
 sql=(root/path).read_text()
 assert sql.endswith('COMMIT;\n') and sql.count('\nBEGIN;\n')==1
 return sql.replace('BEGIN;\n','',1)[:-len('COMMIT;\n')]
# Los dos órdenes comparten CTX17/AUT20/B65. AD3-121 se aplica antes o
# después de AD3-118 según el estado del clon; nunca se repite una instalada.
prefijo=[contenido(candidato,p) for p in [
 'deploy/postgresql/contexto_actor_v1/migraciones/000017_contexto_externo_tipos_temporales.up.sql',
 'deploy/postgresql/autorizacion/migraciones/000020_clausura_externa_tipos_temporales.up.sql',
 'deploy/postgresql/autorizacion_atestada_v3/migraciones/000121_clausura_externa_tipos_temporales.up.sql',
 'deploy/postgresql/bolsa_llamamientos/migraciones/000065_clausura_externa_tipos_temporales.up.sql']]
if estado=='candidato':
 prefijo.extend(contenido(raiz,p) for p in [
  'deploy/postgresql/autorizacion/migraciones/000017_perfil_usuarios_externo.up.sql',
  'deploy/postgresql/autorizacion/migraciones/000018_usuarios_externo_fechas_cero_canonicas.up.sql',
  'deploy/postgresql/autorizacion_atestada_v3/migraciones/000118_consumo_usuarios_externo.up.sql'])
correctivas=[contenido(raiz,p) for p in [
 'deploy/postgresql/autorizacion/migraciones/000021_clausura_externa_tipos_temporales.up.sql',
 'deploy/postgresql/autorizacion_atestada_v3/migraciones/000122_clausura_externa_tipos_temporales.up.sql']]
sonda=(raiz/'deploy/postgresql/autorizacion/pruebas_sql/usuarios_externo_tipos_000021_000122.sql').read_text()
assert sonda.count('-- USUARIOS-CORRECTIVOS-AQUI\n')==1
sonda=sonda.replace('BEGIN;\n','BEGIN;\n'+'\nRESET ROLE;\n'.join(prefijo)+'\nRESET ROLE;\n',1)
sonda=sonda.replace('-- USUARIOS-CORRECTIVOS-AQUI\n','\nRESET ROLE;\n'.join(correctivas))
(destino/'sonda.sql').write_text(sonda)
PY
if ! docker exec -i "$contenedor" psql -U postgres -d "$base" -X -f - < "$temporal/sonda.sql" > "$temporal/sonda.log" 2>&1; then
 tail -8 "$temporal/sonda.log" >&2
 exit 1
fi
grep -q '^USUARIOS-TIPOS-CIERRE-OK$' "$temporal/sonda.log"
echo "USUARIOS-TIPOS-ENSAYO-ROLLBACK-OK: $estado, 15 firmas, control vulnerable y contador persistente"
