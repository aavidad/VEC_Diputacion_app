#!/usr/bin/env bash
# Ensayo nominal RPT27/AD154 sobre la captura causal POST149 de principal.
# Reutiliza el emisor y el transporte existentes, sin reconstruir Baremo.
# Dirección cede el clon tras fijar preimagen y revisar el contenido exacto.
set -euo pipefail
umask 077
continuar=false
desde_registro=false
desde_post154=false
desde_replay=false
desde_checkpoint=false
desde_go=false
solo_concurrente=false
gobierno_version=69
gobierno_anterior=68
gobierno_minutos=60
if [[ ${1:-} == --continuar-go ]]; then continuar=true; desde_go=true; shift; fi
if [[ ${1:-} == --continuar-postcheckpoint ]]; then continuar=true; desde_replay=true; desde_checkpoint=true; shift; fi
if [[ ${1:-} == --continuar-desde-replay ]]; then continuar=true; desde_replay=true; shift; fi
if [[ ${1:-} == --desde-post154 ]]; then desde_post154=true; shift; fi
if [[ ${1:-} == --continuar-fixture ]]; then continuar=true; shift; fi
if [[ ${1:-} == --continuar-registro ]]; then continuar=true; desde_registro=true; shift; fi
[[ $# == 0 ]] || { printf '%s\n' 'RPT27: argumentos de ensayo inválidos' >&2; exit 2; }
base_dir=$(CDPATH='' cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repo_dir=$(CDPATH='' cd -- "$base_dir/../../../.." && pwd)
fallo() { printf 'RPT27 FALLO: %s\n' "$1" >&2; exit 1; }
container=${VEC_RPT27_CLONE_CONTAINER:?clon privado autorizado requerido}
scratch=${VEC_RPT27_SCRATCH:?scratch propio externo requerido}
modcache=${VEC_RPT27_MODCACHE:?módulos locales sin descargas requeridos}
cache=${VEC_RPT27_CACHE:?cache en disco autorizada requerida}
socket=${VEC_RPT27_SOCKET:?socket Unix del clon requerido}
ad154=${VEC_RPT27_UP_AD154:?UP154 activado y revisado requerido}
personal27=${VEC_RPT27_UP_PERSONAL27:?UP27 activado y revisado requerido}
pre_sha=${VEC_RPT27_PREIMAGEN_SHA256:?SHA de captura real requerida}
helper="$scratch/generate_overlay.py"
hmac_nombre=hmac.bin
"$desde_go" && hmac_nombre=hmac69.bin
[[ $container == codexb-rpt-comun-20261003-pg ]] || fallo 'clon nominal propio requerido'
[[ $scratch == /home/alberto/.local/state/vec-codexb-ca27-ensayo-20261003/scratch && -d $scratch && ! -L $scratch && -O $scratch && $(stat -c %a "$scratch") == 700 ]] || fallo 'scratch propio0700 requerido'
[[ $cache == /home/alberto/.cache/go-build && -d $cache && ! -L $cache && -O $cache ]] || fallo 'cache compartida en disco requerida'
[[ $socket == /home/alberto/.local/state/vec-codexb-ca27-ensayo-20261003/data/pgdata/socket && -S $socket/.s.PGSQL.5432 ]] || fallo 'socket ajeno o ausente'
[[ -d $modcache && ! -L $modcache && $pre_sha =~ ^[0-9a-f]{64}$ ]] || fallo 'faltan módulos o huella real'
[[ $ad154 == "$repo_dir"/deploy/postgresql/autorizacion_atestada_v3/migraciones/000154_*.up.sql && -f $ad154 && ! -L $ad154 ]] || fallo 'no aplicar borrador ni otra AD'
[[ $personal27 == "$repo_dir"/deploy/postgresql/personal/migraciones/000027_*.up.sql && -f $personal27 && ! -L $personal27 ]] || fallo 'no aplicar borrador ni otra Personal'
[[ $(docker inspect --format '{{.HostConfig.NetworkMode}} {{.HostConfig.Memory}} {{.HostConfig.NanoCpus}} {{.HostConfig.PidsLimit}}' "$container") == 'none 2147483648 2000000000 128' ]] || fallo 'clon fuera de límites aislados'
psql_run() { local user=$1; shift; docker exec -i "$container" /usr/bin/env -i PATH=/usr/local/bin:/usr/bin:/bin psql -h /tmp -X -qAt -v ON_ERROR_STOP=1 -v VERBOSITY=verbose -U "$user" -d postgres "$@"; }
valor() { psql_run postgres -c "$1"; }
# AD172 conserva consumo_confirmado histórico (sin versión/origen) y añade
# consumo_confirmado_v2/2. to_jsonb permite leer también el esquema anterior
# sin version_consumo. AD173 queda cerrado hasta publicar y revisar su ABI.
# El contador no verifica huellas: rechaza familias o versiones incompatibles y
# cuenta ambos tipos permitidos sin reescribir ni excluir los históricos.
contadores() {
 local estado
 estado=$(valor "$(cat <<'SQLCONTADORES'
WITH auditoria AS (
 SELECT a.tipo_registro,a.proceso,a.canal,
        to_jsonb(a)->'version_consumo' AS version_consumo
 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
)
SELECT CASE WHEN count(*) FILTER (WHERE NOT coalesce(
 CASE tipo_registro
  WHEN 'consumo_confirmado' THEN
   coalesce(version_consumo,'null'::jsonb)='null'::jsonb
   AND proceso IS NULL AND canal IS NULL
  WHEN 'consumo_confirmado_v2' THEN
   version_consumo='2'::jsonb AND proceso IS NOT NULL
   AND proceso ~ '^[a-z][a-z0-9._-]{1,79}$'
   AND canal IN ('interna_corporativa','administracion_privilegiada','externa_personal')
  WHEN 'intento_nominal' THEN coalesce(version_consumo,'null'::jsonb)='null'::jsonb
  WHEN 'preperfil_autenticado' THEN coalesce(version_consumo,'null'::jsonb)='null'::jsonb
  WHEN 'bootstrap_operador' THEN coalesce(version_consumo,'null'::jsonb)='null'::jsonb
  ELSE false
 END,false))=0 THEN
 (SELECT count(*) FROM vec_personal.recibo_relacion_para_rpt)||'|'||
 (SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3)||'|'||
 count(*) FILTER (WHERE tipo_registro IN ('consumo_confirmado','consumo_confirmado_v2'))||'|'||
 count(*) FILTER (WHERE tipo_registro='intento_nominal')
 ELSE 'auditoria_incompatible' END
FROM auditoria;
SQLCONTADORES
 )") || fallo 'consulta de contadores falló'
 [[ $estado =~ ^[0-9]+\|[0-9]+\|[0-9]+\|[0-9]+$ ]] || fallo 'familia o versión de auditoría incompatible'
 printf '%s\n' "$estado"
}
archivo() { psql_run postgres -v rpt27_ensayo_autorizado=on < "$1" > "$scratch/sql.log" 2>&1 || fallo 'SQL falló: diagnóstico privado'; }
[[ $(valor "SELECT current_setting('server_version_num')") == 180004 ]] || fallo 'PG18.4 requerido'
[[ $(valor "SELECT to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(bytea,bytea,jsonb)') IS NOT NULL AND to_regprocedure('vec_contexto_actor_v1.cotejar_contexto_historico_auditoria_v1(text,text,text,bytea)') IS NOT NULL AND to_regprocedure('vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(bytea)') IS NOT NULL") == t ]] || fallo 'auditoría común AD169/CA26/IS13 requerida; no instalar desde este runner'
if ! "$continuar"; then
[[ ! -e $scratch/overlay.json && ! -e $scratch/capacidad_v3_vector_sql_test.go ]] || fallo 'scratch ya usado: conservarlo'
fi
if ! "$continuar"; then
if "$desde_post154"; then
 [[ -f $scratch/migraciones_journal.txt ]] || fallo 'POST154 sin journal conservado'
 sha256sum "$ad154" "$personal27" | cmp -s "$scratch/migraciones_journal.txt" - || fallo 'producto distinto del instalado'
 [[ $(valor "SELECT to_regprocedure('vec_personal.consultar_relacion_para_rpt_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL AND to_regprocedure('vec_autorizacion_atestada_v3.consumir_relacion_para_rpt_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL AND to_regclass('public.rpt27_ensayo_vector') IS NULL") == t ]] || fallo 'POST154 incompleta o fixture ya presente'
else
 [[ $(valor "SELECT to_regprocedure('vec_personal.consultar_relacion_para_rpt_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL AND to_regprocedure('vec_autorizacion_atestada_v3.consumir_relacion_para_rpt_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL") == t ]] || fallo 'RPT27/154 ya presentes: no reaplicar'
fi
else
 [[ -f $scratch/preimagen.json && -f $scratch/migraciones_journal.txt && -f $scratch/preservacion.sql ]] || fallo 'fase anterior incompleta'
 [[ $(sha256sum "$scratch/preimagen.json" | cut -d' ' -f1) == "$pre_sha" ]] || fallo 'captura de reanudación distinta'
 sha256sum "$ad154" "$personal27" | cmp -s "$scratch/migraciones_journal.txt" - || fallo 'producto cambiado desde instalación'
 if "$desde_go"; then
  estado_go=$(contadores)
  if [[ $estado_go == '6|6246|6246|14' ]]; then
   solo_concurrente=true;gobierno_version=70;gobierno_anterior=69;gobierno_minutos=240;hmac_nombre=hmac70.bin
   [[ -f $scratch/rpt27.test && $(sha256sum "$scratch/rpt27.test" | cut -d' ' -f1) == "${VEC_RPT27_BINARIO_SHA256:?SHA del binario conservado requerido}" ]] || fallo 'binario conservado distinto'
   [[ $(valor "SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex')='dd0b06eb3ecb2a8317054739ad01f6c363c725f477d7f98d345f01d83487a78f' FROM pg_proc WHERE oid='vec_personal.consultar_relacion_para_rpt_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure") == t ]] || fallo 'Personal30 no acreditada'
  else
   [[ $estado_go == '4|6244|6244|11' ]] || fallo 'fase SQL conservada distinta'
  fi
  [[ $(valor "SELECT count(*)=1 FROM public.rpt27_ensayo_vector WHERE caso='go_positivo' AND material IS NOT NULL AND bundle IS NOT NULL") == t ]] || fallo 'primer fixture Go anterior ausente'
  if ! "$solo_concurrente"; then
   [[ $(valor "SELECT count(*)=0 FROM public.rpt27_ensayo_vector WHERE caso LIKE 'go_%_v69'") == t ]] || fallo 'continuación Go ya iniciada; conservar sus efectos'
  else
   [[ $(valor "SELECT count(*)=0 FROM public.rpt27_ensayo_vector WHERE caso='go_concurrente_v70'") == t ]] || fallo 'caso final ya preparado'
  fi
 elif "$desde_replay"; then
  if ! "$desde_checkpoint"; then
  binario_sha=${VEC_RPT27_BINARIO_SHA256:?huella del único binario conservado requerida}
  [[ $binario_sha =~ ^[0-9a-f]{64}$ && $(sha256sum "$scratch/rpt27.test" | cut -d' ' -f1) == "$binario_sha" ]] || fallo 'binario distinto del ensayado'
  else
   [[ ! -e $scratch/rpt27.test && ! -e $scratch/overlay.json ]] || fallo 'checkpoint con artefactos de otra compilación'
   [[ $(contadores) == '3|6243|6243|0' ]] || fallo 'checkpoint con efectos distintos'
  fi
  [[ $(valor "SELECT count(*)=3 AND bool_and(caso IN ('positivo_vigente','positivo_suspendida','positivo_finalizada')) FROM public.rpt27_ensayo_vector") == t ]] || fallo 'fase previa distinta de tres positivos'
  [[ $(valor "SELECT count(*)=3 FROM vec_personal.recibo_relacion_para_rpt") == t ]] || fallo 'recibos previos distintos'
  [[ $(valor "SELECT count(*)=0 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE tipo_registro='intento_nominal' AND proceso='rpt27-ensayo'") == t ]] || fallo 'replay ya confirmado; no repetir'
 elif ! "$desde_registro"; then
 [[ -f $scratch/rpt27.test ]] || fallo 'binario previo ausente'
 [[ $(valor "SELECT count(*)=0 FROM public.rpt27_ensayo_vector") == t ]] || fallo 'ya hay casos preparados: no repetir el ensayo'
 else
  [[ -f $scratch/rpt27.test ]] || fallo 'binario previo ausente'
  [[ $(valor "SELECT count(*)=1 FROM public.rpt27_ensayo_vector WHERE caso='go_registro_caido'") == t ]] || fallo 'caso caído no preparado'
  [[ $(valor "SELECT count(*)=0 FROM public.rpt27_ensayo_vector WHERE caso IN ('go_revocada','go_concurrente')") == t ]] || fallo 'fases finales ya preparadas: no repetir'
 fi
fi
captura() { { printf '%s\n' 'SET search_path=pg_catalog,pg_temp;'; cat "$scratch/preservacion.sql"; } | psql_run postgres > "$1"; }
if ! "$continuar"; then
# Snapshot de todas las fachadas A previas y CHECK. El núcleo se normaliza con
# el inverso textual del delta154 congelado, jamás ejecutando DOWN. El contrato
# de segmentos debe coincidir con AD154; su ausencia detiene antes de aplicar.
python3 - "$ad154" "$scratch/preservacion.sql" <<'PYPRESERVACION'
import pathlib,re,sys
s=pathlib.Path(sys.argv[1]).read_text()
def literal(n):
 a=re.findall(r'\b'+n+r' text:=\$'+n+r'\$(.*?)\$'+n+r'\$;',s,re.S)
 assert len(a)==1, 'delta154 debe declarar cada segmento una vez'
 t='$rpt27'+n+'$';assert t not in a[0];return t+a[0]+t
inv='replace(replace(replace(pg_get_functiondef(p.oid),'+literal('extension')+'||'+literal('marca')+','+literal('marca')+'),'+literal('excl_nuevo')+','+literal('excl')+'),'+literal('runtime_nuevo')+','+literal('runtime')+')'
q="""SELECT jsonb_build_object('funciones',(SELECT jsonb_agg(jsonb_build_object('oid',p.oid,'metadatos',to_jsonb(p)-'prosrc',
 'definicion_sha256',encode(sha256(convert_to(CASE WHEN p.proname='consumir_decision_mutacion_v3_interna' THEN __INV__ ELSE pg_get_functiondef(p.oid) END,'UTF8')),'hex'),
 'dependencias',coalesce((SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=p.oid),'[]'::jsonb),
 'dependencias_compartidas',coalesce((SELECT jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype) FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=p.oid),'[]'::jsonb)) ORDER BY p.oid)
 FROM pg_proc p WHERE p.pronamespace='vec_autorizacion_atestada_v3'::regnamespace AND p.oid IS DISTINCT FROM to_regprocedure('vec_autorizacion_atestada_v3.consumir_relacion_para_rpt_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')),
 'fuente17',jsonb_build_object(
 'filas',(__FILAS17__),
 'tabla',(SELECT jsonb_build_object('owner',c.relowner,'acl',c.relacl,'rls',c.relrowsecurity,'forzada',c.relforcerowsecurity) FROM pg_class c WHERE c.oid='vec_personal.relacion_servicio_historia'::regclass),
 'triggers',(SELECT jsonb_agg(to_jsonb(t) ORDER BY t.oid) FROM pg_trigger t WHERE t.tgrelid='vec_personal.relacion_servicio_historia'::regclass AND NOT(t.tgname='relacion_rpt_generacion_insertada' AND t.tgtype=5 AND t.tgfoid=to_regprocedure('vec_personal.avanzar_generacion_relacion_rpt_v1()'))),
 'validador_sha256',(SELECT encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') FROM pg_proc p WHERE p.oid='vec_personal.validar_revision_registro_empleado_v1()'::regprocedure)),
 'audiencias',(SELECT jsonb_build_object('validada',c.convalidated,'definicion',replace(pg_get_constraintdef(c.oid,true), __RETIRAR_AUDIENCIA__, '')) FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass AND c.conname='clave_capacidad_version_audiencia_consumo_check'));"""
q=q.replace('__INV__',inv).replace('__RETIRAR_AUDIENCIA__', "$rpt27audiencia$, 'vec_personal.relacion_rpt.v1'::text$rpt27audiencia$")
filas="SELECT coalesce(jsonb_agg(to_jsonb(h) ORDER BY h.relacion_ref,h.revision),'[]'::jsonb) FROM vec_personal.relacion_servicio_historia h"
pathlib.Path(sys.argv[2]+'.plantilla').write_text(q)
pathlib.Path(sys.argv[2]).write_text(q.replace('__FILAS17__',filas))
PYPRESERVACION

captura "$scratch/preimagen.json"
[[ $(sha256sum "$scratch/preimagen.json" | cut -d' ' -f1) == "$pre_sha" ]] || fallo 'preimagen distinta de la aprobada'
# Fijar los identificadores originales de17: añadir fixtures después no altera
# la comparación; cambiar o borrar una fila previa sí debe hacerla fallar.
python3 - "$scratch/preimagen.json" "$scratch/preservacion.sql" <<'PYCLAVES17'
import json,pathlib,sys
p=pathlib.Path(sys.argv[2]);j=json.loads(pathlib.Path(sys.argv[1]).read_text())
claves=[{'relacion_ref':h['relacion_ref'],'revision':h['revision']} for h in j['fuente17']['filas']]
v=json.dumps(claves,separators=(',',':'));assert '$rpt27claves17$' not in v
filas="SELECT coalesce(jsonb_agg(to_jsonb(h) ORDER BY h.relacion_ref,h.revision),'[]'::jsonb) FROM vec_personal.relacion_servicio_historia h WHERE(h.relacion_ref,h.revision) IN(SELECT x->>'relacion_ref',(x->>'revision')::int FROM jsonb_array_elements($rpt27claves17$"+v+"$rpt27claves17$::jsonb) x)"
p.write_text(pathlib.Path(str(p)+'.plantilla').read_text().replace('__FILAS17__',filas))
PYCLAVES17
if "$desde_post154"; then
 sha256sum "$ad154" "$personal27" | cmp -s "$scratch/migraciones_journal.txt" - || fallo 'journal POST154 divergente'
else
 sha256sum "$ad154" "$personal27" > "$scratch/migraciones_journal.txt"
 archivo "$ad154"; archivo "$personal27"
fi
captura "$scratch/post154.json"
cmp -s "$scratch/preimagen.json" "$scratch/post154.json" || fallo 'delta154 alteró autoridades previas'
# La comparación inversa conserva cada audiencia previa; el literal nuevo
# debe aparecer exactamente una vez en la postimagen aprobada.
[[ $(valor "SELECT length(pg_get_constraintdef(c.oid,true))-length(replace(pg_get_constraintdef(c.oid,true), ', ''vec_personal.relacion_rpt.v1''::text','')) = length(', ''vec_personal.relacion_rpt.v1''::text') FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.convalidated") == t ]] || fallo 'audiencia nominal ausente o duplicada'
archivo "$base_dir/relacion_para_rpt_000027.sql"
valor "CREATE ROLE vec_rpt27_ensayo_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
 GRANT vec_personal_ejecutor TO vec_rpt27_ensayo_runtime WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
 GRANT CONNECT ON DATABASE postgres TO vec_rpt27_ensayo_runtime;
 CREATE ROLE vec_rpt27_ensayo_registrador LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
 GRANT vec_autorizacion_atestada_v3_registrador_intentos TO vec_rpt27_ensayo_registrador WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
 GRANT CONNECT ON DATABASE postgres TO vec_rpt27_ensayo_registrador;
 INSERT INTO vec_autorizacion_atestada_v3.configuracion_runtime_intentos(login_nombre,proceso,canal)
 VALUES('vec_rpt27_ensayo_registrador','rpt27-ensayo','interna_corporativa');
 CREATE ROLE vec_rpt27_ensayo_ca LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
 GRANT vec_contexto_actor_v1_runtime TO vec_rpt27_ensayo_ca WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
 GRANT USAGE ON SCHEMA public TO vec_rpt27_ensayo_runtime;
 GRANT SELECT ON public.rpt27_ensayo_vector TO vec_rpt27_ensayo_runtime;
 GRANT EXECUTE ON FUNCTION public.rpt27_ensayo_consultar(text,text) TO vec_rpt27_ensayo_runtime;" > /dev/null
head -c32 /dev/urandom > "$scratch/hmac.bin"
python3 - "$scratch/hmac.bin" "$scratch/gobierno.sql" <<'PYGOBIERNO'
import pathlib,sys
h=pathlib.Path(sys.argv[1]).read_bytes().hex()
pathlib.Path(sys.argv[2]).write_text("BEGIN; SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario; INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version(clave_id,version,revision_gobierno,huella_gobierno_sha256,secreto_hmac,huella_secreto_sha256,emisor_id,audiencia_consumo,valida_desde,valida_hasta,acto_ref) SELECT 'clave:rpt27:ensayo',max(version)+1,max(revision_gobierno)+1,repeat('f',64),decode('"+h+"','hex'),encode(sha256(decode('"+h+"','hex')),'hex'),'broker-rpt27-sintetico','vec_personal.relacion_rpt.v1',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '2 hours','acto:rpt27:clave' FROM vec_autorizacion_atestada_v3.clave_capacidad_version; INSERT INTO vec_autorizacion_atestada_v3.puntero_clave_emision(orden,clave_id,version,establecida_en,acto_ref) SELECT (SELECT max(orden)+1 FROM vec_autorizacion_atestada_v3.puntero_clave_emision),clave_id,version,clock_timestamp(),'acto:rpt27:puntero-clave' FROM vec_autorizacion_atestada_v3.clave_capacidad_version WHERE clave_id='clave:rpt27:ensayo'; COMMIT;")
PYGOBIERNO
archivo "$scratch/gobierno.sql"
secuencia=$(valor 'SELECT ultima_secuencia+1 FROM vec_autorizacion.motivo_v2_checkpoint_origen WHERE control_id')
valor "BEGIN; SET LOCAL ROLE vec_autorizacion_motivos_proyector; SELECT vec_autorizacion.publicar_motivos_autorizacion_v2('evento_1111111111111111111111111111a027',$secuencia,repeat('e',64),'motivos_rpt27_ensayo',1,repeat('e',64),clock_timestamp()-interval '1 minute',jsonb_build_array(jsonb_build_object('clave','motivo_11111111111111111111111111111111','vigente_desde',to_char(clock_timestamp()-interval '2 minutes','YYYY-MM-DD\"T\"HH24:MI:SS.US\"Z\"'),'vigente_hasta',NULL))); COMMIT;" > "$scratch/motivo.log" 2>&1 || fallo 'publicación de motivo nominal falló'
fi
if "$desde_checkpoint"; then
 # Recuperar exclusivamente la misma clave SINTÉTICA instalada. No se crea otra
 # versión ni se renueva su vigencia. El material permanece en scratch0600.
 [[ $(valor "SELECT count(*)=1 AND bool_and(octet_length(secreto_hmac)=32 AND encode(sha256(secreto_hmac),'hex')=huella_secreto_sha256 AND clock_timestamp()>=valida_desde AND clock_timestamp()<valida_hasta AND emisor_id='broker-rpt27-sintetico' AND audiencia_consumo='vec_personal.relacion_rpt.v1') FROM vec_autorizacion_atestada_v3.clave_capacidad_version WHERE clave_id='clave:rpt27:ensayo'") == t ]] || fallo 'clave sintética original ausente, divergente o caducada'
 psql_run postgres -c "SELECT encode(secreto_hmac,'hex') FROM vec_autorizacion_atestada_v3.clave_capacidad_version WHERE clave_id='clave:rpt27:ensayo'" | python3 -c 'import os,pathlib,re,sys; v=sys.stdin.read().strip(); assert re.fullmatch("[0-9a-f]{64}",v), "clave sintética inválida"; p=pathlib.Path(sys.argv[1]); f=os.open(p,os.O_WRONLY|os.O_CREAT|os.O_EXCL|os.O_NOFOLLOW,0o600); os.write(f,bytes.fromhex(v)); os.close(f)' "$scratch/hmac.bin"
 captura "$scratch/reanudacion.json"
 cmp -s "$scratch/preimagen.json" "$scratch/reanudacion.json" || fallo 'checkpoint alteró autoridades previas'
fi
if "$desde_go"; then
 # Gobierno SINTÉTICO autorizado por Dirección: nueva versión append-only.
 # La 68, sus materiales y contextos permanecen intactos. No cambia permisos.
 vigencia_minutos=${VEC_RPT27_CLAVE_ENSAYO_MINUTOS:-$gobierno_minutos}
 [[ $vigencia_minutos =~ ^[1-9][0-9]{0,2}$ ]] || fallo 'duración de fixture inválida'
 clave_anterior_sha=$(valor "SELECT encode(sha256(convert_to(to_jsonb(k)::text,'UTF8')),'hex') FROM vec_autorizacion_atestada_v3.clave_capacidad_version k WHERE clave_id='clave:rpt27:ensayo' AND version=$gobierno_anterior")
 [[ $clave_anterior_sha =~ ^[0-9a-f]{64}$ ]] || fallo 'fuente de gobierno anterior ausente'
 if [[ $(valor "SELECT count(*)=1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version WHERE clave_id='clave:rpt27:ensayo' AND version=$gobierno_version") == t ]]; then
  [[ -f $scratch/$hmac_nombre && ! -L $scratch/$hmac_nombre && $(stat -c %s "$scratch/$hmac_nombre") == 32 ]] || fallo 'material69 previo ausente'
  [[ $(sha256sum "$scratch/$hmac_nombre" | cut -d' ' -f1) == $(valor "SELECT huella_secreto_sha256 FROM vec_autorizacion_atestada_v3.clave_capacidad_version WHERE clave_id='clave:rpt27:ensayo' AND version=$gobierno_version AND clock_timestamp()>=valida_desde AND clock_timestamp()<valida_hasta") ]] || fallo 'material69 previo divergente o caducado'
  [[ $(valor "SELECT position('clock_timestamp()>=valida_desde AND clock_timestamp()<valida_hasta' in prosrc)>0 FROM pg_proc WHERE oid='public.rpt27_ensayo_entrada(text)'::regprocedure") == t ]] || fallo 'selector vigente previo ausente'
 else
 python3 - "$scratch/$hmac_nombre" "$scratch/gobierno69.sql" "$vigencia_minutos" "$gobierno_version" "$gobierno_anterior" <<'PYGOBIERNO69'
import os,pathlib,secrets,sys
p=pathlib.Path(sys.argv[1]);h=secrets.token_bytes(32)
f=os.open(p,os.O_WRONLY|os.O_CREAT|os.O_EXCL|os.O_NOFOLLOW,0o600)
assert os.write(f,h)==32;os.close(f)
q="""BEGIN; SET LOCAL search_path=pg_catalog; SET LOCAL timezone='UTC';
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version
(clave_id,version,revision_gobierno,huella_gobierno_sha256,secreto_hmac,huella_secreto_sha256,emisor_id,audiencia_consumo,valida_desde,valida_hasta,acto_ref)
SELECT clave_id,__VERSION__,(SELECT max(revision_gobierno)+1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version),repeat('f',64),decode('__H__','hex'),encode(sha256(decode('__H__','hex')),'hex'),emisor_id,audiencia_consumo,clock_timestamp()-interval '1 minute',clock_timestamp()+make_interval(mins=>__MIN__),'acto:rpt27:clave:version__VERSION__'
FROM vec_autorizacion_atestada_v3.clave_capacidad_version WHERE clave_id='clave:rpt27:ensayo' AND version=__ANTERIOR__ AND clock_timestamp()>=valida_hasta;
INSERT INTO vec_autorizacion_atestada_v3.puntero_clave_emision(orden,clave_id,version,establecida_en,acto_ref)
SELECT (SELECT max(orden)+1 FROM vec_autorizacion_atestada_v3.puntero_clave_emision),clave_id,version,clock_timestamp(),'acto:rpt27:puntero-clave:version__VERSION__'
FROM vec_autorizacion_atestada_v3.clave_capacidad_version WHERE clave_id='clave:rpt27:ensayo' AND version=__VERSION__;
COMMIT;
"""
pathlib.Path(sys.argv[2]).write_text(q.replace('__H__',h.hex()).replace('__MIN__',sys.argv[3]).replace('__VERSION__',sys.argv[4]).replace('__ANTERIOR__',sys.argv[5]))
PYGOBIERNO69
 archivo "$scratch/gobierno69.sql"
 [[ $(valor "SELECT encode(sha256(convert_to(to_jsonb(k)::text,'UTF8')),'hex') FROM vec_autorizacion_atestada_v3.clave_capacidad_version k WHERE clave_id='clave:rpt27:ensayo' AND version=$gobierno_anterior") == "$clave_anterior_sha" ]] || fallo 'gobierno modificó historia anterior'
 [[ $(valor "SELECT count(*)=1 AND bool_and(version=$gobierno_version AND octet_length(secreto_hmac)=32 AND encode(sha256(secreto_hmac),'hex')=huella_secreto_sha256) FROM vec_autorizacion_atestada_v3.clave_capacidad_version WHERE clave_id='clave:rpt27:ensayo' AND clock_timestamp()>=valida_desde AND clock_timestamp()<valida_hasta") == t ]] || fallo 'clave vigente divergente'
 # Sólo la función auxiliar del fixture: selector por vigencia, nunca UPDATE
 # de clave, sesión, datos anteriores o funciones instaladas de producto.
 python3 - "$base_dir/relacion_para_rpt_000027.sql" "$scratch/selector69.sql" <<'PYSELECTOR69'
import pathlib,re,sys
s=pathlib.Path(sys.argv[1]).read_text()
m=re.search(r'CREATE FUNCTION public\.rpt27_ensayo_entrada\(.*?\$f\$;',s,re.S);assert m
pathlib.Path(sys.argv[2]).write_text('BEGIN;\n'+m[0].replace('CREATE FUNCTION','CREATE OR REPLACE FUNCTION',1)+'\nCOMMIT;\n')
PYSELECTOR69
 if ! "$solo_concurrente"; then archivo "$scratch/selector69.sql"; fi
 fi
 [[ $(valor "SELECT count(*)=1 AND bool_and(version=$gobierno_version AND octet_length(secreto_hmac)=32 AND encode(sha256(secreto_hmac),'hex')=huella_secreto_sha256) FROM vec_autorizacion_atestada_v3.clave_capacidad_version WHERE clave_id='clave:rpt27:ensayo' AND clock_timestamp()>=valida_desde AND clock_timestamp()<valida_hasta") == t ]] || fallo 'clave vigente divergente'
 captura "$scratch/reanudacion.json"
 cmp -s "$scratch/preimagen.json" "$scratch/reanudacion.json" || fallo 'continuación alteró autoridades previas'
fi
if { ! "$continuar" || "$desde_checkpoint" || "$desde_go"; } && ! "$solo_concurrente"; then
# Generación de overlay con fuente original de sólo lectura.
cat > "$helper" <<'RPT27_GENERADOR_PY'
#!/usr/bin/env python3
# Generates an ephemeral Go test overlay without changing the target worktree.
import json
import os
import pathlib
import sys

if len(sys.argv) != 3:
    raise SystemExit('usage: generate_overlay.py WORKTREE SCRATCH')
root = pathlib.Path(sys.argv[1]).resolve(strict=True)
scratch = pathlib.Path(sys.argv[2]).resolve(strict=True)
source = root / 'internal/vec/adapters/seguridad/confianzaatestacion/capacidad_v3_vector_sql_test.go'
original = source.read_text()
original = original.replace('"crypto/rand"', '"crypto/rand"\n\t"crypto/sha256"\n\t"fmt"\n\t"io"')
original = original.replace('"vec-diputacion-granada/internal/vec/domain"', '"vec-diputacion-granada/internal/vec/application"\n\t"vec-diputacion-granada/internal/vec/domain"')
original = original.replace('"vec-diputacion-granada/internal/vec/ports"', '"vec-diputacion-granada/internal/vec/ports"\n\t"github.com/jackc/pgx/v5/pgxpool"\n\tobservabilidad "vec-diputacion-granada/internal/vec/adapters/observabilidad"\n\tvecpg "vec-diputacion-granada/internal/vec/adapters/postgres"\n\tpersonalcomposicion "vec-diputacion-granada/internal/modules/personal/adapters/composicion"\n\tpersonalrpt "vec-diputacion-granada/internal/modules/personal/adapters/rpt"\n\tpersonalpg "vec-diputacion-granada/internal/modules/personal/adapters/postgres"\n\tpersonalapp "vec-diputacion-granada/internal/modules/personal/application"\n\tpersonaldomain "vec-diputacion-granada/internal/modules/personal/domain"\n\tpersonalports "vec-diputacion-granada/internal/modules/personal/ports"')
extra = r'''
// RPT27 is an ephemeral synthetic PDP fixture. Neither this test nor its
// import bridge establishes production identity, PDP governance or database I/O.
// The parent SQL runner must register the exact preparation before importing
// its receipt; SQL consumers revalidate that durable state before any effect.
const rpt27Accion = "personal.relacion_rpt.consultar"
const rpt27Audiencia = "vec_personal.relacion_rpt.v1"
const rpt27Finalidad = "conciliar_relacion_laboral_para_rpt"
const rpt27Operacion = "relacion_para_rpt"
const rpt27Tipo = "relacion_para_rpt"
const rpt27FormatoTiempo = "2006-01-02T15:04:05.000000Z"

var rpt27Campos = []string{"cobertura", "corte", "estado", "periodo", "procedencia", "version"}

type rpt27Confirmacion struct {
	DecisionRef          string `json:"decision_ref"`
	DecisionHuellaSHA256 string `json:"decision_huella_sha256"`
	EmitidaEn            string `json:"emitida_en"`
	ValidaHasta          string `json:"valida_hasta"`
	RegistradaEn         string `json:"registrada_en"`
}
type rpt27Entrada struct {
	entradaVectorSQLO205
	Fase                string             `json:"fase"`
	EmpleadoRef         string             `json:"empleado_ref"`
	RelacionRef         string             `json:"relacion_ref"`
	OrganismoRef        string             `json:"organismo_ref"`
	VersionEsperada     int64              `json:"version_esperada"`
	VigenteEn           string             `json:"vigente_en"`
	ConocidoEn          string             `json:"conocido_en"`
	MaterialCanonicoB64 string             `json:"material_canonico_b64"`
	ClaveHMACArchivo    string             `json:"clave_hmac_archivo"`
	Confirmacion        *rpt27Confirmacion `json:"confirmacion,omitempty"`
}
type rpt27Salida struct {
	bundleVectorO205
	Fase                 string                       `json:"fase"`
	MaterialCanonicoB64  string                       `json:"material_canonico_b64"`
	MaterialHuellaSHA256 string                       `json:"material_huella_sha256"`
	DecisionHuellaSHA256 string                       `json:"decision_huella_sha256"`
	Confirmacion         *rpt27Confirmacion           `json:"confirmacion,omitempty"`
	Campos               []string                     `json:"campos"`
	Obligaciones         []string                     `json:"obligaciones"`
	Politicas            []domain.PoliticaRestrictiva `json:"politicas"`
	RevisionCatalogo     uint64                       `json:"revision_catalogo"`
	HuellaCatalogoSHA256 string                       `json:"huella_catalogo_sha256"`
	PDP                  string                       `json:"pdp"`
}

func TestGenerarVectorRPT27ParaSQL(t *testing.T) {
	input, output := os.Getenv("VEC_RPT27_VECTOR_ENTRADA"), os.Getenv("VEC_RPT27_VECTOR_SALIDA")
	if input == "" || output == "" {
		t.Skip("ephemeral RPT27 runner only")
	}
	contenido, err := os.ReadFile(input)
	if err != nil || len(contenido) > 2*1024*1024 {
		t.Fatal("RPT27 input unavailable or oversized")
	}
	var e rpt27Entrada
	d := json.NewDecoder(bytes.NewReader(contenido))
	d.DisallowUnknownFields()
	if d.Decode(&e) != nil || d.Decode(new(any)) != io.EOF {
		t.Fatal("RPT27 input invalid")
	}
	if (e.Fase != "preparar" && e.Fase != "emitir") || e.Caso == "" || strings.ContainsAny(e.Caso, "/\\") {
		t.Fatal("RPT27 phase/case invalid")
	}
	if e.ClaveHMACB64 != "" {
		t.Fatal("HMAC must be supplied in a private file")
	}
	salida := rpt27Generar(t, e)
	canon, err := json.Marshal(salida)
	if err != nil {
		t.Fatal("RPT27 output encoding failed")
	}
	rpt27EscribirPrivado(t, output, canon)
}

func rpt27EscribirPrivado(t *testing.T, ruta string, contenido []byte) {
	t.Helper()
	f, err := os.OpenFile(ruta, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal("RPT27 private output unavailable")
	}
	if _, err := f.Write(contenido); err != nil {
		f.Close()
		t.Fatal("RPT27 private output write failed")
	}
	if f.Close() != nil {
		t.Fatal("RPT27 output close failed")
	}
}

func rpt27Generar(t *testing.T, e rpt27Entrada) rpt27Salida {
	t.Helper()
	ahora := rpt27Instante(t, e.Ahora)
	resuelto := rpt27Instante(t, e.ResueltoEn)
	contexto := exigirBase64O205(t, e.ContextoB64)
	manifiesto := exigirBase64O205(t, e.ManifiestoB64)
	material := exigirBase64O205(t, e.MaterialCanonicoB64)
	if len(material) == 0 || len(material) > 128*1024 || !json.Valid(material) {
		t.Fatal("RPT27 material invalid")
	}
	// Canonical PostgreSQL JSONB text is supplied by the parent. It is hashed
	// byte-for-byte; Go must not reserialize it or alter ordering/whitespace.
	var objeto map[string]json.RawMessage
	if json.Unmarshal(material, &objeto) != nil || len(objeto) != 17 {
		t.Fatal("RPT27 material requires exactly 17 keys")
	}
	claves := []string{"esquema", "operacion", "empleado_ref", "relacion_ref", "organismo_ref", "version_esperada", "vigente_en", "conocido_en", "actor_ref", "contexto_actor_ref", "contexto_version", "cuenta_ref", "cuenta_version", "perfil_ref", "perfil_version", "persona_ref", "persona_version"}
	for _, clave := range claves {
		if _, ok := objeto[clave]; !ok {
			t.Fatal("RPT27 material key missing")
		}
	}
	rpt27ExigirCampo(t, objeto, "esquema", "vec.personal.relacion-rpt.consulta.v1")
	rpt27ExigirCampo(t, objeto, "empleado_ref", e.EmpleadoRef)
	suma := sha256.Sum256(material)
	huellaMaterial := fmt.Sprintf("%x", suma)
	var plantilla decisionPlantillaO205
	var motivo motivoCanonicoO205
	if json.Unmarshal(exigirBase64O205(t, e.DecisionPlantillaB64), &plantilla) != nil ||
		json.Unmarshal(exigirBase64O205(t, e.MotivoB64), &motivo) != nil {
		t.Fatal("RPT27 decision/motive invalid")
	}
	if plantilla.Accion != rpt27Accion || plantilla.ModuloID != "personal" || plantilla.TipoRecurso != rpt27Tipo || plantilla.Finalidad != rpt27Finalidad || plantilla.RecursoRef != e.RelacionRef || e.AudienciaConsumo != rpt27Audiencia {
		t.Fatal("RPT27 nominal authority mismatch")
	}
	if e.EmpleadoRef == "" {
		t.Fatal("RPT27 employee reference missing")
	}
	motivoCanon, err := domain.RepresentacionCanonicaMotivoAutorizacionV2(motivo.Referencia)
	if err != nil || !bytes.Equal(motivoCanon, exigirBase64O205(t, e.MotivoB64)) {
		t.Fatal("RPT27 motive is not canonical")
	}
	actor, err := domain.RehidratarContextoActorVinculadoV2(contexto)
	if err != nil {
		t.Fatal("RPT27 actor invalid")
	}
	rpt27ExigirCampo(t, objeto, "actor_ref", actor.Principal.ID)
	rpt27ExigirCampo(t, objeto, "contexto_actor_ref", actor.Instantanea.VinculoRef)
	rpt27ExigirCampo(t, objeto, "cuenta_ref", actor.Instantanea.CuentaRef)
	rpt27ExigirCampo(t, objeto, "cuenta_version", actor.Instantanea.CuentaVersion)
	rpt27ExigirCampo(t, objeto, "perfil_ref", actor.PerfilActivoRef)
	rpt27ExigirCampo(t, objeto, "perfil_version", actor.Instantanea.PerfilVersion)
	rpt27ExigirCampo(t, objeto, "persona_ref", actor.Instantanea.PersonaRef)
	rpt27ExigirCampo(t, objeto, "persona_version", actor.Instantanea.PersonaVersion)
	if actor.Instantanea.PersonaVersion != e.PersonaVersion || actor.Instantanea.PerfilVersion != e.PerfilVersion {
		t.Fatal("RPT27 actor versions mismatch")
	}
	// Concesión nominal de la terna objetivo; no convierte al actor en titular.
	rpt27ExigirCampo(t, objeto, "operacion", "relacion_para_rpt")
	rpt27ExigirCampo(t, objeto, "relacion_ref", e.RelacionRef)
	rpt27ExigirCampo(t, objeto, "organismo_ref", e.OrganismoRef)
	rpt27ExigirCampo(t, objeto, "version_esperada", e.VersionEsperada)
	rpt27ExigirCampo(t, objeto, "vigente_en", e.VigenteEn)
	rpt27ExigirCampo(t, objeto, "conocido_en", e.ConocidoEn)
	resultado := domain.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: plantilla.VinculoAutenticacionActor.RegistroContextoRef,
		Contexto:            actor, RepresentacionCanonica: contexto,
		HuellaSHA256:                      plantilla.VinculoAutenticacionActor.ContextoActorHuellaSHA256,
		ManifiestoProcedenciaCanonico:     manifiesto,
		ManifiestoProcedenciaHuellaSHA256: e.ManifiestoHuellaSHA256,
		AutoridadEfectiva:                 domain.AutoridadProcedenciaContextoActorV1(e.AutoridadEfectiva),
		ResueltoEnAutoritativo:            resuelto,
	}
	if resultado.Validar() != nil {
		t.Fatal("RPT27 registered context invalid")
	}
	vinculo, err := domain.CrearVinculoAutenticacionActorV2(context.Background(),
		revalidadorConfianzaAtestacionV3Prueba{resultado: plantilla.VinculoAutenticacionActor.Autenticacion()},
		domain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: plantilla.VinculoAutenticacionActor.AutenticacionRef, SesionRef: plantilla.VinculoAutenticacionActor.SesionRef},
		resolutorConfianzaAtestacionV3Prueba{resultado: resultado},
		domain.SolicitudContextoActor{Cuenta: domain.CuentaAutenticadaContextoActor{CuentaRef: actor.Instantanea.CuentaRef, Metodo: actor.Principal.AuthMethod, Garantia: actor.Principal.AuthAssurance}, PerfilActivoRef: actor.PerfilActivoRef},
		&relojConfianzaAtestacionV3Prueba{ahora: ahora})
	if err != nil {
		t.Fatal("RPT27 nominal authentication binding failed")
	}
	correlacion, err := domain.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), generadorCorrelacionConfianzaAtestacionV3Prueba{valor: plantilla.CorrelacionRef})
	if err != nil {
		t.Fatal("RPT27 correlation failed")
	}
	solicitud, err := domain.NuevaSolicitudAutorizacionLigadaV3(domain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: vinculo, ReferenciaMotivo: motivo.Referencia,
		Accion: rpt27Accion, Finalidad: rpt27Finalidad, Correlacion: correlacion,
		Recurso: domain.RecursoAutorizable{Referencia: e.RelacionRef, ModuloID: "personal", Tipo: rpt27Tipo,
			Ambitos:   map[string]string{"empleado_ref": e.EmpleadoRef, "organismo_ref": e.OrganismoRef, "relacion_ref": e.RelacionRef},
			Atributos: map[string]string{"conocido_en": e.ConocidoEn, "material_sha256": huellaMaterial, "operacion": rpt27Operacion, "version_esperada": strconv.FormatInt(e.VersionEsperada, 10), "vigente_en": e.VigenteEn}},
	})
	if err != nil {
		t.Fatal("RPT27 nominal request failed")
	}
	id := strings.ReplaceAll(e.Caso, "-", "_")
	rol := domain.VersionRol{
		RolID: "rpt27_go_" + id, Version: 1, Nombre: "RPT27 synthetic nominal fixture", Estado: domain.EstadoVersionRolPublicada,
		Concesiones:  []domain.ConcesionRol{{Accion: rpt27Accion, ModuloID: "personal", TipoRecurso: rpt27Tipo, Finalidades: []string{rpt27Finalidad}, GarantiaMinima: domain.AuthAssuranceHigh, CamposPermitidos: append([]string(nil), rpt27Campos...), Obligaciones: []string{}}},
		PublicadaPor: "autoridad-rpt27-go-sintetica", PublicadaEn: ahora.Add(-10 * time.Minute),
	}
	control := domain.ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 1, Estado: domain.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: rol.PublicadaPor, ActualizadoEn: ahora.Add(-5 * time.Minute)}
	asignacion := domain.AsignacionPerfil{AsignacionID: e.AsignacionID, Version: e.AsignacionVersion, PerfilActivoRef: actor.PerfilActivoRef, PrincipalID: actor.Principal.ID, VersionRolRef: rol.Referencia(), Estado: domain.EstadoAsignacionPerfilActiva,
		Ambitos: []domain.AmbitoPerfil{{Clave: "empleado_ref", Valores: []string{e.EmpleadoRef}}, {Clave: "organismo_ref", Valores: []string{e.OrganismoRef}}, {Clave: "relacion_ref", Valores: []string{e.RelacionRef}}}, EmitidaPor: rol.PublicadaPor, EmitidaEn: ahora.Add(-10 * time.Minute), VigenteDesde: ahora.Add(-5 * time.Minute), VigenteHasta: ahora.Add(30 * time.Minute)}
	huellaCatalogo, err := domain.HuellaCatalogoPoliticasAutorizacion(e.Politicas)
	if err != nil || huellaCatalogo != e.HuellaCatalogoSHA256 || e.RevisionCatalogo == 0 {
		t.Fatal("RPT27 policy catalog mismatch")
	}
	instantanea := domain.InstantaneaAutorizacion{AsignacionPerfil: asignacion, VersionRol: rol, ControlVigenciaVersionRol: control, Politicas: e.Politicas, RevisionCatalogoPoliticas: e.RevisionCatalogo, CatalogoPoliticasHuellaSHA256: e.HuellaCatalogoSHA256}
	evidencia, err := domain.NuevaEvidenciaEvaluacionAutorizacionV3(solicitud, instantanea, plantilla.DecisionRef, ahora, ahora.Add(90*time.Second))
	if err != nil {
		t.Fatal("RPT27 nominal evaluation failed")
	}
	decision, err := domain.NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
	if err != nil {
		t.Fatal("RPT27 nominal decision failed")
	}
	concedida, codigo, err := decision.Resultado()
	if err != nil || !concedida || codigo != "concedida" {
		t.Fatal("RPT27 policy fixture denied")
	}
	decisionCanon, err := domain.RepresentacionCanonicaDecisionAutorizacionV3(decision)
	if err != nil {
		t.Fatal("RPT27 decision serialization failed")
	}
	huellaDecision, err := domain.HuellaSHA256DecisionAutorizacionV3(decision)
	if err != nil {
		t.Fatal("RPT27 decision hash failed")
	}
	salida := rpt27Salida{Fase: e.Fase, MaterialCanonicoB64: e.MaterialCanonicoB64, MaterialHuellaSHA256: huellaMaterial, DecisionHuellaSHA256: huellaDecision, Campos: append([]string(nil), rpt27Campos...), Obligaciones: []string{}, Politicas: e.Politicas, RevisionCatalogo: e.RevisionCatalogo, HuellaCatalogoSHA256: e.HuellaCatalogoSHA256, PDP: "fixture_nominal_sintetica_sin_proveedor_operativo"}
	salida.DecisionB64 = base64.StdEncoding.EncodeToString(decisionCanon)
	salida.MotivoB64 = base64.StdEncoding.EncodeToString(motivoCanon)
	salida.ContextoB64 = e.ContextoB64
	salida.PersonaVersion = e.PersonaVersion
	salida.PerfilVersion = e.PerfilVersion
	salida.VersionRolDocumento = exigirJSONO205(t, rol)
	salida.ControlRolDocumento = exigirJSONO205(t, control)
	salida.AsignacionDocumento = exigirJSONO205(t, asignacion)
	if e.Fase == "preparar" {
		if e.Confirmacion != nil {
			t.Fatal("RPT27 preparation must not import a receipt")
		}
		return salida
	}
	if e.Confirmacion == nil {
		t.Fatal("RPT27 emission requires SQL receipt")
	}
	registradaEn := rpt27Instante(t, e.Confirmacion.RegistradaEn)
	if registradaEn.Before(ahora) {
		t.Fatal("RPT27 registration precedes decision")
	}
	instanteCripto := time.Now().UTC().Truncate(time.Microsecond)
	if instanteCripto.Before(registradaEn) {
		instanteCripto = registradaEn
	}
	puente := &rpt27PDP{t: t, decision: decision, confirmacion: *e.Confirmacion}
	publica, privada, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal("RPT27 ephemeral root generation failed")
	}
	defer borrarBytesConfianzaAtestacion(privada)
	raiz, err := NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(e.RaizClaveID, e.RaizVersion, publica, e.AudienciaDespliegue, EstadoClaveAtestacionAutorizacionV3Activa, ahora.Add(-10*time.Minute), ahora.Add(time.Hour), time.Time{})
	if err != nil {
		t.Fatal("RPT27 ephemeral root invalid")
	}
	config, err := NuevaConfiguracionConfianzaAtestacionAutorizacionV3(e.RevisionConfianza, e.SecuenciaConfianza, ahora.Add(-5*time.Minute), ahora.Add(30*time.Minute), raiz)
	if err != nil {
		t.Fatal("RPT27 trust fixture invalid")
	}
	reloj := &relojConfianzaAtestacionV3Prueba{ahora: instanteCripto}
	confianza, err := NuevoServicioConfianzaAtestacionAutorizacionV3(config, reloj)
	if err != nil {
		t.Fatal("RPT27 trust service failed")
	}
	cabecera := domain.CabeceraAtestacionAutorizacionV3{FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV3, Suite: SuiteAtestacionAutorizacionV3COSEEdDSA, ClaveID: e.RaizClaveID, Audiencia: e.AudienciaDespliegue}
	firmante := rpt27Firmante{t: t, privada: privada, cabecera: cabecera, ahora: instanteCripto}
	atestador, err := application.NuevoServicioAtestacionesAutorizacionV3(cabecera, firmante)
	if err != nil {
		t.Fatal("RPT27 attestation service failed")
	}
	claveBytes := rpt27LeerHMAC(t, e.ClaveHMACArchivo)
	defer borrarBytesConfianzaAtestacion(claveBytes)
	clave, err := NuevaClaveHMACCapacidadAtestacionAutorizacionV3(e.ClaveID, e.ClaveVersion, claveBytes, e.EmisorID, rpt27Audiencia, EstadoClaveHMACCapacidadAtestacionV3Emision, rpt27Instante(t, e.ClaveValidaDesde), rpt27Instante(t, e.ClaveValidaHasta), time.Time{}, e.RevisionGobierno, e.HuellaGobiernoSHA256)
	if err != nil {
		t.Fatal("RPT27 ephemeral HMAC invalid")
	}
	capacidades, err := NuevoEmisorCapacidadesAtestacionAutorizacionV3(clave, reloj)
	if err != nil {
		t.Fatal("RPT27 capability emitter failed")
	}
	emisor, err := NuevoEmisorMaterialAutorizacionAtestadaV3(puente, atestador, confianza, capacidades)
	if err != nil {
		t.Fatal("RPT27 nominal material emitter failed")
	}
	_, confirmacion, exportador, err := emisor.EmitirMaterialAutorizacionAtestadaV3(context.Background(), solicitud, resultado)
	if err != nil {
		t.Fatal("RPT27 public nominal emission failed")
	}
	exportacion, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil {
		t.Fatal("RPT27 public nominal material export failed")
	}
	cd, err := confirmacion.Datos()
	if err != nil {
		t.Fatal("RPT27 confirmation invalid")
	}
	salida.Confirmacion = &rpt27Confirmacion{DecisionRef: cd.DecisionRef, DecisionHuellaSHA256: cd.DecisionHuellaSHA256, EmitidaEn: cd.EmitidaEn.Format(rpt27FormatoTiempo), ValidaHasta: cd.ValidaHasta.Format(rpt27FormatoTiempo), RegistradaEn: cd.RegistradaEn.Format(rpt27FormatoTiempo)}
	salida.CapacidadB64 = base64.StdEncoding.EncodeToString(exportacion.CapacidadCanonica())
	salida.DecisionB64 = base64.StdEncoding.EncodeToString(exportacion.DecisionCanonica())
	salida.MotivoB64 = base64.StdEncoding.EncodeToString(exportacion.MotivoCanonico())
	salida.ContextoB64 = base64.StdEncoding.EncodeToString(exportacion.ContextoActorCanonico())
	salida.PayloadB64 = base64.StdEncoding.EncodeToString(exportacion.PayloadVECAD3())
	salida.COSEB64 = base64.StdEncoding.EncodeToString(exportacion.SobreCOSESign1())
	salida.EvidenciaB64 = base64.StdEncoding.EncodeToString(exportacion.EvidenciaVerificacion())
	salida.SPKIB64 = base64.StdEncoding.EncodeToString(exportacion.RaizPublicaSPKI())
	return salida
}

func rpt27ExigirCampo(t *testing.T, objeto map[string]json.RawMessage, clave string, esperado any) {
	t.Helper()
	canon, err := json.Marshal(esperado)
	if err != nil || !bytes.Equal(objeto[clave], canon) {
		t.Fatal("RPT27 material context mismatch")
	}
}
func rpt27ParsearInstante(valor string) (time.Time, error) {
	instante, err := time.Parse(time.RFC3339Nano, valor)
	if err != nil || instante.IsZero() || instante.Year() < 1 || instante.Year() > 9999 || instante.Nanosecond()%1000 != 0 {
		return time.Time{}, errors.New("RPT27 timestamp invalid")
	}
	_, desplazamiento := instante.Zone()
	if desplazamiento != 0 {
		return time.Time{}, errors.New("RPT27 timestamp must be UTC")
	}
	return instante.UTC(), nil
}
func rpt27Instante(t *testing.T, valor string) time.Time {
	t.Helper()
	instante, err := rpt27ParsearInstante(valor)
	if err != nil {
		t.Fatal("RPT27 timestamp invalid")
	}
	return instante
}

func rpt27LeerHMAC(t *testing.T, ruta string) []byte {
	t.Helper()
	info, err := os.Lstat(ruta)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Size() != 32 {
		t.Fatal("RPT27 HMAC file must be regular, 0600 and 32 raw bytes")
	}
	f, err := os.Open(ruta)
	if err != nil {
		t.Fatal("RPT27 HMAC file unavailable")
	}
	defer f.Close()
	actual, err := f.Stat()
	if err != nil || !os.SameFile(info, actual) {
		t.Fatal("RPT27 HMAC file changed")
	}
	clave, err := io.ReadAll(io.LimitReader(f, 33))
	if err != nil || len(clave) != 32 {
		t.Fatal("RPT27 HMAC file invalid")
	}
	return clave
}

type rpt27PDP struct {
	t            *testing.T
	decision     domain.DecisionAutorizacionLigadaV3
	confirmacion rpt27Confirmacion
}

func (p *rpt27PDP) ExigirSolicitudLigadaV3(ctx context.Context, s domain.SolicitudAutorizacionLigadaV3, r domain.ResultadoContextoActorRegistradoV2) (domain.DecisionAutorizacionLigadaV3, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	d, err := s.Datos()
	if err != nil {
		return domain.DecisionAutorizacionLigadaV3{}, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, err
	}
	orden, err := ports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(s, p.decision, d.ReferenciaMotivo, r)
	if err != nil {
		return domain.DecisionAutorizacionLigadaV3{}, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, err
	}
	confirmada, err := ports.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx, p, orden)
	return p.decision, confirmada, err
}
func (p *rpt27PDP) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(_ context.Context, orden ports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3) (time.Time, error) {
	d, err := orden.Datos()
	if err != nil {
		return time.Time{}, err
	}
	canon, err := domain.RepresentacionCanonicaDecisionAutorizacionV3(d.Decision)
	if err != nil {
		return time.Time{}, err
	}
	var esperada rpt27Confirmacion
	if json.Unmarshal(canon, &esperada) != nil {
		return time.Time{}, errors.New("RPT27 decision invalid")
	}
	esperada.DecisionHuellaSHA256, err = domain.HuellaSHA256DecisionAutorizacionV3(d.Decision)
	if err != nil {
		return time.Time{}, err
	}
	// Only the registered timestamp comes from SQL; identity, decision hash
	// and validity must match the prepared nominal exactly.
	if esperada.DecisionRef != p.confirmacion.DecisionRef || esperada.DecisionHuellaSHA256 != p.confirmacion.DecisionHuellaSHA256 {
		return time.Time{}, errors.New("RPT27 durable receipt mismatch")
	}
	for _, par := range [][2]string{{esperada.EmitidaEn, p.confirmacion.EmitidaEn}, {esperada.ValidaHasta, p.confirmacion.ValidaHasta}} {
		a, ea := rpt27ParsearInstante(par[0])
		b, eb := rpt27ParsearInstante(par[1])
		if ea != nil || eb != nil || !a.Equal(b) {
			return time.Time{}, errors.New("RPT27 durable receipt time mismatch")
		}
	}
	return rpt27ParsearInstante(p.confirmacion.RegistradaEn)
}

type rpt27Firmante struct {
	t        *testing.T
	privada  ed25519.PrivateKey
	cabecera domain.CabeceraAtestacionAutorizacionV3
	ahora    time.Time
}

func (f rpt27Firmante) FirmarAtestacionAutorizacionV3(ctx context.Context, solicitud ports.SolicitudFirmaAtestacionAutorizacionV3) (ports.ResultadoFirmaAtestacionAutorizacionV3, error) {
	if ctx.Err() != nil {
		return ports.ResultadoFirmaAtestacionAutorizacionV3{}, ctx.Err()
	}
	payload, err := solicitud.Mensaje()
	if err != nil {
		return ports.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	aad, err := AADExternoAtestacionAutorizacionV3(f.cabecera.Audiencia)
	if err != nil {
		return ports.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	sobre := firmarSobreConfianzaAtestacionV3Prueba(f.t, f.privada, []byte(f.cabecera.ClaveID), payload, aad)
	return ports.NuevoResultadoFirmaAtestacionAutorizacionV3(solicitud, sobre, "evidencia:rpt27:fixture:crypto-real", f.ahora)
}


// This fixture transports the native material already issued and registered by
// the runner. PostgreSQL still validates COSE/HMAC and live authority.
type rpt27AutorizadorPG struct { material ports.ExportacionMaterialConsumoAutorizacionAtestadaV3 }
func (a rpt27AutorizadorPG) AutorizarRelacionParaRPT(context.Context, personaldomain.MaterialLectorRelacionRPT) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) { return a.material, nil }

// La frontera fixture devuelve el contexto y vínculo originales ya registrados.
// No resuelve otra identidad al confirmar el intento después del rollback.
type rpt27IdentidadOriginal struct { valor personalcomposicion.IdentidadRegistradaLectorRelacionRPT }
func (i rpt27IdentidadOriginal) ResolverIdentidadLectorRelacionRPT(context.Context) (personalcomposicion.IdentidadRegistradaLectorRelacionRPT,error) { return i.valor,nil }

// Envuelve el puerto común real para cotejar su acuse, sin sustituirlo.
type rpt27RegistradorObservado struct { comun *vecpg.RegistradorIntentosAuditoriaPostgreSQL; orden ports.OrdenIntentoAuditoria; confirmados int }
func (r *rpt27RegistradorObservado) PreflightIntentoAuditoria(ctx context.Context) error { return r.comun.PreflightIntentoAuditoria(ctx) }
func (r *rpt27RegistradorObservado) AppendIntentoAuditoria(ctx context.Context,o ports.OrdenIntentoAuditoria) (ports.AcuseIntentoAuditoria,error) {
 acuse,err:=r.comun.AppendIntentoAuditoria(ctx,o)
 if err==nil && acuse.ValidarPara(o)==nil { r.orden=o;r.confirmados++ }
 return acuse,err
}

type rpt27DestinoTecnicoAcotado struct { datos bytes.Buffer; fallar bool }
func (d *rpt27DestinoTecnicoAcotado) Write(p []byte) (int,error) {
 if d.fallar || d.datos.Len()+len(p)>4096 { return 0,errors.New("RPT27 technical sink unavailable") }
 return d.datos.Write(p)
}

func TestLectorRPT27ConPoolsPostgreSQL(t *testing.T) {
	modo := os.Getenv("VEC_RPT27_GO_MODO")
	if modo == "" { t.Skip("private PostgreSQL runner only") }
	if modo != "positivo" && modo != "sink_caido" && modo != "replay" && modo != "cruzados" && modo != "registro_caido" && modo != "revocada" && modo != "concurrente" { t.Fatal("RPT27 unknown test mode") }
	ctx,cancel := context.WithTimeout(context.Background(),20*time.Second); defer cancel()
	pool := func(usuario string) *pgxpool.Pool {
		c,err := pgxpool.ParseConfig("host=/pgsocket dbname=postgres sslmode=disable")
		if err != nil { t.Fatal("RPT27 pool configuration unavailable") }
		c.ConnConfig.User=usuario; c.MaxConns=1; c.MinConns=0
		c.ConnConfig.RuntimeParams["timezone"]="UTC"
		c.ConnConfig.RuntimeParams["statement_timeout"]="15000"
		v,err := pgxpool.NewWithConfig(ctx,c)
		if err != nil { t.Fatal("RPT27 pool unavailable") }
		t.Cleanup(v.Close)
		var actual string
		if v.QueryRow(ctx,"SELECT session_user").Scan(&actual)!=nil || actual!=usuario { t.Fatal("RPT27 nominal SQL identity mismatch") }
		return v
	}
	lectura := pool("vec_rpt27_ensayo_runtime")
	registro := pool("vec_rpt27_ensayo_registrador")
	if lectura==registro { t.Fatal("RPT27 pools must be segregated") }
	buf,err := os.ReadFile(os.Getenv("VEC_RPT27_VECTOR_SALIDA"))
	if err!=nil || len(buf)>2<<20 { t.Fatal("RPT27 exported material unavailable") }
	var salida rpt27Salida
	if json.Unmarshal(buf,&salida)!=nil { t.Fatal("RPT27 exported material invalid") }
	capacidad := exigirBase64O205(t,salida.CapacidadB64)
	var cap capacidadAtestacionAutorizacionV3JSON
	if json.Unmarshal(capacidad,&cap)!=nil || cap.validarEstructura()!=nil { t.Fatal("RPT27 capability invalid") }
	resumen,err := ports.NuevoResumenCapacidadAtestacionAutorizacionV3(cap.DecisionRef,cap.HuellaDecisionSHA256,cap.HuellaMotivoSHA256,cap.ContextoRef,cap.HuellaContextoSHA256,cap.Operacion,cap.EfectoRef,cap.HuellaEfectoSHA256,cap.AudienciaConsumo,rpt27Instante(t,cap.EmitidaEn),rpt27Instante(t,cap.ExpiraEn))
	if err!=nil { t.Fatal("RPT27 summary invalid") }
	a,err := ports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(capacidad,resumen,exigirBase64O205(t,salida.DecisionB64),exigirBase64O205(t,salida.MotivoB64),exigirBase64O205(t,salida.ContextoB64),salida.PersonaVersion,salida.PerfilVersion,exigirBase64O205(t,salida.PayloadB64),exigirBase64O205(t,salida.COSEB64),exigirBase64O205(t,salida.EvidenciaB64),exigirBase64O205(t,salida.SPKIB64))
	if err!=nil { t.Fatal("RPT27 native transport invalid") }
	actor,err := domain.RehidratarContextoActorVinculadoV2(a.ContextoActorCanonico())
	if err!=nil { t.Fatal("RPT27 registered actor invalid") }
	var material struct {
		EmpleadoRef string `json:"empleado_ref"`
		RelacionRef string `json:"relacion_ref"`
		OrganismoRef string `json:"organismo_ref"`
		Version int64 `json:"version_esperada"`
		VigenteEn string `json:"vigente_en"`
		ConocidoEn string `json:"conocido_en"`
	}
	if json.Unmarshal(exigirBase64O205(t,salida.MaterialCanonicoB64),&material)!=nil { t.Fatal("RPT27 selector invalid") }
	r,err := personalpg.NuevoRepositorioLectorRelacionRPTPostgreSQL(lectura)
	if err!=nil { t.Fatal("RPT27 reader unavailable") }
	destino := registro
	if modo=="cruzados" { destino=lectura }
	comun,err := vecpg.NuevoRegistradorIntentosAuditoriaPostgreSQL(destino,"rpt27-ensayo","interna_corporativa",5*time.Second)
	if err!=nil { t.Fatal("RPT27 common registry unavailable") }
	entradaBytes,err := os.ReadFile(os.Getenv("VEC_RPT27_VECTOR_ENTRADA"))
	var entrada rpt27Entrada
	if err!=nil || json.Unmarshal(entradaBytes,&entrada)!=nil { t.Fatal("RPT27 original identity fixture unavailable") }
	var plantilla decisionPlantillaO205
	var motivo motivoCanonicoO205
	if json.Unmarshal(exigirBase64O205(t,entrada.DecisionPlantillaB64),&plantilla)!=nil || json.Unmarshal(exigirBase64O205(t,entrada.MotivoB64),&motivo)!=nil { t.Fatal("RPT27 original binding fixture invalid") }
	resultado := domain.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef:plantilla.VinculoAutenticacionActor.RegistroContextoRef,
		Contexto:actor,RepresentacionCanonica:a.ContextoActorCanonico(),
		HuellaSHA256:plantilla.VinculoAutenticacionActor.ContextoActorHuellaSHA256,
		ManifiestoProcedenciaCanonico:exigirBase64O205(t,entrada.ManifiestoB64),
		ManifiestoProcedenciaHuellaSHA256:entrada.ManifiestoHuellaSHA256,
		AutoridadEfectiva:domain.AutoridadProcedenciaContextoActorV1(entrada.AutoridadEfectiva),
		ResueltoEnAutoritativo:rpt27Instante(t,entrada.ResueltoEn),
	}
	vinculo,err := domain.CrearVinculoAutenticacionActorV2(ctx,
		revalidadorConfianzaAtestacionV3Prueba{resultado:plantilla.VinculoAutenticacionActor.Autenticacion()},
		domain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef:plantilla.VinculoAutenticacionActor.AutenticacionRef,SesionRef:plantilla.VinculoAutenticacionActor.SesionRef},
		resolutorConfianzaAtestacionV3Prueba{resultado:resultado},
		domain.SolicitudContextoActor{Cuenta:domain.CuentaAutenticadaContextoActor{CuentaRef:actor.Instantanea.CuentaRef,Metodo:actor.Principal.AuthMethod,Garantia:actor.Principal.AuthAssurance},PerfilActivoRef:actor.PerfilActivoRef},
		&relojConfianzaAtestacionV3Prueba{ahora:rpt27Instante(t,entrada.Ahora)})
	if err!=nil { t.Fatal("RPT27 original binding invalid") }
	registroObservado:=&rpt27RegistradorObservado{comun:comun}
	i,err := personalcomposicion.NuevoRegistroIntentosLectorRelacionRPT(registroObservado,personalcomposicion.ConfiguracionIntentosLectorRPT{
		Proceso:"rpt27-ensayo",Canal:"interna_corporativa",RecursoEntradaInvalida:"personal:relacion-rpt:entrada",
		MotivoDenegado:motivo.Referencia,MotivoEntradaInvalida:motivo.Referencia,MotivoNoDisponible:motivo.Referencia})
	if err!=nil { t.Fatal("RPT27 nominal registry unavailable") }
	servicio,err := personalapp.NuevoServicioLectorRelacionRPT(rpt27AutorizadorPG{a},r,i,time.Now)
	if err!=nil { t.Fatal("RPT27 service unavailable") }
	base,err := personalrpt.NuevoLectorRelacionSeleccionadaRPT(servicio,i)
	if err!=nil { t.Fatal("RPT27 selector unavailable") }
	destinoTecnico:=&rpt27DestinoTecnicoAcotado{fallar:modo=="sink_caido"}
	emisorTecnico,err:=observabilidad.NuevoEmisorJSONLines(observabilidad.OpcionesEmisor{Destino:destinoTecnico,Capacidad:8,Entorno:"pruebas",VersionBinario:"5c5e305a3"})
	if err!=nil { t.Fatal("RPT27 technical emitter unavailable") }
	t.Cleanup(func(){ c,cancelar:=context.WithTimeout(context.Background(),time.Second);defer cancelar();_ = emisorTecnico.Cerrar(c) })
	frontera,err := personalcomposicion.NuevoLectorRelacionSeleccionadaRPTConIdentidad(base,rpt27IdentidadOriginal{personalcomposicion.IdentidadRegistradaLectorRelacionRPT{Vinculo:vinculo,Resultado:resultado}},5*time.Second,emisorTecnico,personalcomposicion.ConfiguracionResultadosTecnicosLectorRPT{Componente:domain.ComponenteIncidenciaComposicion,Etapa:domain.EtapaIncidenciaConsulta})
	if err!=nil { t.Fatal("RPT27 trusted capture unavailable") }
	ctx,err=ports.ConCorrelacionIncidenciasPeticion(ctx)
	if err!=nil { t.Fatal("RPT27 trusted correlation unavailable") }
	corte:=personaldomain.CorteEmpleadoB2{VigenteEn:personaldomain.FechaCivil(material.VigenteEn),ConocidoEn:rpt27Instante(t,material.ConocidoEn)}
	preparacion:=personaldomain.PreparacionRelacionParaRPT{Esquema:"vec.personal.preparacion-relacion-rpt.v1",Uso:"preparacion",Cobertura:"no_acreditada",EstadoRPT:"pendiente_fuente_rpt",EmpleadoRef:material.EmpleadoRef,Corte:corte,
		Relaciones:[]personaldomain.RelacionPreparacionParaRPT{{RelacionRef:material.RelacionRef,Traza:personaldomain.TrazaEmpleadoB2{Desde:"2026-01-01",RegistradaEn:corte.ConocidoEn,Version:material.Version,ActoRef:"acto:rpt27:sintetico",FuenteRef:"fuente:rpt27:sintetica",FuenteVersion:1}}}}
	v,err := frontera.ConsultarSeleccionada(ctx,actor,preparacion,material.OrganismoRef,material.RelacionRef)
	cierreCtx,cierreCancel:=context.WithTimeout(context.Background(),time.Second)
	defer cierreCancel()
	if emisorTecnico.Cerrar(cierreCtx)!=nil { t.Fatal("RPT27 technical queue did not drain") }
	metricas:=emisorTecnico.MetricasResultadosTecnicos()
	if metricas.Aceptados!=1 || metricas.Descartados!=0 || metricas.Invalidos!=0 || metricas.SinCorrelacion!=0 { t.Fatal("RPT27 technical result lost or duplicated") }
	if modo=="sink_caido" {
		if metricas.FallosEscritura!=1 || metricas.Escritos!=0 || destinoTecnico.datos.Len()!=0 { t.Fatal("RPT27 sink failure was not counted") }
	} else {
		var linea map[string]any
		if metricas.Escritos!=1 || metricas.FallosEscritura!=0 || json.Unmarshal(bytes.TrimSpace(destinoTecnico.datos.Bytes()),&linea)!=nil || len(linea)!=10 { t.Fatal("RPT27 technical JSONL missing or invalid") }
		correlacion,ok:=ports.CorrelacionIncidenciasPeticion(ctx)
		esperadoTecnico:="denegado"
		if modo=="positivo" { esperadoTecnico="correcto" }
		if modo=="cruzados" || modo=="registro_caido" || modo=="concurrente" { esperadoTecnico="no_disponible" }
		if !ok || linea["esquema"]!=domain.EsquemaResultadoTecnico || linea["correlacion"]!=correlacion || linea["correlacion_ref"]!="correlacion_"+correlacion || linea["resultado"]!=esperadoTecnico { t.Fatal("RPT27 technical result or correlation diverged") }
	}
	if modo=="replay" || modo=="revocada" || modo=="concurrente" {
		datos,errOrden:=registroObservado.orden.Datos()
		correlacion,_:=ports.CorrelacionIncidenciasPeticion(ctx)
		if registroObservado.confirmados!=1 || errOrden!=nil || datos.Datos.CorrelacionRef!="correlacion_"+correlacion || datos.ResultadoContexto.HuellaSHA256!=resultado.HuellaSHA256 { t.Fatal("RPT27 nominal receipt and technical correlation diverged") }
	} else if registroObservado.confirmados!=0 { t.Fatal("RPT27 unexpected nominal attempt") }
	if modo=="positivo" || modo=="sink_caido" {
		if err!=nil || v.Relacion.RelacionRef!=material.RelacionRef || v.Relacion.Version!=material.Version || v.Cobertura!=personalports.CoberturaPersonalNoAcreditadaV1 || v.Relacion.Procedencia.Certeza!=personalports.CertezaPersonalNoAcreditadaV1 || v.Evidencia.ReciboRef=="" { t.Fatal("RPT27 native positive did not produce validated receipt") }
		return
	}
	esperado := personaldomain.ErrLectorRelacionRPTDenegado
	if modo=="cruzados" || modo=="registro_caido" || modo=="concurrente" { esperado=personaldomain.ErrLectorRelacionRPTNoDisponible }
	if !errors.Is(err,esperado) || v.Relacion.RelacionRef!="" || v.Evidencia.ReciboRef!="" { t.Fatal("RPT27 native negative returned data or wrong nominal error") }
}

'''
output = scratch / 'capacidad_v3_vector_sql_test.go'
with output.open('x') as f:
    os.chmod(output, 0o600)
    f.write(original + extra)
with (scratch / 'overlay.json').open('x') as f:
    os.chmod(scratch / 'overlay.json', 0o600)
    json.dump({'Replace': {str(source): str(output)}}, f)
print('overlay prepared')
RPT27_GENERADOR_PY
python3 "$helper" "$repo_dir" "$scratch" > "$scratch/generador.log" 2>&1 || fallo 'generador falló'
python3 - "$scratch/overlay.json" "$repo_dir" "$scratch" <<'PY'
import json,pathlib,sys
p=pathlib.Path(sys.argv[1]); o=json.loads(p.read_text()); o['Replace']={k.replace(sys.argv[2],'/src',1):v.replace(sys.argv[3],'/scratch',1) for k,v in o['Replace'].items()};p.write_text(json.dumps(o))
PY
fi
toolchain=/modcache/golang.org/toolchain@v0.0.1-go1.26.6.linux-amd64
sandbox() {
 systemd-run --user --quiet --wait --pipe --collect -p MemoryMax=2G -p TasksMax=256 -p CPUQuota=200% -p LimitFSIZE=268435456 -p LimitNOFILE=256 \
 /usr/bin/env -i PATH=/usr/bin:/bin /usr/bin/timeout 1800 /usr/bin/bwrap --unshare-all --die-with-parent \
 --ro-bind /usr /usr --ro-bind /bin /bin --ro-bind /lib /lib --ro-bind /lib64 /lib64 --proc /proc --dev /dev --tmpfs /tmp \
 --ro-bind "$repo_dir" /src --ro-bind "$modcache" /modcache --bind "$scratch" /scratch --bind "$cache" /buildcache --ro-bind "$socket" /pgsocket --chdir /src \
 /usr/bin/env -i PATH="$toolchain/bin:/usr/bin:/bin" HOME=/scratch GOROOT="$toolchain" GOTOOLCHAIN=local GOPATH=/scratch/gopath GOMODCACHE=/modcache GOCACHE=/buildcache GOPROXY=off GOSUMDB=off CGO_ENABLED=0 GOMAXPROCS=2 \
 VEC_RPT27_VECTOR_ENTRADA=/scratch/entrada.json VEC_RPT27_VECTOR_SALIDA=/scratch/salida.json VEC_RPT27_GO_MODO="${go_modo:-}" "$@"
}
if { ! "$continuar" || "$desde_checkpoint" || "$desde_go"; } && ! "$solo_concurrente"; then
sandbox "$toolchain/bin/go" test -c -p 8 -overlay /scratch/overlay.json -o /scratch/rpt27.test ./internal/vec/adapters/seguridad/confianzaatestacion > "$scratch/compilar.log" 2>&1 || fallo 'compilación focal aislada falló'
elif ! "$desde_checkpoint" && ! "$solo_concurrente"; then
 # Únicamente las dos funciones auxiliares propias, nunca las migraciones.
 python3 - "$base_dir/relacion_para_rpt_000027.sql" "$scratch/fixture_corregida.sql" <<'PYFIXTURE'
import pathlib,re,sys
s=pathlib.Path(sys.argv[1]).read_text();partes=[]
for n in ('rpt27_ensayo_fuente','rpt27_ensayo_revision'):
 m=re.search(r'CREATE FUNCTION public\.'+n+r'\(.*?\$f\$;',s,re.S)
 assert m is not None
 partes.append(m[0].replace('CREATE FUNCTION','CREATE OR REPLACE FUNCTION',1))
pathlib.Path(sys.argv[2]).write_text(chr(92)+'set ON_ERROR_STOP on\nBEGIN;\n'+'\n'.join(partes)+'\nCOMMIT;\n')
PYFIXTURE
 archivo "$scratch/fixture_corregida.sql"
 captura "$scratch/reanudacion.json"
 cmp -s "$scratch/preimagen.json" "$scratch/reanudacion.json" || fallo 'reanudación alteró autoridades previas'
fi

normalizar() { # entrada exportadaSQL; fase; confirmación opcional
 python3 - "$scratch" "$1" "${2:-}" "$hmac_nombre" <<'PY'
import base64,datetime,json,pathlib,sys
s=pathlib.Path(sys.argv[1]);p=s/'entrada.json';j=json.loads(p.read_text())
def tiempo(x):
    return datetime.datetime.fromisoformat(x.replace('Z','+00:00')).astimezone(datetime.timezone.utc).isoformat(timespec='microseconds').replace('+00:00','Z')
for k in ('ahora','resuelto_en','clave_valida_desde','clave_valida_hasta'):j[k]=tiempo(j[k])
t=json.loads(base64.b64decode(j['decision_plantilla_b64']));v=t['vinculo_autenticacion_actor']
for k in ('autenticacion_verificada_en','sesion_emitida_en','sesion_valida_hasta','sesion_revalidada_en'):v[k]=tiempo(v[k])
j['decision_plantilla_b64']=base64.b64encode(json.dumps(t,separators=(',',':')).encode()).decode()
j['fase']=sys.argv[2];j['clave_hmac_archivo']='/scratch/'+sys.argv[4]
if sys.argv[3]:
    c=json.loads((s/'confirmacion.json').read_text());c['registrada_en']=tiempo(c['registrada_en']);j['confirmacion']=c
p.write_text(json.dumps(j,separators=(',',':')))
PY
}
importar() { # función de ensayo; caso
 python3 - "$scratch" "$1" "$2" <<'PY'
import pathlib,sys
s=pathlib.Path(sys.argv[1]);b=(s/'salida.json').read_text()
assert '$rpt27bundle$' not in b
(s/'importar.sql').write_text("BEGIN ISOLATION LEVEL SERIALIZABLE; SET LOCAL timezone='UTC'; SELECT public."+sys.argv[2]+"('"+sys.argv[3]+"',$rpt27bundle$"+b+"$rpt27bundle$::jsonb); COMMIT;\n")
PY
 psql_run postgres < "$scratch/importar.sql" > "$scratch/confirmacion.json" 2> "$scratch/importar.log" || fallo 'importación/registro durable falló'
}
preparar() {
 local caso=$1 familia=${2:-$1} estado=${3:-vigente} s
 s=$(printf '%s' "$caso" | sha256sum | cut -c1-32)
 valor "SELECT public.rpt27_ensayo_actor('$caso'); SELECT public.rpt27_ensayo_fuente('$caso','$familia','$estado');" > /dev/null
 psql_run vec_rpt27_ensayo_ca -c "BEGIN ISOLATION LEVEL SERIALIZABLE;
 SELECT count(*) FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2('oca_rpt27_$s','rca_rpt27_$s','cta_rpt27_$s','prf_rpt27_$s','certificado','alto',clock_timestamp(),ARRAY['empleado']); COMMIT;" > "$scratch/contexto.log" 2>&1 || fallo 'registroCA7 nominal falló'
 if [[ -n ${corte_fijado:-} ]]; then valor "UPDATE public.rpt27_ensayo_vector SET conocido_en='$corte_fijado'::timestamptz WHERE caso='$caso'" > /dev/null; fi
 valor "SELECT public.rpt27_ensayo_entrada('$caso')" > "$scratch/entrada.json"
 normalizar preparar
 python3 - "$scratch/salida.json" <<'PYSALIDA'
import pathlib,sys
p=pathlib.Path(sys.argv[1])
if p.is_file() and not p.is_symlink():p.unlink()
PYSALIDA
 sandbox /scratch/rpt27.test -test.run '^TestGenerarVectorRPT27ParaSQL$' -test.count=1 > "$scratch/preparar.log" 2>&1 || fallo 'preparaciónGo nominal falló'
 importar rpt27_ensayo_preparado "$caso"
 normalizar emitir confirmada
 python3 - "$scratch/salida.json" <<'PYSALIDA'
import pathlib,sys
p=pathlib.Path(sys.argv[1])
if p.is_file() and not p.is_symlink():p.unlink()
PYSALIDA
 sandbox /scratch/rpt27.test -test.run '^TestGenerarVectorRPT27ParaSQL$' -test.count=1 > "$scratch/emitir.log" 2>&1 || fallo 'emisiónCOSE/HMAC real falló'
 importar rpt27_ensayo_firmado "$caso"
}

preflight_registro() {
 # Catálogo solamente: no SELECT laboral en el pool exclusivo de escritura.
 [[ $(psql_run vec_rpt27_ensayo_registrador -c "SELECT vec_autorizacion_atestada_v3.preflight_registrador_intentos_v1('rpt27-ensayo','interna_corporativa')" 2> "$scratch/preflight.log") == t ]]
}
consultar() {
 preflight_registro || { printf '%s\n' 'no_disponible: registrador obligatorio ausente' >&2; return 75; }
 psql_run vec_rpt27_ensayo_runtime -c "BEGIN ISOLATION LEVEL SERIALIZABLE; SET LOCAL timezone='UTC'; SET LOCAL statement_timeout='15s'; SET LOCAL idle_in_transaction_session_timeout='20s'; SELECT public.rpt27_ensayo_consultar('$1','${2:-valida}'); COMMIT;"
}
registrar_intento() { # caso, motivo; evidencia ORIGINAL histórica acreditada
 local caso=$1 motivo=$2
 valor "SELECT jsonb_build_object('contexto_b64',bundle->>'contexto_b64',
 'vinculo',convert_from(decode(bundle->>'decision_b64','base64'),'UTF8')::jsonb->'vinculo_autenticacion_actor',
 'relacion_ref',relacion_ref) FROM public.rpt27_ensayo_vector WHERE caso='$caso'" > "$scratch/intento_entrada.json"
 python3 - "$scratch" "$caso" "$motivo" <<'PYINTENTO'
import base64,hashlib,json,pathlib,sys
s=pathlib.Path(sys.argv[1]);e=json.loads((s/'intento_entrada.json').read_text());v=e['vinculo']
h=hashlib.sha256(sys.argv[2].encode()).hexdigest()[:32]
orden={'intento_ref':'intento_'+h,'registro_contexto_ref':v['registro_contexto_ref'],
'contexto_sha256':v['contexto_actor_huella_sha256'],'procedencia_sha256':v['manifiesto_procedencia_huella_sha256'],
'autenticacion_ref':v['autenticacion_ref'],'sesion_ref':v['sesion_ref'],
'autenticacion_sha256':v['autenticacion_huella_sha256'],'accion':'personal.relacion_rpt.consultar',
'modulo_id':'personal','recurso_ref':e['relacion_ref'],'finalidad_ref':'conciliar_relacion_laboral_para_rpt',
'resultado':'denegado' if sys.argv[3]=='denegado' else 'error','motivo_ref':'motivo_11111111111111111111111111111111',
'proceso':'rpt27-ensayo','canal':'interna_corporativa','correlacion_ref':'correlacion_'+h}
vb=base64.b64encode(json.dumps(v,separators=(',',':')).encode()).decode()
b=json.dumps(orden,separators=(',',':'));assert '$rpt27intento$' not in b
(s/'intento.sql').write_text("BEGIN ISOLATION LEVEL SERIALIZABLE; SET LOCAL timezone='UTC'; SELECT auditoria_ref FROM vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(decode('"+e['contexto_b64'].replace('\n','')+"','base64'),decode('"+vb+"','base64'),$rpt27intento$"+b+"$rpt27intento$::jsonb); COMMIT;\n")
PYINTENTO
 # OTRO LOGIN/conexión: el acuse sale después del COMMIT posterior al rollback.
 psql_run vec_rpt27_ensayo_registrador < "$scratch/intento.sql" > "$scratch/intento_acuse.txt" 2> "$scratch/intento.log" || return 75
 [[ $(cat "$scratch/intento_acuse.txt") == aud_v3_i_* ]] || return 75
}
rechazar() {
 local caso=$1 variante=${2:-valida} antes despues motivo=denegado
 antes=$(contadores)
 if consultar "$caso" "$variante" > "$scratch/denegada.out" 2> "$scratch/denegada.log"; then fallo 'negativa devolvió datos'; fi
 [[ ! -s $scratch/denegada.out && $(contadores) == "$antes" ]] || fallo 'rollback dejó datos/recibo/consumo/auditoría'
 [[ $(cat "$scratch/denegada.log") == *42501* || $(cat "$scratch/denegada.log") == *22023* || $(cat "$scratch/denegada.log") == *40001* ]] || fallo 'negativa inesperada o fixture sin mutación'
 [[ $(cat "$scratch/denegada.log") == *40001* ]] && motivo=no_disponible
 [[ $(cat "$scratch/denegada.log") == *22023* ]] && motivo=entrada_invalida
 registrar_intento "$caso" "$motivo" || fallo 'no_disponible: intento no confirmado'
 despues=$(contadores)
 python3 - "$antes" "$despues" <<'PYCONTADORES'
import sys
a=list(map(int,sys.argv[1].split('|')));b=list(map(int,sys.argv[2].split('|')))
assert b==a[:3]+[a[3]+1], 'intento fuera del rollback sin confirmar'
PYCONTADORES
 printf 'RPT27 negativo: %s/%s; rollback completo e intento durable.\n' "$caso" "$variante"
}
positivo() { # caso, estado esperado; fuente_version se conserva como decimalSTRING
 local antes despues
 antes=$(contadores)
 consultar "$1" > "$scratch/positivo_$1.json" 2> "$scratch/positivo.log" || fallo 'positivo nominal falló'
 python3 - "$scratch/positivo_$1.json" "$scratch/entrada.json" "$2" <<'PYRESULTADO'
import base64,json,pathlib,sys
r=json.loads(pathlib.Path(sys.argv[1]).read_text());e=json.loads(pathlib.Path(sys.argv[2]).read_text());m=json.loads(base64.b64decode(e['material_canonico_b64']))
assert set(r)=={'relacion','corte','cobertura','evidencia'}
x=r['relacion'];assert set(x)=={'empleado_ref','relacion_ref','organismo_ref','version','estado','periodo','procedencia'}
assert (x['empleado_ref'],x['relacion_ref'],x['organismo_ref'],x['version'])==(m['empleado_ref'],m['relacion_ref'],m['organismo_ref'],m['version_esperada'])
assert x['estado']==sys.argv[3] and r['cobertura']=='no_acreditada'
assert set(x['periodo'])=={'desde','hasta'} and isinstance(x['periodo']['hasta'],str)
assert set(x['procedencia'])=={'acto_ref','fuente_ref','fuente_version','certeza'}
assert x['procedencia']['certeza']=='no_acreditado' and isinstance(x['procedencia']['fuente_version'],str)
assert x['procedencia']['fuente_version'].isdigit() and int(x['procedencia']['fuente_version'])>0
assert r['corte']=={'vigente_en':m['vigente_en'],'conocido_en':m['conocido_en']}
assert set(r['evidencia'])=={'recibo_ref','decision_ref','efecto_ref','consumo_huella_sha256','auditoria_ref','consultada_en'}
assert r['evidencia']['recibo_ref'].startswith('relacionrpt:') and r['evidencia']['efecto_ref']==m['relacion_ref']
# El actor de la solicitud es el original acreditado, distinto del objetivo.
c=json.loads(base64.b64decode(e['contexto_b64']))
assert m['actor_ref']==c['principal_ref'] and all(v['referencia']!=m['empleado_ref'] for v in c['vinculos'] if v['tipo']=='empleado')
PYRESULTADO
 despues=$(contadores)
 python3 - "$antes" "$despues" <<'PYPOSITIVO'
import sys
a=list(map(int,sys.argv[1].split('|')));b=list(map(int,sys.argv[2].split('|')))
assert b==[a[0]+1,a[1]+1,a[2]+1,a[3]]
PYPOSITIVO
}
# Casos con actores/contextos CA registrados y permiso RPT propio.
if ! "$desde_registro" && ! "$desde_go"; then
if ! "$desde_replay"; then
 for estado in vigente suspendida finalizada; do preparar "positivo_$estado" "positivo_$estado" "$estado"; positivo "positivo_$estado" "$estado"; done
fi
rechazar positivo_vigente
for variante in actor perfil ambito relacion organismo campos cose; do preparar "$variante"; rechazar "$variante" "$variante"; done
# Revocación actual de la sesión después de emitir: no se transforma la
# identidad ni se rescata control1. El intento se registra en el pool separado.
preparar sesion_revocada
s=$(printf '%s' sesion_revocada | sha256sum | cut -c1-32)
valor "BEGIN; SET LOCAL ROLE vec_autorizacion_propietario;
 INSERT INTO vec_autorizacion.control_sesion_v1(control_sesion_ref,revision,sesion_ref,estado,huella_sha256,sesion_revalidada_en,sesion_valida_hasta)
 SELECT control_sesion_ref,2,sesion_ref,'revocada',repeat('f',64),clock_timestamp(),sesion_valida_hasta
 FROM vec_autorizacion.control_sesion_v1 WHERE control_sesion_ref='cse_rpt27_$s' AND revision=1;
 UPDATE vec_autorizacion.control_sesion_actual_v1 SET revision=2,actualizada_en=clock_timestamp(),acto_ref='acto:rpt27:revocacion-sintetica'
 WHERE sesion_ref='ses_rpt27_$s'; COMMIT;" > /dev/null
rechazar sesion_revocada
[[ $(valor "SELECT actor_ref='per_rpt27_$s' FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE intento_ref='intento_$s'") == t ]] || fallo 'intento perdió identidad ORIGINAL histórica acreditada'
# Última revisión conocida primero: ni estado ni fechas rescatan la revisión1.
preparar original_historia familia_historia
corte_historico=$(valor "SELECT date_trunc('microseconds',clock_timestamp())")
rel=$(valor "SELECT relacion_ref FROM public.rpt27_ensayo_vector WHERE caso='original_historia'")
valor "SELECT public.rpt27_ensayo_revision('$rel','finalizada','2028-01-01','2029-01-01')" > "$scratch/revision2.txt"
preparar revision_vieja familia_historia
rechazar revision_vieja
corte_fijado=$corte_historico
preparar corte_historico familia_historia
unset corte_fijado
positivo corte_historico vigente
# El registrador no puede leer historia, control, recibos ni intentos, ni invocar
# la lectura o consumidores54/74/149. Esta comprobación no le presta autoridad.
[[ $(valor "SELECT NOT has_table_privilege('vec_rpt27_ensayo_registrador','vec_personal.relacion_servicio_historia','SELECT')
 AND NOT has_table_privilege('vec_rpt27_ensayo_registrador','vec_personal.control_generacion_relacion_rpt','SELECT')
 AND NOT has_table_privilege('vec_rpt27_ensayo_registrador','vec_personal.recibo_relacion_para_rpt','SELECT')
 AND NOT has_table_privilege('vec_rpt27_ensayo_registrador','vec_autorizacion_atestada_v3.auditoria_consumo_v3','SELECT')
 AND NOT has_function_privilege('vec_rpt27_ensayo_registrador','vec_personal.consultar_relacion_para_rpt_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')") == t ]] || fallo 'registrador tiene lectura'
# Falla el permiso del LOGIN/pool propio, no se modifica una ACL de producto.
preparar registro_caido
antes=$(contadores)
valor 'REVOKE vec_autorizacion_atestada_v3_registrador_intentos FROM vec_rpt27_ensayo_registrador' > /dev/null
if consultar registro_caido > "$scratch/registro_caido.out" 2> "$scratch/registro_caido.log"; then fallo 'lectura sin registrador'; fi
[[ $(cat "$scratch/registro_caido.log") == *no_disponible* && ! -s $scratch/registro_caido.out && $(contadores) == "$antes" ]] || fallo 'preflight sin registro produjo efecto'
if registrar_intento registro_caido denegado; then fallo 'registrador sin permiso confirmó'; fi
valor 'GRANT vec_autorizacion_atestada_v3_registrador_intentos TO vec_rpt27_ensayo_registrador WITH ADMIN FALSE,INHERIT TRUE,SET FALSE' > /dev/null
# Snapshot SERIALIZABLE anterior al COMMIT del escritor: barrera27 tras V3
# fuerza40001 y todo el consumo se revierte; el intento se confirma aparte.
preparar concurrente
rel=$(valor "SELECT relacion_ref FROM public.rpt27_ensayo_vector WHERE caso='concurrente'")
antes=$(contadores)
psql_run postgres -c "SET application_name='rpt27_publicador_lento'; BEGIN;
 SELECT public.rpt27_ensayo_revision('$rel','suspendida','2026-01-01',NULL); SELECT pg_sleep(2); COMMIT;" > "$scratch/publicador.out" 2> "$scratch/publicador.log" &
publicador_pid=$!
lista=false
for _ in {1..80}; do
 if [[ $(valor "SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE application_name='rpt27_publicador_lento' AND wait_event='PgSleep')") == t ]]; then lista=true; break; fi
 sleep 0.02
done
[[ $lista == true ]] || fallo 'escritor no alcanzó barrera'
if consultar concurrente > "$scratch/concurrente.out" 2> "$scratch/concurrente.log"; then fallo 'snapshot obsoleto entregó datos'; fi
wait "$publicador_pid" || fallo 'escritor falló'
[[ $(cat "$scratch/concurrente.log") == *40001* && ! -s $scratch/concurrente.out && $(contadores) == "$antes" ]] || fallo 'barrera no produjo40001 limpio'
registrar_intento concurrente no_disponible || fallo '40001 sin intento durable'

fi
# El mismo binario de la emisión ejercita ahora el servicio y adaptadores Go
# originales de #437 con los dos LOGIN/pools reales por socket Unix aislado.
lector_go() {
 go_modo=$1 sandbox /scratch/rpt27.test -test.run '^TestLectorRPT27ConPoolsPostgreSQL$' -test.count=1 > "$scratch/go_$1.log" 2>&1 || fallo 'adaptador Go con pools reales falló: diagnóstico privado'
}
comprobar_contadores_go() {
 python3 - "$1" "$(contadores)" "$2" <<'PYCONTADORESGO'
import sys
a=list(map(int,sys.argv[1].split('|')));b=list(map(int,sys.argv[2].split('|')))
d=list(map(int,sys.argv[3].split('|')))
assert b==[x+y for x,y in zip(a,d)], 'efectos Go divergentes o intento dentro del rollback'
PYCONTADORESGO
}
go_sufijo=""
"$desde_go" && go_sufijo=_v69
go_positivo="go_positivo$go_sufijo"
go_revocada="go_revocada$go_sufijo"
go_concurrente="go_concurrente$go_sufijo"
if ! "$desde_registro" && ! "$solo_concurrente"; then
preparar "$go_positivo"
antes=$(contadores)
lector_go positivo
comprobar_contadores_go "$antes" '1|1|1|0'
# El fallo del sink ocurre después del COMMIT: no cambia el recibo ni crea
# otra operación. El emisor común cuenta la pérdida sin modificar el negocio.
preparar "go_sink_caido$go_sufijo"
antes=$(contadores)
lector_go sink_caido
comprobar_contadores_go "$antes" '1|1|1|0'
# Replay recupera sólo la negativa y su intento; no repite el efecto permitido.
# Reutiliza el vector de go_positivo para verificar la operación ya confirmada.
valor "SELECT entrada FROM public.rpt27_ensayo_vector WHERE caso='$go_positivo'" > "$scratch/entrada.json"
normalizar emitir
valor "SELECT bundle FROM public.rpt27_ensayo_vector WHERE caso='$go_positivo'" > "$scratch/salida.json"
antes=$(contadores)
lector_go replay
comprobar_contadores_go "$antes" '0|0|0|1'
# Punteros distintos no bastan: el registrador del LOGIN lector debe fallar.
preparar "go_cruzados$go_sufijo"
antes=$(contadores)
lector_go cruzados
comprobar_contadores_go "$antes" '0|0|0|0'
preparar "go_registro_caido$go_sufijo"
antes=$(contadores)
valor 'REVOKE vec_autorizacion_atestada_v3_registrador_intentos FROM vec_rpt27_ensayo_registrador' > /dev/null
fi
if ! "$solo_concurrente"; then
# La continuación no vuelve a emitir la capacidad ni repite positivos: el
# preflight debe fallar antes de usar el material del caso ya preparado.
antes=$(contadores)
lector_go registro_caido
comprobar_contadores_go "$antes" '0|0|0|0'
valor 'GRANT vec_autorizacion_atestada_v3_registrador_intentos TO vec_rpt27_ensayo_registrador WITH ADMIN FALSE,INHERIT TRUE,SET FALSE' > /dev/null
# La identidad cacheada de Go no rescata una sesión revocada en SQL actual.
preparar "$go_revocada"
s=$(printf '%s' "$go_revocada" | sha256sum | cut -c1-32)
valor "BEGIN; SET LOCAL ROLE vec_autorizacion_propietario;
 INSERT INTO vec_autorizacion.control_sesion_v1(control_sesion_ref,revision,sesion_ref,estado,huella_sha256,sesion_revalidada_en,sesion_valida_hasta)
 SELECT control_sesion_ref,2,sesion_ref,'revocada',repeat('f',64),clock_timestamp(),sesion_valida_hasta
 FROM vec_autorizacion.control_sesion_v1 WHERE control_sesion_ref='cse_rpt27_$s' AND revision=1;
 UPDATE vec_autorizacion.control_sesion_actual_v1 SET revision=2,actualizada_en=clock_timestamp(),acto_ref='acto:rpt27:revocacion-sintetica'
 WHERE sesion_ref='ses_rpt27_$s'; COMMIT;" > /dev/null
antes=$(contadores)
lector_go revocada
comprobar_contadores_go "$antes" '0|0|0|1'
fi
if "$solo_concurrente"; then go_concurrente=go_concurrente_v70; fi
# 40001 en el adaptador Go: sin DTO, consumo ni recibo; intento confirmado
# por el pool separado cuando ya terminó el rollback de la transacción lectora.
preparar "$go_concurrente"
rel=$(valor "SELECT relacion_ref FROM public.rpt27_ensayo_vector WHERE caso='$go_concurrente'")
antes=$(contadores)
# La puerta del ensayo sostiene al escritor antes del COMMIT. Se libera sólo
# cuando el LOGIN lector espera el advisory REAL de la relación, tras V3.
puerta="rpt27:ensayo:puerta:$go_concurrente"
coproc RPT27_PUERTA { psql_run postgres; }
puerta_in=${RPT27_PUERTA[1]}; puerta_out=${RPT27_PUERTA[0]}
printf "SELECT pg_advisory_lock(hashtextextended('%s',0));\n" "$puerta" >&"$puerta_in"
IFS= read -r puerta_lista <&"$puerta_out" || fallo 'puerta SQL no adquirida'
psql_run postgres -c "SET application_name='rpt27_publicador_go_lento'; BEGIN;
 SELECT public.rpt27_ensayo_revision('$rel','suspendida','2026-01-01',NULL);
 SELECT pg_advisory_xact_lock(hashtextextended('$puerta',0)); COMMIT;" > "$scratch/publicador_go.out" 2> "$scratch/publicador_go.log" &
publicador_pid=$!
lista=false
for _ in {1..80}; do
 if [[ $(valor "SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE application_name='rpt27_publicador_go_lento' AND wait_event='advisory')") == t ]]; then lista=true;break;fi
 sleep 0.02
done
if [[ $lista != true ]]; then
 printf "SELECT pg_advisory_unlock(hashtextextended('%s',0));\n" "$puerta" >&"$puerta_in"
 fallo 'escritor no sostuvo la relación antes del COMMIT'
fi
lector_go concurrente &
lector_pid=$!
lista=false
for _ in {1..80}; do
 if [[ $(valor "WITH llave AS(SELECT hashtextextended('vec_personal:registro-b2:relacion_servicio_historia:$rel',0) AS k)
 SELECT EXISTS(SELECT 1 FROM pg_locks l JOIN pg_stat_activity a ON a.pid=l.pid CROSS JOIN llave
 WHERE a.usename='vec_rpt27_ensayo_runtime' AND l.locktype='advisory' AND NOT l.granted
 AND l.classid=((k>>32)&4294967295)::oid AND l.objid=(k&4294967295)::oid AND l.objsubid=1)") == t ]]; then lista=true;break;fi
 sleep 0.01
done
printf "SELECT pg_advisory_unlock(hashtextextended('%s',0));\n" "$puerta" >&"$puerta_in"
IFS= read -r puerta_liberada <&"$puerta_out" || fallo 'puerta SQL no liberada'
[[ $puerta_liberada == t ]] || fallo 'puerta SQL no pertenecía al controlador'
printf '%s\n' '\q' >&"$puerta_in"
wait "$publicador_pid" || fallo 'escritor Go falló'
[[ $lista == true ]] || fallo 'lector no alcanzó el bloqueo real de la relación'
wait "$lector_pid" || fallo 'caso Go concurrente falló'
comprobar_contadores_go "$antes" '0|0|0|1'
[[ $(valor "SELECT count(*)=$("$solo_concurrente" && printf 4 || printf 3) AND bool_and(actor_ref IS NOT NULL AND perfil_activo_ref IS NOT NULL) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE tipo_registro='intento_nominal' AND proceso='rpt27-ensayo' AND intento_ref NOT IN (SELECT 'intento_'||substr(encode(sha256(convert_to(caso,'UTF8')),'hex'),1,32) FROM public.rpt27_ensayo_vector)") == t ]] || fallo 'intentos Go perdieron identidad original o no confirmaron'

captura "$scratch/final.json"
cmp -s "$scratch/preimagen.json" "$scratch/final.json" || fallo 'ensayo alteró extensiones previas de A'
printf '%s\n' 'RPT27 ENSAYO-OK: SQL/COSE, lector Go/pools, rollback, revocación, MVCC e historia previa conservada.'
