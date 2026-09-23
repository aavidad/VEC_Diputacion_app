#!/usr/bin/env bash
set -euo pipefail
export LC_ALL=C TZ=UTC

# Solo un contenedor sin red. La preimagen privada se proporciona fuera de Git.
: "${VEC_PREIMAGEN_DIR:?preimagen schema-only privada requerida}"
: "${VEC_AD3_51_UP:?ruta de AD3-51 revisada requerida}"
: "${VEC_AD3_51_SHA256:?huella de AD3-51 revisada requerida}"
: "${VEC_IDENTIDAD2_UP:?ruta de Identidad2 revisada requerida}"
: "${VEC_IDENTIDAD2_SHA256:?huella de Identidad2 revisada requerida}"
[[ "${VEC_IDENTIDAD2_UP##*/}" == '000002_revalidacion_consulta_rrhh_ambito_v1.up.sql' \
   && "$VEC_IDENTIDAD2_SHA256" == '96840ddfc06c0a2a24ace75cdadc3bbb9b09450910eb8c3e1ca0f502b9bd5f66' ]] || {
  echo 'Identidad2: ruta o huella ajena a revalidación RRHH de ámbito' >&2
  exit 1
}

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)
# El lanzador `go` local puede seleccionar una versión demasiado antigua con
# GOTOOLCHAIN=local. Elegir el binario concreto antes de abrir PostgreSQL.
go_pruebas=${VEC_GO_TEST_BIN:-$(go env GOROOT)/bin/go}
[[ -x "$go_pruebas" ]] || { echo 'binario Go de prueba ausente' >&2; exit 1; }
version_go=$(GOTOOLCHAIN=local "$go_pruebas" version)
[[ "$version_go" =~ go1\.26\.([5-9]|[1-9][0-9]+)([[:space:]]|$) ]] || {
  echo 'binario Go de prueba incompatible' >&2
  exit 1
}
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-ca6-ct109-${USER:-agente}-$$"
temporal=$(mktemp -d)
chmod 700 "$temporal"
limpiar() {
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
  rm -rf -- "$temporal"
}
trap limpiar EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

comprobar_sha() {
  local archivo=$1 esperado=$2 observado
  observado=$(sha256sum "$archivo" | cut -d' ' -f1)
  [[ "$observado" == "$esperado" ]] || { echo "preimagen/AD3 incompatible: ${archivo##*/}" >&2; exit 1; }
}
comprobar_sha "$VEC_PREIMAGEN_DIR/cidonia_schema_10_esquemas.sql" \
  9df366c79f29322b833fbe271212eed739a7a0ef9cfad789b6602f45c28d528f
comprobar_sha "$VEC_PREIMAGEN_DIR/roles_vec.txt" \
  18a0ca25831b83d0ef82cb80f65c7a33dfc1e35bd0fa52ee9dca23af2014aa1c
comprobar_sha "$VEC_PREIMAGEN_DIR/roles_acl_saneadas.tsv" \
  43e43ea31fc92097db955089b144d52f1d872e7c2fc3cc22a6ede0519fa58809
comprobar_sha "$VEC_AD3_51_UP" "$VEC_AD3_51_SHA256"
comprobar_sha "$VEC_IDENTIDAD2_UP" "$VEC_IDENTIDAD2_SHA256"
rg -Fq 'vec_identidad_sesiones_v1.revalidar_consulta_rrhh_v1(text,text)' \
  "$VEC_IDENTIDAD2_UP" || {
  echo 'Identidad2: función de revalidación RRHH de ámbito ausente' >&2
  exit 1
}
python3 - "$VEC_PREIMAGEN_DIR/cidonia_schema_10_esquemas.sql" \
  "$raiz/deploy/postgresql/contexto_actor_v1/pruebas_sql/ct109_positivo_genesis.sql" \
  "$temporal/schema_con_genesis.sql" <<'PY'
from pathlib import Path
import hashlib, sys
origen, genesis, destino = map(Path, sys.argv[1:])
base, filas = origen.read_bytes(), genesis.read_bytes()
if hashlib.sha256(base).hexdigest() != '9df366c79f29322b833fbe271212eed739a7a0ef9cfad789b6602f45c28d528f':
    raise SystemExit('preimagen estructural distinta')
ancla = (b'CREATE TRIGGER bloquear_insercion_checkpoint BEFORE INSERT ON '
         b'vec_autorizacion.motivo_v2_checkpoint_origen FOR EACH ROW EXECUTE '
         b'FUNCTION vec_autorizacion.motivo_v2_bloquear_mutacion_inmutable();')
if base.count(ancla) != 1 or b'\\restrict ' not in base or b'\\unrestrict ' not in base:
    raise SystemExit('punto de genesis no univoco')
if any(line.lstrip().startswith(b'\\') for line in filas.splitlines()):
    raise SystemExit('genesis contiene metacomandos')
pos = base.index(ancla)
if base[:pos].count(b'\nBEGIN;\n') or base[:pos].count(b'\nCOMMIT;\n'):
    raise SystemExit('dump inesperadamente transaccional antes de genesis')
