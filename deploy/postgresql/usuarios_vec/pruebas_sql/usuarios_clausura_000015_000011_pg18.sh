#!/usr/bin/env bash
set -euo pipefail
# Prepara un preview para el dueño del clon; no conecta ni aplica SQL.
# Uso: script <directorio_privado_de_salida>
destino=${1:?Indique un directorio privado para el preview}
raiz=$(cd "$(dirname "$0")/../../../.." && pwd)
mkdir -p -- "$destino"
chmod 700 "$destino"
umask 077
python3 - "$raiz" "$destino" <<'PY'
from pathlib import Path
import sys
raiz,destino=map(Path,sys.argv[1:])
correctivas=[]
for ruta in ['deploy/postgresql/autorizacion_atestada_v3/migraciones/000124_clausura_tipos_temporales.up.sql',
             'deploy/postgresql/documentos/migraciones/000011_clausura_tipos_temporales.up.sql',
             'deploy/postgresql/usuarios_vec/migraciones/000015_clausura_tipos_temporales.up.sql']:
 sql=(raiz/ruta).read_text()
 assert sql.endswith('COMMIT;\n') and sql.count('\nBEGIN;\n')==1
 correctivas.append(sql.replace('BEGIN;\n','',1)[:-len('COMMIT;\n')])
sonda=(raiz/'deploy/postgresql/usuarios_vec/pruebas_sql/usuarios_clausura_000015_000011.sql').read_text()
marca='-- U15-D11-CORRECTIVOS-AQUI\n'
assert sonda.count(marca)==1 and sonda.count('\nBEGIN;\n')==1 and sonda.count('\nROLLBACK;\n')==1
preview=sonda.replace(marca,'\nRESET ROLE;\n'.join(correctivas))
(destino/'usuarios-clausura-external-preview.sql').write_text(preview)
(destino/'usuarios-clausura-internal-preview.sql').write_text(preview.replace('GRANT vec_usuarios_ejecutor_externo TO prueba_usuarios_clausura','GRANT vec_usuarios_ejecutor_interno TO prueba_usuarios_clausura'))
PY
printf '%s\n' "PREVIEW: $destino/usuarios-clausura-{external,internal}-preview.sql; ejecutar solo por el dueño del clon PG18 desechable."
