#!/usr/bin/env bash
# Copia física sintética aislada: sólo se escribe en los recursos de este ensayo.
# Uso: script BASELINE_METADATA SOURCE_ROOT [SQL_LIST]
# Sin SQL_LIST acredita únicamente la preimagen. El ensayo funcional requiere
# además el escenario V3 real; aplicar las DDL no lo acredita.
# Preparado para baseline82 sintética fría, source 4617f08c32fd8f19796cd59a605bfece5de3873c.
# La lista SQL futura debe comprobar preexistencias V2 y ordenar los seis
# prerrequisitos propios de BR4; AD143/AD144 esperan el orden de Dirección.
# La fixture SQL acompaña al runner y debe ejecutarse expresamente después
# de instalar BR4. No contiene el escenario positivo de autorización V3.
set -Eeuo pipefail
umask 077

metadata=${1:?falta el manifiesto privado de la copia fría}
source_root=${2:?falta la raíz exacta de fuentes SQL}
sql_list=${3:-}
[[ -f $metadata && -d $source_root ]] || exit 2
source_root=$(realpath -- "$source_root")
name="codexg-gobierno-$$"
scratch=$(mktemp -d "/dev/shm/$name-XXXXXX")
private=$(mktemp -d "/var/tmp/$name-XXXXXX")
results_root="${XDG_STATE_HOME:-${HOME:?}/.local/state}/vec-codexg-gobierno-ensayo"
mkdir -p "$results_root"
result="$results_root/$name.json"
stage=baseline_metadata
status=PARO
image=postgres@sha256:3a82e1f56c8f0f5616a11103ac3d47e632c3938698946a7ad26da0df1334744a