ensayo = base[:pos] + filas + b'\n' + base[pos:]
if ensayo[:pos] + ensayo[pos+len(filas)+1:] != base:
    raise SystemExit('preimagen modificada fuera de genesis')
destino.write_bytes(ensayo)
PY
chmod 600 "$temporal/schema_con_genesis.sql"

python3 - "$VEC_PREIMAGEN_DIR/roles_acl_saneadas.tsv" "$temporal/roles.sql" <<'PY'
import re,sys
src,dst=sys.argv[1:]
lines=['BEGIN;','REVOKE ALL ON DATABASE postgres FROM PUBLIC;']
roles=[];members=[];acls=[]
for raw in open(src,encoding='utf8'):
    fields=raw.strip().split('|')
    if fields[0]=='ROLE': roles.append(fields)
    elif fields[0]=='MEMBER': members.append(fields)
    elif fields[0]=='DBACL': acls.append(fields)
    else: raise SystemExit('formato de ACL inesperado')
def ident(s):
    if not re.fullmatch(r'[a-z_][a-z0-9_]*',s): raise SystemExit('identificador no admitido')
    return '"'+s+'"'
def flag(v,a,b):
    if v not in ('t','f'): raise SystemExit('booleano no admitido')
    return a if v=='t' else b
for f in roles:
    if len(f)!=5 or not re.fullmatch(r'-1|[0-9]{1,3}',f[4]): raise SystemExit('rol incompatible')
    lines.append('CREATE ROLE '+ident(f[1])+' '+flag(f[2],'LOGIN','NOLOGIN')+' '+flag(f[3],'INHERIT','NOINHERIT')+' NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS CONNECTION LIMIT '+f[4]+';')
for f in members:
    if len(f)!=6: raise SystemExit('membresia incompatible')
    lines.append('GRANT '+ident(f[2])+' TO '+ident(f[1])+' WITH ADMIN '+flag(f[3],'TRUE','FALSE')+', INHERIT '+flag(f[4],'TRUE','FALSE')+', SET '+flag(f[5],'TRUE','FALSE')+';')
for f in acls:
    if len(f)!=5 or f[1]!='postgres' or f[3] not in ('CONNECT','CREATE','TEMPORARY'):
        raise SystemExit('ACL de base incompatible')
    lines.append('GRANT '+f[3]+' ON DATABASE postgres TO '+ident(f[2])+(' WITH GRANT OPTION' if f[4]=='t' else '')+';')
lines.append('COMMIT;')
open(dst,'w',encoding='utf8').write('\n'.join(lines)+'\n')
PY
chmod 600 "$temporal/roles.sql"

docker run -d --rm --network none --name "$contenedor" \
  -e POSTGRES_HOST_AUTH_METHOD=trust "$imagen" >/dev/null
for _ in $(seq 1 200); do
  if docker logs "$contenedor" 2>&1 | rg -q 'PostgreSQL init process complete'; then break; fi
  sleep 0.2
done
docker logs "$contenedor" 2>&1 | rg -q 'PostgreSQL init process complete'
consecutivas=0
for _ in $(seq 1 100); do
  if version=$(docker exec "$contenedor" psql -XAtq -v ON_ERROR_STOP=1 \
    -U postgres -d postgres -c "SELECT current_setting('server_version_num')||'|'||pg_is_in_recovery()" \
    2>/dev/null) && [[ "$version" == '180004|false' ]]; then
    consecutivas=$((consecutivas+1))
    [[ "$consecutivas" -eq 3 ]] && break
  else
    consecutivas=0
  fi
  sleep 0.25
done
[[ "$consecutivas" -eq 3 ]] || { echo 'PostgreSQL 18.4 no arrancó estable' >&2; exit 1; }
psql_archivo() {
  docker exec -i "$contenedor" psql -Xq -v ON_ERROR_STOP=1 -U postgres -d postgres < "$1"
}
psql_sql() {
  docker exec -i "$contenedor" psql -XqAt -v ON_ERROR_STOP=1 -U postgres -d postgres
}
psql_archivo "$temporal/roles.sql"
psql_sql <<'SQL'
CREATE EXTENSION IF NOT EXISTS pgcrypto;
SQL
psql_archivo "$temporal/schema_con_genesis.sql"
# La inserción histórica de génesis debe dejar idénticos esquema, ACL y
# definiciones/estado de triggers al restaurar el mismo dump sin datos.
docker exec "$contenedor" createdb -U postgres -T template0 ct109_schema_referencia
docker exec "$contenedor" psql -Xq -v ON_ERROR_STOP=1 \
  -U postgres -d ct109_schema_referencia -c 'CREATE EXTENSION pgcrypto;' >/dev/null
docker exec -i "$contenedor" psql -Xq -v ON_ERROR_STOP=1 \
  -U postgres -d ct109_schema_referencia \
  < "$VEC_PREIMAGEN_DIR/cidonia_schema_10_esquemas.sql"
docker exec "$contenedor" pg_dump -U postgres -d postgres --schema-only \
  > "$temporal/schema_con_genesis.dump.sql"
