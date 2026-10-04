#!/usr/bin/env bash
# Instalación local; configuración y mantenimiento privados, fuera del kit.
set -Eeuo pipefail
umask 077
[[ $# == 3 ]] || { echo 'Uso: instalar.sh KIT CONFIG SHA256_MANIFIESTO' >&2; exit 2; }
KIT=$(realpath -- "$1"); CONFIG=$(realpath -- "$2"); MANIFIESTO_SHA=$3
paro() { printf 'PARO clave=%s actual=%s esperado=%s\n' "$1" "$2" "$3" >&2; return 1; }
[[ $(id -un) == openclaw ]] || paro usuario "$(id -un)" openclaw
[[ "$CONFIG" != "$KIT"/* && -f "$CONFIG" && ! -L "$2" ]] || paro config externa_requerida externa_regular
[[ $(stat -c '%u:%a' "$CONFIG") == "$(id -u):600" ]] || paro config_permisos "$(stat -c '%u:%a' "$CONFIG")" "$(id -u):600"
# Archivo elaborado y revisado por el operador; nunca se obtiene del kit.
# shellcheck disable=SC1090
source "$CONFIG"
for var in APP PG PGDATA PGCONF PGHBA ART CONF WEB_ROOT LOCALES_ROOT PGDATA_RUNTIME_PATH BACKUP_ROOT PREIMAGEN_DB_SHA PREIMAGEN_ART_SHA PREIMAGEN_CONF_SHA PREIMAGEN_WEB_SHA PREIMAGEN_LOCALES_SHA RUNTIME_MOUNTS_SHA MANTENIMIENTO_CERRAR MANTENIMIENTO_COMPROBAR MANTENIMIENTO_ABRIR; do
  [[ -n ${!var:-} ]] || paro "$var" ausente requerido
done
[[ -d "$ART" && ! -L "$ART" ]] || paro artefacto_tipo archivo_o_ausente directorio
[[ -d "$CONF" && ! -L "$CONF" ]] || paro config_tipo archivo_o_ausente directorio
[[ -d "$WEB_ROOT" && ! -L "$WEB_ROOT" ]] || paro web_tipo archivo_o_ausente directorio
[[ -d "$LOCALES_ROOT" && ! -L "$LOCALES_ROOT" ]] || paro locales_tipo archivo_o_ausente directorio
WEB_RUNTIME_PATH=${WEB_RUNTIME_PATH:-/app/web}
LOCALES_RUNTIME_PATH=${LOCALES_RUNTIME_PATH:-/app/locales}
BIN_RUNTIME_PATH=${BIN_RUNTIME_PATH:-/usr/local/bin/vec-server}
for var in CONTAINER_HOST CONTAINER_CONNECTION PODMAN_CONNECTION PODMAN_REMOTE PODMAN_HOST DOCKER_HOST; do
  [[ -z ${!var:-} ]] || paro "$var" remoto local
done
podman() { command podman --remote=false "$@"; }
for cmd in python3 podman sha256sum cp diff flock sync rg; do command -v "$cmd" >/dev/null || paro herramienta "$cmd" instalada; done
for var in MANTENIMIENTO_CERRAR MANTENIMIENTO_COMPROBAR MANTENIMIENTO_ABRIR; do
  [[ ${!var} == /* && -x ${!var} && ${!var} != "$KIT"/* ]] || paro "$var" ejecutable_invalido externo_revisado
done
[[ "$MANIFIESTO_SHA" =~ ^[0-9a-f]{64}$ ]] || paro manifiesto_sha formato sha256
actual=$(sha256sum "$KIT/SHA256SUMS"); actual=${actual%% *}
[[ "$actual" == "$MANIFIESTO_SHA" ]] || paro manifiesto_sha "$actual" "$MANIFIESTO_SHA"
# Validar cobertura del kit, rutas y estructura transaccional antes de parar.
validar_kit() {
python3 - "$KIT" <<'PY'
import hashlib,pathlib,re,sys
k=pathlib.Path(sys.argv[1]); covered=set()
def stop(key,actual,expected):
    raise SystemExit(f'PARO clave={key} actual={actual} esperado={expected}')
if (k/'NO_INSTALAR').exists(): stop('ensayo_kit','no_acreditado','ensayo_SQL_y_arranque_confirmados')
for p in k.rglob('*'):
    if p.is_symlink() or not (p.is_file() or p.is_dir()): stop('kit_tipo','enlace_o_especial','regular')
for line in (k/'SHA256SUMS').read_text().splitlines():
    match=re.fullmatch(r'([0-9a-f]{64})  ([A-Za-z0-9_./-]+)',line)
    if not match: stop('manifiesto_linea','formato','sha256_dos_espacios_ruta')
    digest,name=match.groups(); path=pathlib.PurePosixPath(name)
    if path.is_absolute() or '..' in path.parts or name in covered or name=='SHA256SUMS': stop('manifiesto_ruta','invalida','relativa_unica')
    f=k/name
    if not f.is_file(): stop('manifiesto_archivo','ausente','regular')
    current=hashlib.sha256(f.read_bytes()).hexdigest()
    if current!=digest: stop(name,current,digest)
    covered.add(name)
files={p.relative_to(k).as_posix() for p in k.rglob('*') if p.is_file()}-{'SHA256SUMS'}
if files!=covered: stop('manifiesto_cobertura','incompleta','todos_los_archivos')
for name in ('sql.list','consultas_preimagen.sql','bin/vec-server'):
    if name not in covered: stop(name,'ausente','incluido')
if not (k/'web/static').is_dir(): stop('web/static','ausente','directorio')
if not (k/'locales').is_dir(): stop('locales','ausente','directorio')
rows=(k/'sql.list').read_text().splitlines(); seen=set()
for name in rows:
    if not re.fullmatch(r'[A-Za-z0-9_./-]+\.up\.sql',name) or name not in covered or name in seen: stop('sql.list','ruta_invalida_o_repetida','up_unica_cubierta')
    seen.add(name); text=(k/name).read_text()
    # Las SQL revisadas de VEC contienen BEGIN/COMMIT en líneas independientes.
    begins=list(re.finditer(r'^BEGIN;\s*$',text,re.M)); commits=list(re.finditer(r'^COMMIT;\s*$',text,re.M))
    if len(begins)!=1 or len(commits)!=1 or begins[0].start()>commits[0].start() or text[commits[0].end():].strip(): stop(name,'tx_no_admitida','BEGIN_unico_COMMIT_final')
    # El kit solo admite el comando psql de errores, nunca includes/shell/connect.
    if any(line.startswith('\\') and line.strip() not in ('\\set ON_ERROR_STOP on','\\set ON_ERROR_STOP 1') for line in text.splitlines()): stop(name,'psql_comando','solo_ON_ERROR_STOP')
if not rows: stop('sql.list','vacia','plan_explicito')
PY
}
validar_kit

mkdir -p -- "$BACKUP_ROOT"
exec 9>"$BACKUP_ROOT/h9.lock"
flock -n 9 || paro instalacion activa exclusiva
COPIA=$(mktemp -d "$BACKUP_ROOT/pre-h9-XXXXXXXX")
# Trabajar con una instantánea privada verificada, no con el kit compartido.
cp -a -- "$KIT" "$COPIA/kit"
KIT="$COPIA/kit"
actual=$(sha256sum "$KIT/SHA256SUMS"); actual=${actual%% *}
[[ "$actual" == "$MANIFIESTO_SHA" ]] || paro manifiesto_copia "$actual" "$MANIFIESTO_SHA"
validar_kit
podman inspect -f '{{json .Mounts}}' "$PG" "$APP" > "$COPIA/montajes.jsonl"
actual=$(python3 - "$COPIA/montajes.jsonl" "$PGDATA_RUNTIME_PATH" "$PGDATA" "$BIN_RUNTIME_PATH" "$ART/vec-server" "$WEB_RUNTIME_PATH" "$WEB_ROOT" "$LOCALES_RUNTIME_PATH" "$LOCALES_ROOT" <<'PYMOUNTS'
import hashlib,json,pathlib,sys
containers=[json.loads(line) for line in pathlib.Path(sys.argv[1]).read_text().splitlines()]
def stop(key,actual,expected):
    raise SystemExit(f'PARO clave={key} actual={actual} esperado={expected}')
if len(containers)!=2: stop('runtime_montajes','cardinalidad','dos_contenedores')
for mounts in containers:
    mounts.sort(key=lambda m:(m.get('Destination',''),m.get('Source',''),m.get('Type','')))
    for mount in mounts:
        if isinstance(mount.get('Options'),list): mount['Options'].sort()
checks=[('pgdata',containers[0],sys.argv[2],sys.argv[3])]
checks += [(key,containers[1],sys.argv[i],sys.argv[i+1]) for key,i in [('binario',4),('web',6),('locales',8)]]
for key,mounts,runtime,expected in checks:
    runtime=pathlib.PurePosixPath(runtime); candidates=[]
    for mount in mounts:
        destination=pathlib.PurePosixPath(mount['Destination'])
        if runtime.is_relative_to(destination): candidates.append((len(destination.parts),mount,runtime.relative_to(destination)))
    if not candidates: stop('runtime_'+key,'sin_montaje','bind_inventariado')
    _,mount,suffix=max(candidates,key=lambda c:c[0])
    if mount.get('Type')!='bind': stop('runtime_'+key,mount.get('Type','ausente'),'bind')
    mapped=(pathlib.Path(mount['Source'])/suffix).resolve()
    expected=pathlib.Path(expected).resolve()
    if mapped!=expected:
        stop('runtime_'+key,hashlib.sha256(str(mapped).encode()).hexdigest(),hashlib.sha256(str(expected).encode()).hexdigest())
print(hashlib.sha256(json.dumps(containers,sort_keys=True,separators=(',',':')).encode()).hexdigest())
PYMOUNTS
)
[[ "$actual" == "$RUNTIME_MOUNTS_SHA" ]] || paro runtime_montajes "$actual" "$RUNTIME_MOUNTS_SHA"
ORIGENES=("$PGDATA" "$PGCONF" "$PGHBA" "$ART" "$CONF" "$WEB_ROOT" "$LOCALES_ROOT")
if declare -p EXTRA_COPIA >/dev/null 2>&1; then ORIGENES+=("${EXTRA_COPIA[@]}"); fi
declare -A COPIA_VISTA=()
for path in "${ORIGENES[@]}"; do
  [[ -z ${COPIA_VISTA[$path]:-} ]] || paro copia_origenes repetidos disjuntos
  COPIA_VISTA[$path]=si
  [[ "$path" == /* && "$path" != / && -e "$path" && ! -L "$path" && "$COPIA" != "$path"/* && ! -e "$path.h9-restaurar" ]] || paro origen_copia invalido absoluto_existente
  # WAL, tablespaces y configuración enlazados requieren otro plan de copia.
  [[ -z $(podman unshare find "$path" -type l -print -quit) ]] || paro copia_enlaces presentes sin_enlaces
  for other in "${ORIGENES[@]}"; do
    [[ "$path" == "$other" || "$path" != "$other"/* ]] || paro copia_origenes solapados disjuntos
  done
done
[[ -z $(podman unshare find "$PGDATA/pg_tblspc" -mindepth 1 -maxdepth 1 -print -quit) ]] || paro tablespaces externos ninguno
parado() { [[ $(podman inspect -f '{{.State.Running}}' "$1") == false ]]; }
cerrar() {
  "$MANTENIMIENTO_CERRAR" >"$COPIA/mantenimiento.log" 2>&1 || return 1
  "$MANTENIMIENTO_COMPROBAR" >>"$COPIA/mantenimiento.log" 2>&1 || return 1
}
asegurar_ventana() {
  parado "$APP" || { paro app activa parada; return 1; }
  "$MANTENIMIENTO_COMPROBAR" >>"$COPIA/mantenimiento.log" 2>&1 || { paro mantenimiento abierto cerrado; return 1; }
}
psql_local() { podman exec -i --env 'PGOPTIONS=-c lock_timeout=5s -c statement_timeout=1800s' "$PG" psql -X -q -At -v ON_ERROR_STOP=1 -U postgres -d postgres -f -; }
esperar_pg() {
  local i
  for i in {1..30}; do
    if podman exec "$PG" pg_isready -U postgres -d postgres >/dev/null 2>&1; then return; fi
    sleep 1
  done
  return 1
}
huella_arbol() {
  podman unshare python3 - "$1" <<'PY'
import hashlib,pathlib,sys
root=pathlib.Path(sys.argv[1]); h=hashlib.sha256()
for p in sorted(root.rglob('*'),key=lambda p:p.relative_to(root).as_posix()):
    if p.is_symlink() or not (p.is_file() or p.is_dir()): raise SystemExit(1)
    name=p.relative_to(root).as_posix(); mode=p.stat().st_mode & 0o777
    h.update(f'{name}\0{mode:o}\0'.encode())
    h.update(hashlib.sha256(p.read_bytes()).digest() if p.is_file() else b'directory')
print(h.hexdigest())
PY
}
comprobar_preimagen() {
  local actual
  podman exec -i --env 'PGOPTIONS=-c default_transaction_read_only=on -c statement_timeout=1800s' "$PG" psql -X -q -At -v ON_ERROR_STOP=1 -U postgres -d postgres -f - < "$KIT/consultas_preimagen.sql" > "$COPIA/preimagen-db.txt" 2>"$COPIA/preimagen-db.log" || return 1
  actual=$(sha256sum "$COPIA/preimagen-db.txt") || return 1
  actual=${actual%% *}
  [[ "$actual" == "$PREIMAGEN_DB_SHA" ]] || { paro preimagen_db "$actual" "$PREIMAGEN_DB_SHA"; return 1; }
  actual=$(huella_arbol "$ART") || return 1
  [[ "$actual" == "$PREIMAGEN_ART_SHA" ]] || { paro preimagen_artefacto "$actual" "$PREIMAGEN_ART_SHA"; return 1; }
  actual=$(huella_arbol "$CONF") || return 1
  [[ "$actual" == "$PREIMAGEN_CONF_SHA" ]] || { paro preimagen_config "$actual" "$PREIMAGEN_CONF_SHA"; return 1; }
  actual=$(huella_arbol "$WEB_ROOT") || return 1
  [[ "$actual" == "$PREIMAGEN_WEB_SHA" ]] || { paro preimagen_web "$actual" "$PREIMAGEN_WEB_SHA"; return 1; }
  actual=$(huella_arbol "$LOCALES_ROOT") || return 1
  [[ "$actual" == "$PREIMAGEN_LOCALES_SHA" ]] || { paro preimagen_locales "$actual" "$PREIMAGEN_LOCALES_SHA"; return 1; }
}
esperar_app() {
  local desde=$1 i
  for i in {1..60}; do
    podman logs --since "$desde" "$APP" >"$COPIA/arranque.log" 2>&1 || return 1
    [[ $(podman inspect -f '{{.State.Running}}' "$APP") == true ]] || return 1
    if rg -q 'vec server listening' "$COPIA/arranque.log"; then return 0; fi
    sleep 1
  done
  return 1
}
COPIA_LISTA=no; CAMBIOS=no; TRAFICO_ABIERTO=no; ETAPA=preimagen
recuperar() {
  trap - ERR EXIT INT TERM
  set +e
  printf 'PARO clave=etapa actual=%s esperado=instalacion_completa\n' "$ETAPA" >&2
  # Nunca restaurar una vez aceptadas escrituras. Los ganchos no abren parcialmente.
  if [[ "$TRAFICO_ABIERTO" == si ]]; then printf 'PARO clave=recuperacion actual=trafico_abierto esperado=conciliar_postimagen\n' >&2; exit 1; fi
  if ! cerrar || ! podman stop --time 30 "$APP" >/dev/null || ! parado "$APP"; then exit 1; fi
  if [[ "$COPIA_LISTA" == si && "$CAMBIOS" == si ]]; then
    if ! podman stop --time 60 "$PG" >/dev/null || ! parado "$PG"; then exit 1; fi
    mkdir -- "$COPIA/postimagen" || exit 1
    for i in "${!ORIGENES[@]}"; do
      path=${ORIGENES[$i]}
      # Conservar lo fallido; reponer cada preimagen sin DOWN ni sobreescritura.
      podman unshare cp -a -- "$COPIA/origen-$i" "$path.h9-restaurar" || exit 1
      podman unshare diff -qr -- "$COPIA/origen-$i" "$path.h9-restaurar" >"$COPIA/restore-$i.log" 2>&1 || exit 1
      if [[ -e "$path" ]]; then podman unshare mv -- "$path" "$COPIA/postimagen/origen-$i" || exit 1; fi
      podman unshare mv -- "$path.h9-restaurar" "$path" || exit 1
    done
    podman unshare sync -f "$PGDATA" || exit 1
    podman start "$PG" >/dev/null && esperar_pg || exit 1
    comprobar_preimagen || exit 1
    DESDE=$(date -u +%Y-%m-%dT%H:%M:%S.%NZ)
    podman start "$APP" >/dev/null || exit 1
    if ! esperar_app "$DESDE"; then podman stop --time 30 "$APP" >/dev/null; exit 1; fi
    if ! "$MANTENIMIENTO_COMPROBAR" >>"$COPIA/mantenimiento.log" 2>&1; then
      podman stop --time 30 "$APP" >/dev/null || exit 1
      parado "$APP" || exit 1
      cerrar || exit 1
      exit 1
    fi
    TRAFICO_ABIERTO=si
    "$MANTENIMIENTO_ABRIR" >>"$COPIA/mantenimiento.log" 2>&1 || exit 1
    printf 'RECUPERADA; aplicación anterior arrancada y tráfico abierto. Copia privada: %s\n' "$COPIA" >&2
  else
    printf 'Sin restauración; aplicación parada y mantenimiento cerrado. Copia privada: %s\n' "$COPIA" >&2
  fi
  exit 1
}
trap recuperar ERR INT TERM
cerrar
podman stop --time 30 "$APP" >/dev/null
parado "$APP" || paro app parada_requerida false
comprobar_preimagen
ETAPA=copia_fria
podman stop --time 60 "$PG" >/dev/null
parado "$PG" || paro pg parada_requerida false
for i in "${!ORIGENES[@]}"; do
  if ! parado "$APP" || ! parado "$PG"; then paro copia_fria servicio_activo ambos_parados; fi
  podman unshare cp -a -- "${ORIGENES[$i]}" "$COPIA/origen-$i"
  podman unshare diff -qr -- "${ORIGENES[$i]}" "$COPIA/origen-$i" >"$COPIA/copia-$i.log" 2>&1
 done
podman unshare sync -f "$COPIA"
COPIA_LISTA=si
podman start "$PG" >/dev/null
esperar_pg
comprobar_preimagen
CAMBIOS=si
while IFS= read -r sql; do
  [[ -n "$sql" ]] || continue
  asegurar_ventana
  ETAPA="ensayo:$sql"
  # Reemplazar únicamente COMMIT final. Una TX ensayo, otra TX instalación.
  python3 - "$KIT/$sql" > "$COPIA/ensayo.sql" <<'PY'
import pathlib,re,sys
text=pathlib.Path(sys.argv[1]).read_text()
print(re.sub(r'^COMMIT;\s*$', 'ROLLBACK;',text,flags=re.M),end='')
PY
  psql_local < "$COPIA/ensayo.sql" > "$COPIA/sql-ensayo.log" 2>&1
  ETAPA="commit:$sql"
  asegurar_ventana
  psql_local < "$KIT/$sql" > "$COPIA/sql-commit.log" 2>&1
  printf '%s\n' "$sql" >> "$COPIA/aplicadas.list"
done < "$KIT/sql.list"
ETAPA=artefactos
parado "$APP" || paro app activa parada
# La copia fría conserva el artefacto anterior completo. App permanece parada.
cp -a -- "$ART" "$COPIA/artefacto-nuevo"
install -m 755 -- "$KIT/bin/vec-server" "$COPIA/artefacto-nuevo/vec-server"
mv -- "$ART" "$COPIA/artefacto-sustituido"
cp -a -- "$COPIA/artefacto-nuevo" "$ART"
# Son los árboles servidos por /app, aunque procedan de un worktree distinto.
cp -a -- "$KIT/web" "$COPIA/web-nueva"
cp -a -- "$KIT/locales" "$COPIA/locales-nuevos"
mv -- "$WEB_ROOT" "$COPIA/web-sustituida"
cp -a -- "$COPIA/web-nueva" "$WEB_ROOT"
mv -- "$LOCALES_ROOT" "$COPIA/locales-sustituidos"
cp -a -- "$COPIA/locales-nuevos" "$LOCALES_ROOT"
ETAPA=arranque
DESDE=$(date -u +%Y-%m-%dT%H:%M:%S.%NZ)
podman start "$APP" >/dev/null
esperar_app "$DESDE" || paro arranque sin_listening listening
"$MANTENIMIENTO_COMPROBAR" >>"$COPIA/mantenimiento.log" 2>&1
# Tras abrir tráfico ya no se permite el retorno automático a la copia fría.
ETAPA=abrir_trafico
TRAFICO_ABIERTO=si
"$MANTENIMIENTO_ABRIR" >>"$COPIA/mantenimiento.log" 2>&1
trap - ERR INT TERM
printf 'H9-OK copia_privada=%s\n' "$COPIA"