# El cliente Docker usa únicamente el socket local y una configuración vacía.
docker_local() {
    env -i PATH=/usr/bin:/bin HOME="$private" DOCKER_CONFIG="$private/docker" \
        docker --host unix:///var/run/docker.sock "$@"
}
cleanup() {
    local code=$?
    trap - EXIT INT TERM
    docker_local rm -f "$name" >/dev/null 2>&1 || true
    docker_local run --rm --pull never --network none --read-only --user 0:0 \
        --name "$name-cleanup" \
        --cap-drop ALL --cap-add DAC_OVERRIDE --security-opt no-new-privileges --cpus .2 --memory 64m \
        --pids-limit 8 --mount "type=bind,src=$scratch,dst=/var/lib/postgresql" \
        --entrypoint /bin/rm "$image" -rf -- /var/lib/postgresql/18 \
        >/dev/null 2>&1 || true
    python3 - "$result" "$private" "$status" "$stage" "$code" <<'PY'
import json,pathlib,sys
out,private,status,stage,code=sys.argv[1:]
root=pathlib.Path(private)
document={'status':status,'stage':stage,'exit_code':int(code)}
startup=root/'logs/postgresql.log'
if status=='PARO' and startup.is_file():
    # Arranque con log_statement=none y sin incluir sentencias SQL. La causa
    # mínima se conserva antes de borrar el contenedor y los ficheros propios.
    message=startup.read_bytes()[:8192].decode('utf8','replace')
    fatal=[line.partition('FATAL:')[2].strip() for line in message.splitlines() if 'FATAL:' in line]
    nonempty=[line.strip() for line in message.splitlines() if line.strip()]
    if fatal or nonempty: document['startup_cause']=(fatal or nonempty)[0][:512]
error=root/'error.log'
if status=='PARO' and error.is_file():
    import re
    message=error.read_bytes()[:65536].decode('utf8','replace')
    conditions={'Disk quota exceeded':'disk_quota', 'Permission denied':'permission_denied',
                'No space left on device':'disk_full', 'Cannot mkdir':'mkdir_failed'}
    document['actual']=next((value for key,value in conditions.items() if key in message),
                            'exit_'+code)
    states=re.findall(r'^ERROR:\s+([0-9A-Z]{5}):',message,re.M)
    if states: document['sqlstate']=states[0]
for label in ('source_stamp','baseline','after'):
    entry=root/(label+'.json')
    if entry.is_file(): document[label]=json.loads(entry.read_text())
pathlib.Path(out).write_text(json.dumps(document,sort_keys=True)+'\n')
PY
    rm -rf -- "$scratch" "$private" 2>/dev/null || true
    if [[ $status == PARO ]]; then
        printf 'PARO %s expected=completed actual=exit_%s\n' "$stage" "$code" >&2
    fi
    printf 'RESULTADO %s\n' "$result"
    exit "$code"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

# No extrae enlaces, dispositivos ni rutas fuera de PGDATA. Verifica el
# contenido, no inode/ctime/xattrs, y nunca modifica el archivo compartido.
python3 - "$metadata" "$source_root" "$sql_list" "$scratch" "$private" \
    2> "$private/error.log" <<'PY'
import hashlib,json,pathlib,shutil,subprocess,sys,tarfile
metadata,source_root,sql_list,scratch,private=map(pathlib.Path,sys.argv[1:])
document=json.loads(metadata.read_text())
if document.get('status')!='STOPPED_BY_OPERATOR_FIN_20261001_2315':
    raise SystemExit('PARO baseline_state expected=stopped actual=incompatible')
archive=pathlib.Path(document['archive'])
if document.get('archive_sha256')!='326f7a2dcb59c89cb1726eda9a49e3b04677c77b3906993613787040d93c378f':
    raise SystemExit('PARO baseline_source expected=baseline82_cold actual=unapproved_hash')
def digest(path):
    with path.open('rb') as file: return hashlib.file_digest(file,'sha256').hexdigest()
if digest(archive)!=document['archive_sha256']:
    raise SystemExit('PARO archive_hash expected=manifest actual=mismatch')
stamp={'baseline_sha256':document['archive_sha256'],
       'baseline_source':'4617f08c32fd8f19796cd59a605bfece5de3873c',
       'source_ref':subprocess.check_output(
           ['/usr/bin/git','-C',str(source_root),'rev-parse','HEAD'],
           env={'PATH':'/usr/bin:/bin','GIT_CONFIG_NOSYSTEM':'1',
                'GIT_CONFIG_GLOBAL':'/dev/null'},text=True).strip(),
       'sql':[]}
if str(sql_list)!='.':
    seen=set()
    for line in sql_list.read_text().splitlines():
        if not line.strip() or line.startswith('#'): continue
        entry=pathlib.PurePosixPath(line)
        if entry.is_absolute() or '..' in entry.parts or line in seen:
            raise SystemExit('PARO sql_list expected=unique_relative actual=invalid')
        seen.add(line)
        target=(source_root/entry).resolve(strict=True)
        if source_root not in target.parents or not target.is_file():
            raise SystemExit('PARO sql_source expected=contained_file actual=invalid')
        copy=private/('sql-'+str(len(stamp['sql']))+'.sql')
        shutil.copyfile(target,copy)
        stamp['sql'].append({'path':line,'sha256':digest(copy)})
    if not stamp['sql']: raise SystemExit('PARO sql_list expected=nonempty actual=empty')
with tarfile.open(archive,'r:gz') as bundle:
    members=bundle.getmembers()
    if sum(m.size for m in members)>1024**3:
        raise SystemExit('PARO archive_size expected=le_1GiB actual=larger')
    for member in members:
        path=pathlib.PurePosixPath(member.name)
        if (path.is_absolute() or '..' in path.parts
            or (str(path)!='.' and path.parts[:2]!=('18','docker'))
            or not (member.isfile() or member.isdir())):
            # El directorio superior 18 también forma parte de esta copia.
            if not (str(path)=='18' and member.isdir()):
                raise SystemExit('PARO archive_entry expected=pgdata_regular actual=invalid')
    if bundle.extractfile('./18/docker/PG_VERSION').read().strip()!=b'18':
        raise SystemExit('PARO pg_version expected=18 actual=mismatch')
(private/'archive_path').write_text(str(archive))
stamp['stamp_sha256']=hashlib.sha256(json.dumps(
    stamp,sort_keys=True,separators=(',',':')).encode()).hexdigest()
(private/'source_stamp.json').write_text(json.dumps(stamp,sort_keys=True))
PY

mkdir -p "$private/config"
chmod 700 "$scratch"
chmod 755 "$private/config"
stage=archive_extract
archive=$(<"$private/archive_path")
docker_local run --rm --pull never --network none --read-only --user 0:0 \
    --name "$name-extract" \
    --cap-drop ALL --cap-add CHOWN --cap-add DAC_OVERRIDE \
    --security-opt no-new-privileges --cpus .5 --memory 768m \
    --pids-limit 8 --mount "type=bind,src=$scratch,dst=/var/lib/postgresql" \
    --mount "type=bind,src=$archive,dst=/input.tgz,readonly" \
    --entrypoint /bin/sh "$image" -ec '
        tar -xzf /input.tgz -C /var/lib/postgresql --strip-components=1 --no-same-owner --no-same-permissions
        chown -R 999:999 -- /var/lib/postgresql/18
    ' \
    > "$private/output.log" 2> "$private/error.log"
# Sólo cambia configuración del clon: ninguna credencial o configuración de
# salida del origen queda activa. La identidad postgres sirve al bootstrap local.
stage=clone_config
docker_local run --rm --pull never --network none --read-only --user 999:999 \
    --name "$name-config" \
    --cap-drop ALL --security-opt no-new-privileges --cpus .1 --memory 16m \
    --pids-limit 8 --mount "type=bind,src=$scratch/18,dst=/var/lib/postgresql" \
    --entrypoint /bin/sh "$image" -c ': > /var/lib/postgresql/docker/postgresql.auto.conf'
printf 'local all all trust\n' > "$private/config/pg_hba.conf"
cat > "$private/config/postgresql.conf" <<'CONF'
listen_addresses = ''
unix_socket_directories = '/socket'
hba_file = '/config/pg_hba.conf'
ssl = off
shared_preload_libraries = ''
shared_buffers = '32MB'
max_connections = 20
logging_collector = off
log_statement = 'none'
log_min_error_statement = 'panic'
log_min_messages = 'error'
CONF
chmod 644 "$private/config/pg_hba.conf" "$private/config/postgresql.conf"
mkdir -p "$private/logs"
# Sólo este contenedor puede llegar al directorio por el mount; el padre host
# es 0700. El archivo queda privado tras copiar su causa mínima al resultado.
chmod 777 "$private/logs"

stage=container_start
docker_local image inspect "$image" >/dev/null
docker_local run --detach --rm --pull never --name "$name" --network none \
    --read-only --user 999:999 --cap-drop ALL \
    --security-opt no-new-privileges --cpus 1 --memory 768m --pids-limit 64 \
    --ulimit nofile=1024:1024 --ulimit fsize=1073741824:1073741824 \
    --tmpfs /tmp:rw,noexec,nosuid,nodev,size=16m \
    --tmpfs /socket:rw,noexec,nosuid,nodev,size=1m,uid=999,gid=999,mode=0700 \
    --mount "type=bind,src=$scratch/18,dst=/var/lib/postgresql" \
    --mount "type=bind,src=$private/config,dst=/config,readonly" \
    --mount "type=bind,src=$private/logs,dst=/logs" \
    --entrypoint /usr/bin/env "$image" -i PATH=/usr/lib/postgresql/18/bin:/usr/local/bin:/usr/bin:/bin \
    HOME=/tmp LANG=C.UTF-8 TMPDIR=/tmp /bin/sh -ec \
    'exec postgres -D /var/lib/postgresql/docker -c config_file=/config/postgresql.conf 2>/logs/postgresql.log' >/dev/null

ready=false
for ((attempt=0; attempt<60; attempt++)); do
    if docker_local exec "$name" env -i PATH=/usr/lib/postgresql/18/bin:/usr/local/bin:/usr/bin:/bin \
        HOME=/tmp PGHOST=/socket pg_isready -q -U postgres -d postgres \
        > /dev/null 2>> "$private/error.log"; then
        ready=true
        break
    fi
    sleep 0.5
done
[[ $ready == true ]]
# Las ejecuciones SQL tienen límites de reloj, locks y memoria; las entradas
# se transmiten por stdin, sin montar las fuentes ni otros recursos del host.
psql_run() {
    timeout --signal=TERM --kill-after=5 120 env -i PATH=/usr/bin:/bin \
        HOME="$private" DOCKER_CONFIG="$private/docker" \
        docker --host unix:///var/run/docker.sock exec --interactive "$name" \
        env -i PATH=/usr/lib/postgresql/18/bin:/usr/local/bin:/usr/bin:/bin HOME=/tmp LANG=C.UTF-8 \
        PGHOST=/socket PGOPTIONS='-c statement_timeout=60000 -c lock_timeout=5000 -c idle_in_transaction_session_timeout=10000 -c work_mem=4MB' \
        psql -X -q -At --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
        --username postgres --dbname postgres "$@"
}
inventory() {
    psql_run --command "SELECT json_build_object(
        'server_version',current_setting('server_version'),
        'roles',(SELECT count(*) FROM pg_roles),
        'memberships',(SELECT count(*) FROM pg_auth_members),
        'v3_present',to_regnamespace('vec_autorizacion_atestada_v3') IS NOT NULL,
        'baremo_present',to_regnamespace('vec_bolsa_reglas_baremo') IS NOT NULL,
        'acl_sha256',encode(sha256(convert_to((SELECT coalesce(string_agg(
            n.nspname||'.'||p.proname||':'||pg_get_function_identity_arguments(p.oid)||':'||
            coalesce(p.proacl::text,''),'|' ORDER BY n.nspname,p.oid), '')
            FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
            WHERE n.nspname LIKE 'vec_%'),'UTF8')),'hex'),
        'roles_sha256',encode(sha256(convert_to((SELECT string_agg(
            rolname||':'||rolcanlogin||':'||rolsuper||':'||rolbypassrls,
            '|' ORDER BY rolname) FROM pg_roles),'UTF8')),'hex'))" \
        > "$private/$1.json" 2> "$private/error.log"
}
stage=baseline_inventory
inventory baseline
python3 - "$private/baseline.json" <<'PY'
import json,sys
b=json.load(open(sys.argv[1]))
if (not b['server_version'].startswith('18.4') or not b['v3_present']
    or b['roles']!=194 or b['memberships']!=94):
    raise SystemExit('PARO preimage expected=PG18.4_V3 actual=incompatible')
print('CLON_READY PostgreSQL=18.4 V3=present')
PY

if [[ -n $sql_list ]]; then
    index=0
    while [[ -f $private/sql-$index.sql ]]; do
        stage="sql_$index"
        psql_run --output /dev/null < "$private/sql-$index.sql" \
            > "$private/output.log" 2> "$private/error.log"
        index=$((index+1))
    done
    stage=after_inventory
    inventory after
    status=DDL_APPLIED_NEEDS_REAL_V3_SCENARIO
    printf '%s SQL=%s\n' "$status" "$index"
else
    status=PREIMAGE_ONLY
fi
stage=completed