docker exec "$contenedor" pg_dump -U postgres -d ct109_schema_referencia --schema-only \
  > "$temporal/schema_referencia.dump.sql"
python3 - "$temporal/schema_con_genesis.dump.sql" \
  "$temporal/schema_referencia.dump.sql" <<'PY'
from pathlib import Path
import sys
def normalizar(ruta):
    lineas=[]
    for linea in Path(ruta).read_bytes().splitlines(keepends=True):
        if linea.startswith(b'\\restrict '): linea=b'\\restrict ID\n'
        elif linea.startswith(b'\\unrestrict '): linea=b'\\unrestrict ID\n'
        lineas.append(linea)
    return b''.join(lineas)
if normalizar(sys.argv[1]) != normalizar(sys.argv[2]):
    raise SystemExit('genesis altero esquema, ACL o triggers frente a restauracion integra')
PY
psql_sql <<'SQL'
DO $guardas_restauradas$
BEGIN
 IF (SELECT count(*) FROM pg_trigger
     WHERE (tgrelid='vec_autorizacion.motivo_v2_checkpoint_origen'::regclass
            AND tgname='bloquear_insercion_checkpoint'
            OR tgrelid='vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass
            AND tgname='vinculacion_motivo_rrhh_checkpoint_inmutable')
       AND NOT tgisinternal AND tgenabled='O')<>2
 THEN RAISE EXCEPTION 'guardas de genesis no restauradas activas'; END IF;
END $guardas_restauradas$;
SQL

# Schema-only no contiene la fila de generacion: verificar antes de sembrarla.
psql_sql <<'SQL'
DO $preimagen$
BEGIN
 IF EXISTS (SELECT 1 FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2)
 THEN RAISE EXCEPTION 'fixture CA2 ya presente'; END IF;
END $preimagen$;
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
 (control_id,generacion,actualizada_en) VALUES (true,0,clock_timestamp());
COMMIT;
SQL
psql_sql <<'SQL'
DO $ausencia$
BEGIN
 IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.control_cadena_accesos_rrhh)
   OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.checkpoint_gobierno)
   OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.control_cadena_auditoria)
 THEN RAISE EXCEPTION 'singleton CT/V3 ya presente en schema-only'; END IF;
END $ausencia$;
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
INSERT INTO vec_contratacion_temporal.control_cadena_accesos_rrhh
 (control,ultima_secuencia,cabeza_sha256,actualizada_en)
 VALUES (true,0,repeat('0',64),date_trunc('microseconds',clock_timestamp()));
COMMIT;
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
INSERT INTO vec_autorizacion_atestada_v3.checkpoint_gobierno
 (control_id,revision,configuracion_secuencia_minima,raiz_version_minima,actualizada_en)
 VALUES (true,0,0,0,clock_timestamp());
INSERT INTO vec_autorizacion_atestada_v3.control_cadena_auditoria
 (control_id,secuencia,cabeza_sha256,actualizada_en)
 VALUES (true,0,repeat('0',64),clock_timestamp());
COMMIT;
SQL
psql_sql <<'SQL'
-- Fixture de rol únicamente efímero. El instalador oficial del selector
-- requiere una preimagen de membresías más estrecha que este dump de
-- desarrollo; no se relajan ni eliminan las membresías conservadas.
DO $ausencia_selector$
BEGIN
 IF to_regrole('vec_contexto_actor_corporativo_rrhh_selector') IS NOT NULL
 THEN RAISE EXCEPTION 'selector ya presente en preimagen'; END IF;
END $ausencia_selector$;
CREATE ROLE vec_contexto_actor_corporativo_rrhh_selector
 NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE
 NOREPLICATION NOBYPASSRLS CONNECTION LIMIT -1 PASSWORD NULL;
COMMENT ON ROLE vec_contexto_actor_corporativo_rrhh_selector IS
 'vec_contexto_actor_v1:rol-contexto-corporativo-rrhh-selector:v1';
GRANT CONNECT ON DATABASE postgres TO vec_contexto_actor_corporativo_rrhh_selector;
SQL
psql_sql <<'SQL'
-- CA3/CA4 son anteriores a la membresía de gobierno AD3 del dump. En esta
-- efímera se reproduce su preimagen histórica y se restaura después.
DO $membresia_historica$
BEGIN
 IF (SELECT count(*) FROM pg_auth_members m
     WHERE m.roleid='vec_contexto_actor_v1_propietario'::regrole
       AND m.member='vec_ad3_o207_gobierno'::regrole
       AND NOT m.admin_option AND m.inherit_option AND m.set_option)<>1
 THEN RAISE EXCEPTION 'membresía CA posterior no acreditada'; END IF;
