#!/usr/bin/env bash
# Sólo laboratorio Docker aislado y cedido en exclusiva por Dirección.
# No acepta DSN, publica puertos ni ejecuta DOWN. UP nuevo + pruebas + ROLLBACK.
set -euo pipefail
if [[ $# -ne 1 || ! $1 =~ ^vec-codexb-(kit-b2|org-auditoria-oh)-20261002$ ]]; then
    printf '%s\n' 'Uso: guion <contenedor propio de ensayo CT162 autorizado>' >&2
    exit 64
fi
contenedor_ct162=$1
raiz_ct162=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../../.." && pwd)
read -r vivo_ct162 red_ct162 pids_ct162 memoria_ct162 cpu_ct162 < <(
 docker inspect --format '{{.State.Running}} {{.HostConfig.NetworkMode}} {{.HostConfig.PidsLimit}} {{.HostConfig.Memory}} {{.HostConfig.NanoCpus}}' "$contenedor_ct162")
if [[ $vivo_ct162 != true || $pids_ct162 -lt 1 || $pids_ct162 -gt 256 ||
      $memoria_ct162 -lt 1 || $memoria_ct162 -gt 2147483648 ||
      $cpu_ct162 -lt 1 || $cpu_ct162 -gt 2000000000 ]]; then
    printf '%s\n' 'PARO: contenedor sin límites de ensayo.' >&2
    exit 78
fi
if [[ $red_ct162 != none && $(docker network inspect --format '{{.Internal}}' "$red_ct162") != true ]]; then
    printf '%s\n' 'PARO: red de ensayo no aislada.' >&2
    exit 78
fi
python3 - "$raiz_ct162" <<'PY' | docker exec -i "$contenedor_ct162" psql -X -v ON_ERROR_STOP=1 -U postgres -d postgres
from pathlib import Path
import sys
r=Path(sys.argv[1])
up=(r/'deploy/postgresql/contratacion_temporal/migraciones/000162_auditoria_frontera_organizacion_historica.up.sql').read_text()
test=(r/'deploy/postgresql/contratacion_temporal/pruebas_sql/auditoria_frontera_organizacion_historica_000162.sql').read_text()
assert up.count('\nBEGIN;\n')==1 and up.endswith('COMMIT;\n')
up=up.replace('\nBEGIN;\n','\nSAVEPOINT ct162_nuevo_up;\n',1)
up=up[:-len('COMMIT;\n')]+'RELEASE SAVEPOINT ct162_nuevo_up;\n'
snapshot="""jsonb_build_object(
 'funcion',(SELECT to_jsonb(p) FROM pg_proc p WHERE p.oid='vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text)'::regprocedure),
 'tabla',(SELECT to_jsonb(c) FROM pg_class c WHERE c.oid='vec_contratacion_temporal.auditoria_frontera_ruta_exacta'::regclass),
 'constraints',(SELECT jsonb_agg(to_jsonb(c) ORDER BY c.conname) FROM pg_constraint c WHERE c.conrelid='vec_contratacion_temporal.auditoria_frontera_ruta_exacta'::regclass),
 'triggers',(SELECT jsonb_agg(to_jsonb(t) ORDER BY t.tgname) FROM pg_trigger t WHERE t.tgrelid='vec_contratacion_temporal.auditoria_frontera_ruta_exacta'::regclass),
 'policies',(SELECT jsonb_agg(to_jsonb(p) ORDER BY p.polname) FROM pg_policy p WHERE p.polrelid='vec_contratacion_temporal.auditoria_frontera_ruta_exacta'::regclass),
 'indices',(SELECT jsonb_agg(to_jsonb(i) ORDER BY i.indexrelid) FROM pg_index i WHERE i.indrelid='vec_contratacion_temporal.auditoria_frontera_ruta_exacta'::regclass),
 'secuencia',(SELECT jsonb_build_object('last_value',s.last_value,'log_cnt',s.log_cnt,'is_called',s.is_called) FROM vec_contratacion_temporal.auditoria_frontera_ruta_exacta_evento_id_seq s),
 'historia',(SELECT encode(sha256(convert_to(coalesce(jsonb_agg(to_jsonb(t) ORDER BY t.evento_id)::text,'[]'),'UTF8')),'hex') FROM vec_contratacion_temporal.auditoria_frontera_ruta_exacta t))"""
print('CREATE TEMP TABLE ct162_restauracion AS SELECT '+snapshot+' AS estado;')
print('''\nBEGIN;
SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='30s';
CREATE TEMP TABLE ct162_preimagen ON COMMIT DROP AS
SELECT encode(sha256(convert_to(coalesce(jsonb_agg(to_jsonb(t) ORDER BY t.evento_id)::text,'[]'),'UTF8')),'hex') AS huella_filas
FROM vec_contratacion_temporal.auditoria_frontera_ruta_exacta t;
''')
print(up)
print(test)
print('ROLLBACK;')
print('DO $restauracion$ BEGIN IF (SELECT estado FROM ct162_restauracion) IS DISTINCT FROM '+snapshot+" THEN RAISE EXCEPTION 'CT162: postROLLBACK no equivale a preimagen'; END IF; END $restauracion$;")
print("SELECT 'CT162-ROLLBACK-PREIMAGEN-FUNCION-ACL-FILAS-SECUENCIA-IDENTICA' AS resultado;")
PY
