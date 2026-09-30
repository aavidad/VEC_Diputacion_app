#!/usr/bin/env bash
set -euo pipefail
# Solo acepta el contenedor desechable suministrado; todos los casos revierten.
contenedor=${1:?Indique el clon PostgreSQL 18 desechable}
base=${2:-postgres}
raiz=$(cd "$(dirname "$0")/../../../.." && pwd)
temporal=$(mktemp -d /dev/shm/vec-portales-tipos.XXXXXX)
trap 'rm -rf -- "$temporal"' EXIT
chmod 700 "$temporal"
umask 077
version=$(docker exec "$contenedor" psql -U postgres -d "$base" -AtX -c 'SHOW server_version_num')
[[ "$version" =~ ^18[0-9]{4}$ ]] || { echo 'Se requiere PostgreSQL 18' >&2; exit 1; }
python3 - "$raiz" "$temporal" <<'PY'
from pathlib import Path
import sys
raiz, destino=map(Path,sys.argv[1:])
archivos=[
 'deploy/postgresql/autorizacion/migraciones/000020_clausura_externa_tipos_temporales.up.sql',
 'deploy/postgresql/autorizacion_atestada_v3/migraciones/000121_clausura_externa_tipos_temporales.up.sql',
 'deploy/postgresql/bolsa_llamamientos/migraciones/000065_clausura_externa_tipos_temporales.up.sql']
correctivas=[]
for archivo in archivos:
 sql=(raiz/archivo).read_text()
 assert sql.endswith('COMMIT;\n') and sql.count('\nBEGIN;\n')==1
 correctivas.append(sql.replace('BEGIN;\n','',1)[:-len('COMMIT;\n')])
sonda=(raiz/'deploy/postgresql/autorizacion/pruebas_sql/cierre_externo_tipos_000020_000121_000065.sql').read_text()
assert sonda.count('-- PORTALES-CORRECTIVOS-AQUI\n')==1
(destino/'sonda.sql').write_text(sonda.replace('-- PORTALES-CORRECTIVOS-AQUI\n','\n'.join(correctivas)))
venenos={
 'path_publico':"ALTER FUNCTION vec_autorizacion.login_candidato_externo_v1(text,text) SET search_path=pg_catalog,public;",
 'acl_publica':"GRANT EXECUTE ON FUNCTION vec_autorizacion.login_candidato_externo_v1(text,text) TO PUBLIC;",
 'definidor':"ALTER FUNCTION vec_autorizacion.login_candidato_externo_v1(text,text) SECURITY INVOKER;",
 'cuerpo':"CREATE OR REPLACE FUNCTION vec_autorizacion.login_candidato_externo_v1(p_login text,p_grupo text) RETURNS boolean LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog AS 'SELECT false';"}
for nombre,veneno in venenos.items():
 (destino/f'{nombre}.sql').write_text('\\set ON_ERROR_STOP on\nBEGIN;\n'+veneno+'\n'+correctivas[0]+'\nROLLBACK;\n')
PY
if ! docker exec -i "$contenedor" psql -U postgres -d "$base" -X -f - < "$temporal/sonda.sql" > "$temporal/sonda.log" 2>&1; then
 tail -8 "$temporal/sonda.log" >&2
 exit 1
fi
grep -q '^PORTALES-TIPOS-CIERRE-OK$' "$temporal/sonda.log"
for caso in path_publico acl_publica definidor cuerpo; do
 if docker exec -i "$contenedor" psql -U postgres -d "$base" -X -f - < "$temporal/$caso.sql" > "$temporal/$caso.log" 2>&1; then
  echo "PORTALES-TIPOS: aceptó preimagen alterada ($caso)" >&2
  exit 1
 fi
 grep -q 'AUT20: preimagen incompatible' "$temporal/$caso.log"
done
echo 'PORTALES-TIPOS-ENSAYO-ROLLBACK-OK: 71 firmas, canon e historia, cuatro preimágenes alteradas'