END $membresia_historica$;
REVOKE vec_contexto_actor_v1_propietario FROM vec_ad3_o207_gobierno;
SQL
psql_archivo "$raiz/deploy/postgresql/contexto_actor_v1/migraciones/000003_organizacion_corporativa_v1.up.sql"
# El dump es posterior a CA5, pero anterior a CA3/CA4. La guarda histórica
# de CA4 debe rechazarlo; para probar CA6 se materializa solo la estructura
# desde el segmento DDL y postcondiciones de CA4, con huella fija. Esto NO
# acredita la instalación histórica CA4 ni equivale a aplicarla en cidonia.
python3 - "$raiz/deploy/postgresql/contexto_actor_v1/migraciones/000004_vinculo_corporativo_rrhh_v1.up.sql" "$temporal/ca4_fixture.sql" <<'PY'
from pathlib import Path
import hashlib,sys
fuente=Path(sys.argv[1]); salida=Path(sys.argv[2]); s=fuente.read_text()
if hashlib.sha256(fuente.read_bytes()).hexdigest()!='f1be3123b1286e7fe8ffae99073179b90b08303c31c8ee8c3783315a6078f368':
    raise SystemExit('CA4: fuente historica alterada')
a=s.index('SET LOCAL ROLE vec_contexto_actor_v1_propietario;')
b=s.rindex('\nCOMMIT;')
segmento=s[a:b]
if hashlib.sha256(segmento.encode()).hexdigest()!='70fe8690fcd9858f342aef43355cf9521cc7d3dfbd0279359a05777a09c5c9d3':
    raise SystemExit('CA4: segmento estructural alterado')
salida.write_text('BEGIN;\nSET LOCAL search_path=pg_catalog;\nSET LOCAL timezone=\'UTC\';\n'+segmento+'\nCOMMIT;\n')
PY
chmod 600 "$temporal/ca4_fixture.sql"
psql_archivo "$temporal/ca4_fixture.sql"
psql_sql <<'SQL'
GRANT vec_contexto_actor_v1_propietario TO vec_ad3_o207_gobierno
 WITH ADMIN FALSE, INHERIT TRUE, SET TRUE;
DO $membresia_restituida$
BEGIN
 IF (SELECT count(*) FROM pg_auth_members m
     WHERE m.roleid='vec_contexto_actor_v1_propietario'::regrole
       AND m.member='vec_ad3_o207_gobierno'::regrole
       AND NOT m.admin_option AND m.inherit_option AND m.set_option)<>1
 THEN RAISE EXCEPTION 'membresía CA posterior no restituida'; END IF;
END $membresia_restituida$;
SQL
psql_archivo "$raiz/deploy/postgresql/contratacion_temporal/roles_consultor_rrhh_ambito_up.sql"
psql_archivo "$VEC_AD3_51_UP"
psql_archivo "$VEC_IDENTIDAD2_UP"
psql_archivo "$raiz/deploy/postgresql/contexto_actor_v1/migraciones/000006_acreditacion_ambito_rrhh_v1.up.sql"
psql_archivo "$raiz/deploy/postgresql/contratacion_temporal/migraciones/000109_consultas_rrhh_ambito_v1.up.sql"
psql_archivo "$raiz/deploy/postgresql/contexto_actor_v1/pruebas_sql/ensayar_ca6_ct109_pg18.sql"
psql_sql <<'SQL'
CREATE ROLE ct_ambito_pg_prueba LOGIN INHERIT NOSUPERUSER NOCREATEDB
 NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_contratacion_temporal_consultor_rrhh_ambito
 TO ct_ambito_pg_prueba WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SQL
acl_login=$(docker exec -i "$contenedor" psql -XqAt -v ON_ERROR_STOP=1 \
 -U ct_ambito_pg_prueba -d postgres <<'SQL'
SELECT has_function_privilege(current_user,
 'vec_contratacion_temporal.consultar_cuadro_rrhh_ambito_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,'EXECUTE'),
 has_function_privilege(current_user,
 'vec_contratacion_temporal.consultar_detalle_rrhh_ambito_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,'EXECUTE'),
 has_function_privilege(current_user,
 'vec_contratacion_temporal.consultar_estadisticas_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,text,date,date)'::regprocedure,'EXECUTE'),
 has_table_privilege(current_user,'vec_contratacion_temporal.registro_acceso_rrhh','SELECT');
SQL
)
[[ "$acl_login" == 't|t|f|f' ]] || { echo "ACL de login nominal CT incompatible: $acl_login" >&2; exit 1; }
if docker exec -i "$contenedor" psql -Xq -v ON_ERROR_STOP=1 \
 -U ct_ambito_pg_prueba -d postgres -c 'SET ROLE vec_contratacion_temporal_consultor_rrhh' \
 > "$temporal/impersonacion.log" 2>&1; then
 echo 'login nominal pudo asumir el rol legado' >&2
 exit 1
fi
psql_sql <<'SQL'
GRANT SELECT ON public.ca6_comprobante_prueba TO ct_ambito_pg_prueba;
SQL
if docker exec -i "$contenedor" psql -Xq -v ON_ERROR_STOP=1 \
 -v VERBOSITY=verbose -U ct_ambito_pg_prueba -d postgres \
 > "$temporal/ct109_material_invalido.log" 2>&1 <<'SQL'
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SELECT f.* FROM public.ca6_comprobante_prueba p
 CROSS JOIN LATERAL vec_contratacion_temporal.consultar_cuadro_rrhh_ambito_v1(
 ROW('org_diputaciondemo0001','organizacion','org_diputaciondemo0001')::
   vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 ROW('','','',10,'')::vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
 p.comprobante,
 convert_to(repeat('x',512),'UTF8'),convert_to('x','UTF8'),
 convert_to('x','UTF8'),
 convert_to(jsonb_build_object(
   'cuenta_ref',p.comprobante->>'cuenta_ref',
   'cuenta_version',(p.comprobante->>'cuenta_version')::numeric,
   'persona_ref',p.comprobante->>'persona_ref',
   'persona_version',(p.comprobante->>'persona_version')::numeric,
   'perfil_activo_ref',p.comprobante->>'perfil_ref',
   'perfil_version',(p.comprobante->>'perfil_version')::numeric,
   'contexto_actor_ref',p.comprobante->>'contexto_ref',
   'contexto_version',(p.comprobante->>'contexto_version')::numeric)::text,'UTF8'),
 1,1,convert_to('x','UTF8'),convert_to('x','UTF8'),
 convert_to('x','UTF8'),convert_to('x','UTF8')) f;
COMMIT;
SQL
then
 echo 'CT109 aceptó material V3 inválido' >&2
 exit 1
fi
rg -q '42501' "$temporal/ct109_material_invalido.log" || {
 echo 'CT109 no rechazó con SQLSTATE 42501' >&2
 tail -n 20 "$temporal/ct109_material_invalido.log" >&2
 exit 1
}
rg -q 'consulta RRHH rechazada|motor de consultas RRHH rechazado' \
 "$temporal/ct109_material_invalido.log" || {
 echo 'CT109 no alcanzó el motor tras CA6 positivo' >&2
 tail -n 20 "$temporal/ct109_material_invalido.log" >&2
 exit 1
}
psql_sql <<'SQL'
DO $rollback$
BEGIN
 IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.registro_acceso_rrhh)
   OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3)
 THEN RAISE EXCEPTION 'CT109: rechazo dejó acceso o consumo'; END IF;
END $rollback$;
SQL

cat > "$temporal/snapshot.sql" <<'SQL'
SET application_name='ca6_snapshot_obsoleto';
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SELECT count(*) FROM vec_contexto_actor_v1.vinculo_corporativo_actual;
SELECT pg_sleep(3);
SELECT vec_contexto_actor_v1.acreditar_ambito_rrhh_v1(
 p.comprobante,p.comprobante->>'cuenta_ref',p.comprobante->>'persona_ref',
 p.comprobante->>'perfil_ref',p.comprobante->>'contexto_ref',
 p.comprobante->>'organizacion_ref')
 FROM public.ca6_comprobante_prueba p;
COMMIT;
SQL
docker exec -i "$contenedor" psql -Xq -v ON_ERROR_STOP=1 -v VERBOSITY=verbose \
  -U postgres -d postgres < "$temporal/snapshot.sql" > "$temporal/snapshot.log" 2>&1 &
snapshot_pid=$!
for _ in $(seq 1 100); do
  estado=$(psql_sql <<'SQL'
SELECT count(*) FROM pg_stat_activity
 WHERE application_name='ca6_snapshot_obsoleto' AND query LIKE '%pg_sleep%';
SQL
)
  [[ "$estado" == 1 ]] && break
  sleep 0.02
done
[[ "${estado:-0}" == 1 ]] || { echo 'snapshot anterior no arrancó' >&2; exit 1; }

# La revocación confirma mientras el lector conserva un snapshot antiguo.
psql_sql <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones
 SELECT (jsonb_populate_record(NULL::vec_contexto_actor_v1.vinculo_corporativo_versiones,
  to_jsonb(v)||jsonb_build_object('version',2,'estado','revocado'))).*
 FROM vec_contexto_actor_v1.vinculo_corporativo_versiones v
 WHERE v.vinculo_corporativo_ref='vcr_corporativo_rrhh_000000000001' AND v.version=1;
UPDATE vec_contexto_actor_v1.vinculo_corporativo_actual SET version=2
 WHERE cuenta_ref='cta_corporativa_rrhh_000000000001';
COMMIT;
SQL
set +e
wait "$snapshot_pid"
snapshot_estado=$?
set -e
if [[ "$snapshot_estado" -eq 0 ]] || ! rg -q '40001' "$temporal/snapshot.log"; then
  echo 'snapshot obsoleto no terminó en 40001' >&2
  tail -n 15 "$temporal/snapshot.log" >&2
  exit 1
fi

# La misma etiqueta de cuenta/organización con nueva versión no revive el
# comprobante anterior; una concesión nueva obtiene comprobante distinto.
psql_sql <<'SQL'
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO $revocada$
DECLARE p jsonb;
BEGIN
 SELECT comprobante INTO STRICT p FROM public.ca6_comprobante_prueba;
 IF vec_contexto_actor_v1.acreditar_ambito_rrhh_v1(
  p,p->>'cuenta_ref',p->>'persona_ref',p->>'perfil_ref',
  p->>'contexto_ref',p->>'organizacion_ref') IS NOT NULL
 THEN RAISE EXCEPTION 'CA6: revocación confirmada ignorada'; END IF;
END $revocada$;
COMMIT;
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones
 SELECT (jsonb_populate_record(NULL::vec_contexto_actor_v1.vinculo_corporativo_versiones,
  to_jsonb(v)||jsonb_build_object('version',3,'estado','activo'))).*
 FROM vec_contexto_actor_v1.vinculo_corporativo_versiones v
 WHERE v.vinculo_corporativo_ref='vcr_corporativo_rrhh_000000000001' AND v.version=1;
UPDATE vec_contexto_actor_v1.vinculo_corporativo_actual SET version=3
 WHERE cuenta_ref='cta_corporativa_rrhh_000000000001';
COMMIT;
CREATE TABLE public.ca6_comprobante_nuevo AS
 SELECT vec_contexto_actor_v1.resolver_comprobante_ambito_rrhh_v1(
  'cta_corporativa_rrhh_000000000001',1,
  'per_corporativa_rrhh_000000000001',1,
  'prf_corporativo_rrhh_000000000001',1,
  'vca_corporativo_rrhh_000000000001',1) AS comprobante;
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO $nueva$
DECLARE vieja jsonb; nueva jsonb;
BEGIN
 SELECT comprobante INTO STRICT vieja FROM public.ca6_comprobante_prueba;
 SELECT comprobante INTO STRICT nueva FROM public.ca6_comprobante_nuevo;
 IF nueva IS NULL OR nueva=vieja
   OR nueva->>'vinculo_corporativo_version'<>'3'
   OR vec_contexto_actor_v1.acreditar_ambito_rrhh_v1(
    vieja,vieja->>'cuenta_ref',vieja->>'persona_ref',vieja->>'perfil_ref',
    vieja->>'contexto_ref',vieja->>'organizacion_ref') IS NOT NULL
   OR vec_contexto_actor_v1.acreditar_ambito_rrhh_v1(
    nueva,nueva->>'cuenta_ref',nueva->>'persona_ref',nueva->>'perfil_ref',
    nueva->>'contexto_ref',nueva->>'organizacion_ref') IS NULL
 THEN RAISE EXCEPTION 'CA6: nueva concesión reutilizó autoridad vieja'; END IF;
END $nueva$;
COMMIT;
SQL

cat > "$temporal/lector_bloquea.sql" <<'SQL'
SET application_name='ca6_lector_bloquea';
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SELECT vec_contexto_actor_v1.acreditar_ambito_rrhh_v1(
 p.comprobante,p.comprobante->>'cuenta_ref',p.comprobante->>'persona_ref',
 p.comprobante->>'perfil_ref',p.comprobante->>'contexto_ref',
 p.comprobante->>'organizacion_ref') IS NOT NULL
 FROM public.ca6_comprobante_nuevo p;
SELECT pg_sleep(3);
COMMIT;
SQL
cat > "$temporal/revocar_bloqueada.sql" <<'SQL'
SET application_name='ca6_revocador_bloqueado';
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones
 SELECT (jsonb_populate_record(NULL::vec_contexto_actor_v1.vinculo_corporativo_versiones,
  to_jsonb(v)||jsonb_build_object('version',4,'estado','revocado'))).*
 FROM vec_contexto_actor_v1.vinculo_corporativo_versiones v
 WHERE v.vinculo_corporativo_ref='vcr_corporativo_rrhh_000000000001' AND v.version=1;
UPDATE vec_contexto_actor_v1.vinculo_corporativo_actual SET version=4
 WHERE cuenta_ref='cta_corporativa_rrhh_000000000001';
COMMIT;
SQL
docker exec -i "$contenedor" psql -XqAt -v ON_ERROR_STOP=1 \
 -U postgres -d postgres < "$temporal/lector_bloquea.sql" > "$temporal/lector_bloquea.log" 2>&1 &
lector_pid=$!
for _ in $(seq 1 100); do
  estado=$(psql_sql <<'SQL'
SELECT count(*) FROM pg_stat_activity
 WHERE application_name='ca6_lector_bloquea' AND query LIKE '%pg_sleep%';
SQL
)
  [[ "$estado" == 1 ]] && break
  sleep 0.02
done
[[ "${estado:-0}" == 1 ]] || { echo 'lector con lock no arrancó' >&2; exit 1; }
docker exec -i "$contenedor" psql -XqAt -v ON_ERROR_STOP=1 \
 -U postgres -d postgres < "$temporal/revocar_bloqueada.sql" > "$temporal/revocar_bloqueada.log" 2>&1 &
revocador_pid=$!
for _ in $(seq 1 100); do
  bloqueada=$(psql_sql <<'SQL'
SELECT count(*) FROM pg_stat_activity
 WHERE application_name='ca6_revocador_bloqueado' AND wait_event_type='Lock';
SQL
)
  [[ "$bloqueada" == 1 ]] && break
  sleep 0.02
done
[[ "${bloqueada:-0}" == 1 ]] || { echo 'revocación no esperó al lector' >&2; exit 1; }
wait "$lector_pid"
wait "$revocador_pid"
rg -q '^t$' "$temporal/lector_bloquea.log" || { echo 'lector no acreditó uso' >&2; exit 1; }

# El comprobante vence mientras espera el lock global del mutador. Su
# comprobación previa prueba que aún era válido al iniciar el consumo.
psql_sql <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones
 SELECT (jsonb_populate_record(NULL::vec_contexto_actor_v1.vinculo_corporativo_versiones,
  to_jsonb(v)||jsonb_build_object('version',5,'estado','activo',
   'vigente_desde',clock_timestamp()-interval '1 hour',
   'vigente_hasta',clock_timestamp()+interval '4 seconds'))).*
 FROM vec_contexto_actor_v1.vinculo_corporativo_versiones v
 WHERE v.vinculo_corporativo_ref='vcr_corporativo_rrhh_000000000001' AND v.version=1;
UPDATE vec_contexto_actor_v1.vinculo_corporativo_actual SET version=5
 WHERE cuenta_ref='cta_corporativa_rrhh_000000000001';
COMMIT;
CREATE TABLE public.ca6_comprobante_corto AS
 SELECT vec_contexto_actor_v1.resolver_comprobante_ambito_rrhh_v1(
  'cta_corporativa_rrhh_000000000001',1,
  'per_corporativa_rrhh_000000000001',1,
  'prf_corporativo_rrhh_000000000001',1,
  'vca_corporativo_rrhh_000000000001',1) AS comprobante;
SQL
cat > "$temporal/bloquear_caducidad.sql" <<'SQL'
SET application_name='ca6_bloqueador_caducidad';
BEGIN;
SELECT pg_advisory_xact_lock(hashtextextended(
 'vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));
SELECT pg_sleep(6);
COMMIT;
SQL
cat > "$temporal/caducidad.sql" <<'SQL'
SET application_name='ca6_caducidad';
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SELECT clock_timestamp()<(p.comprobante->>'vigente_hasta')::timestamptz
 FROM public.ca6_comprobante_corto p;
SELECT vec_contexto_actor_v1.acreditar_ambito_rrhh_v1(
 p.comprobante,p.comprobante->>'cuenta_ref',p.comprobante->>'persona_ref',
 p.comprobante->>'perfil_ref',p.comprobante->>'contexto_ref',
 p.comprobante->>'organizacion_ref') IS NULL
 FROM public.ca6_comprobante_corto p;
COMMIT;
SQL
docker exec -i "$contenedor" psql -XqAt -v ON_ERROR_STOP=1 \
 -U postgres -d postgres < "$temporal/bloquear_caducidad.sql" > "$temporal/bloquear_caducidad.log" 2>&1 &
bloqueador_pid=$!
for _ in $(seq 1 100); do
  estado=$(psql_sql <<'SQL'
SELECT count(*) FROM pg_stat_activity
 WHERE application_name='ca6_bloqueador_caducidad' AND query LIKE '%pg_sleep%';
SQL
)
  [[ "$estado" == 1 ]] && break
  sleep 0.02
done
[[ "${estado:-0}" == 1 ]] || { echo 'bloqueador de caducidad no arrancó' >&2; exit 1; }
docker exec -i "$contenedor" psql -XqAt -v ON_ERROR_STOP=1 \
 -U postgres -d postgres < "$temporal/caducidad.sql" > "$temporal/caducidad.log" 2>&1 &
caducidad_pid=$!
for _ in $(seq 1 100); do
  bloqueada=$(psql_sql <<'SQL'
SELECT count(*) FROM pg_stat_activity
 WHERE application_name='ca6_caducidad' AND wait_event_type='Lock';
SQL
)
  [[ "$bloqueada" == 1 ]] && break
  sleep 0.02
done
[[ "${bloqueada:-0}" == 1 ]] || { echo 'acreditación no esperó al lock' >&2; exit 1; }
wait "$bloqueador_pid"
wait "$caducidad_pid"
[[ $(rg -c '^t$' "$temporal/caducidad.log") == 2 ]] || {
 echo 'caducidad durante espera no denegó con control previo positivo' >&2
 tail -n 15 "$temporal/caducidad.log" >&2
 exit 1
}

# DOWN solo en esta base desechable. Un login activo y cualquier avance de
# historia o cadena deben bloquearlo antes de modificar funciones/ACL.
down_ct="$raiz/deploy/postgresql/contratacion_temporal/migraciones/000109_consultas_rrhh_ambito_v1.down.sql"
down_ca="$raiz/deploy/postgresql/contexto_actor_v1/migraciones/000006_acreditacion_ambito_rrhh_v1.down.sql"
if psql_archivo "$down_ct" > "$temporal/down_login.log" 2>&1; then
 echo 'CT109 DOWN aceptó login productivo activo' >&2
 exit 1
fi
rg -q 'identidad activa' "$temporal/down_login.log" || {
 echo 'CT109 DOWN falló por causa ajena al login' >&2
 tail -n 15 "$temporal/down_login.log" >&2
 exit 1
}
psql_sql <<'SQL'
REVOKE SELECT ON public.ca6_comprobante_prueba FROM ct_ambito_pg_prueba;
REVOKE vec_contratacion_temporal_consultor_rrhh_ambito FROM ct_ambito_pg_prueba;
DROP ROLE ct_ambito_pg_prueba;
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
UPDATE vec_contratacion_temporal.control_cadena_accesos_rrhh
 SET ultima_secuencia=1,cabeza_sha256=repeat('a',64)
 WHERE control;
COMMIT;
SQL
if psql_archivo "$down_ct" > "$temporal/down_ct_historia.log" 2>&1; then
 echo 'CT109 DOWN aceptó cadena CT avanzada' >&2
 exit 1
fi
rg -q 'historia o genesis incompatible' "$temporal/down_ct_historia.log" || {
 echo 'CT109 DOWN no comprobó historia CT' >&2
 tail -n 15 "$temporal/down_ct_historia.log" >&2
 exit 1
}
psql_sql <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
UPDATE vec_contratacion_temporal.control_cadena_accesos_rrhh
 SET ultima_secuencia=0,cabeza_sha256=repeat('0',64)
 WHERE control;
COMMIT;
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
UPDATE vec_autorizacion_atestada_v3.control_cadena_auditoria
 SET secuencia=1,cabeza_sha256=repeat('b',64)
 WHERE control_id;
COMMIT;
SQL
if psql_archivo "$down_ct" > "$temporal/down_v3_historia.log" 2>&1; then
 echo 'CT109 DOWN aceptó cadena V3 avanzada' >&2
 exit 1
fi
rg -q 'historia o genesis incompatible' "$temporal/down_v3_historia.log" || {
 echo 'CT109 DOWN no comprobó historia V3' >&2
 tail -n 15 "$temporal/down_v3_historia.log" >&2
 exit 1
}
psql_sql <<'SQL'
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
UPDATE vec_autorizacion_atestada_v3.control_cadena_auditoria
 SET secuencia=0,cabeza_sha256=repeat('0',64)
 WHERE control_id;
COMMIT;
SQL
psql_archivo "$down_ct"
psql_archivo "$down_ca"
psql_sql <<'SQL'
DO $acl_restituida$
BEGIN
 IF to_regprocedure('vec_contexto_actor_v1.acreditar_ambito_rrhh_v1(jsonb,text,text,text,text,text)') IS NOT NULL
 OR to_regprocedure('vec_contratacion_temporal.consultar_cuadro_rrhh_ambito_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR has_schema_privilege('vec_contexto_actor_corporativo_rrhh_selector',
   'vec_contexto_actor_v1','USAGE')
 OR has_schema_privilege('vec_contratacion_temporal_propietario',
   'vec_contexto_actor_v1','USAGE')
 OR NOT has_schema_privilege('vec_contexto_actor_v1_runtime',
   'vec_contexto_actor_v1','USAGE')
 THEN RAISE EXCEPTION 'CA6/CT109: ACL no restituidas'; END IF;
END $acl_restituida$;
SQL
psql_archivo "$raiz/deploy/postgresql/contexto_actor_v1/migraciones/000006_acreditacion_ambito_rrhh_v1.up.sql"
psql_archivo "$raiz/deploy/postgresql/contratacion_temporal/migraciones/000109_consultas_rrhh_ambito_v1.up.sql"

# El positivo usa la misma preimagen y migraciones, pero después del ensayo
# DOWN/UP vacío: un acceso real avanza las cadenas CT y V3 y ya no admite DOWN.
psql_archivo "$raiz/deploy/postgresql/contexto_actor_v1/pruebas_sql/ct109_positivo_fixture.sql"
docker exec -i "$contenedor" psql -Xq -v ON_ERROR_STOP=1 \
  -U ct109_motivos_proyector -d postgres \
  < "$raiz/deploy/postgresql/contexto_actor_v1/pruebas_sql/ct109_positivo_motivos.sql"
(cd "$raiz" && CGO_ENABLED=0 GOTOOLCHAIN=local "$go_pruebas" test -c \
  -o "$temporal/ct109_positivo.test" ./internal/app/bootstrap)
chmod 700 "$temporal/ct109_positivo.test"
docker cp "$temporal/ct109_positivo.test" "$contenedor:/tmp/ct109_positivo.test"
docker exec \
  -e VEC_CT109_PG_DESECHABLE=1 \
  -e 'VEC_CT109_PG_DSN=postgres://postgres@/postgres?host=/var/run/postgresql' \
  "$contenedor" /tmp/ct109_positivo.test -test.run '^TestCT109PositivoPostgreSQL$' -test.v
psql_archivo "$raiz/deploy/postgresql/contexto_actor_v1/pruebas_sql/ct109_positivo_comprobar.sql"
echo 'CA6 + AD3-51 + CT109: ACL, concurrencia, DOWN/UP vacío y consulta positiva nominal OK'
