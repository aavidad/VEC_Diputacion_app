#!/usr/bin/env bash
# Entorno local de recorridos: fuente fijada, base sintética y recursos propios.
set -euo pipefail
umask 077

guiones=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repo=$(git -C "$guiones" rev-parse --show-toplevel)
estado=${VEC_RECORRIDOS_ESTADO:-"$HOME/.local/state/vec-recorridos"}
archivo=${VEC_RECORRIDOS_ARCHIVO:-"$HOME/.local/state/vec-clon/estado-cidonia-20260929-hito1.tgz"}
referencia=${VEC_RECORRIDOS_REFERENCIA:-origin/main}
nombre=${VEC_RECORRIDOS_CONTENEDOR:-vec-recorridos-local}
puerto_pg=${VEC_RECORRIDOS_PUERTO_PG:-55531}
puerto_web=${VEC_RECORRIDOS_PUERTO_WEB:-18531}
puerto_smtp=${VEC_RECORRIDOS_PUERTO_SMTP:-11025}
puerto_correo_web=${VEC_RECORRIDOS_PUERTO_CORREO_WEB:-18532}
refrescar_prueba=${VEC_RECORRIDOS_REFRESCAR_PRUEBA_INTERNA:-false}
[[ "$refrescar_prueba" == true || "$refrescar_prueba" == false ]] || exit 2
accion=${1:-preparar}
case "$accion" in preparar|estado|reiniciar|parar|retirar|plan) ;; *) echo 'Uso: preparar_clon.sh [preparar|plan|estado|reiniciar|parar|retirar]' >&2; exit 2;; esac
[[ "$nombre" =~ ^vec-[a-z0-9-]+$ ]] || exit 2
for puerto in "$puerto_pg" "$puerto_web" "$puerto_smtp" "$puerto_correo_web"; do
  [[ "$puerto" =~ ^[0-9]+$ ]] || exit 2
  ((puerto > 1024 && puerto < 65536)) || exit 2
done
python3 - "$puerto_pg" "$puerto_web" "$puerto_smtp" "$puerto_correo_web" <<'PY'
import sys
if len(set(map(int, sys.argv[1:]))) != 4:
    raise SystemExit("Los cuatro servicios necesitan puertos diferentes.")
PY
python3 - "$estado" <<'PY'
from pathlib import Path
import sys
p = Path(sys.argv[1]).expanduser().resolve()
if not p.is_absolute() or p == Path('/') or p == Path.home() or any((a / '.git').exists() for a in (p, *p.parents)):
    raise SystemExit('El estado privado debe quedar fuera de cualquier repositorio.')
p.mkdir(mode=0o700, parents=True, exist_ok=True)
if p.stat().st_uid != __import__('os').getuid() or p.stat().st_mode & 0o077:
    raise SystemExit('El directorio de estado necesita propietario actual y permisos 0700.')
PY
[[ ! -L "$estado" ]] || exit 2
estado=$(realpath -- "$estado")
exec 9>"$estado/preparar.lock"
flock -n 9 || { echo 'El clon tiene otra operación en curso.' >&2; exit 1; }
marcador="$estado/clon.json"
if [[ -f "$estado/RETIRADO.json" ]]; then
  echo 'Este estado conserva un clon retirado. Elija otro directorio para reconstruirlo.' >&2
  exit 2
fi
restauracion_nueva=false
pgdata=''
limpiar_restauracion_incompleta() {
  if [[ "$restauracion_nueva" == true && ! -e "$marcador" ]]; then
    if [[ "$pgdata" == /dev/shm/vec-recorridos-* && -d "$pgdata" && ! -L "$pgdata" ]]; then
      docker run --rm --network none -v "$pgdata:/datos" alpine:3.22 sh -c 'find /datos -mindepth 1 -delete' >/dev/null
      rmdir -- "$pgdata"
    fi
    [[ ! -L "$estado/fuente" ]] && rm -rf -- "$estado/fuente"
  fi
}
trap limpiar_restauracion_incompleta EXIT

publicar_ready() {
runtime verify || return $?
python3 - "$estado" <<'PY'
import hashlib,json,pathlib,subprocess,sys
s=pathlib.Path(sys.argv[1]); original=json.loads((s/'material-manifest.json').read_text()); r=json.loads((s/'runtime-process.json').read_text()); j=json.loads((s/'sql-journal.json').read_text())
assert r['container_mode'] == 'interno'
m=json.loads(pathlib.Path(r['runtime_manifest_path']).read_text())
cfg=json.loads(pathlib.Path(r['runtime_config_path']).read_text())
assert cfg.get('VEC_PORTAL_PROCESO') == 'interno'
assert cfg.get('VEC_BOLSA_POLITICA_OFERTAS_ENABLED') == 'true', 'Falta preparar la configuración del hito 5.'
home=s/'chrome-home'; nss=home/'.pki/nssdb'; nss.mkdir(parents=True,mode=0o700,exist_ok=True); home.chmod(0o700); (home/'.pki').chmod(0o700)
if not (nss/'cert9.db').exists():
    subprocess.run(['certutil','-N','--empty-password','-d','sql:'+str(nss)],check=True,stdout=subprocess.DEVNULL)
subprocess.run(['certutil','-A','-d','sql:'+str(nss),'-n','vec-clon-sintetico','-t','C,,','-i',str(s/'material/ca/ca.crt')],check=True,stdout=subprocess.DEVNULL)
for f in nss.iterdir():
    if f.is_file(): f.chmod(0o600)
binary=pathlib.Path(r['exe']); actual=j.get('current_source_ref',j['source_ref'])
assert actual == r['source_commit'] == m['target']['source_commit']
v=dict(tipo='clon_local_h3_h5',clon='local',datos='sinteticos',hitos=['H3','H4','H5'],hitos_verificados=['H3','H4','H5'],clon_sintetico=True,clon_ref=json.loads((s/'clon.json').read_text())['contenedor'],origen='https://127.0.0.1:'+str(r['port']),commit=actual,binario=str(binary),binario_sha256=r['binary_sha256'],pid=r['pid'],sql_instaladas=len(j['installed']),material_sha256=hashlib.sha256(pathlib.Path(r['runtime_manifest_path']).read_bytes()).hexdigest(),portal_proceso='interno',recorridos_externos_habilitados=False,bloqueos=original.get('blockers',[]),chrome_home=str(home),ca=str(s/'material/ca/ca.crt'))
p=s/'READY.json'; t=s/'READY.json.nuevo'; t.write_text(json.dumps(v,ensure_ascii=False,indent=2)+'\n'); t.chmod(0o600); t.replace(p)
PY
}

finalizar_arranque() {
  rm -f -- "$estado/READY.json"
  runtime start
  if ! publicar_ready; then
    runtime stop
    rm -f -- "$estado/READY.json"
    return 1
  fi
}

runtime() {
  local operacion=$1 hash
  hash=$(python3 - "$marcador" <<'PY'
import json,sys
print(json.load(open(sys.argv[1]))['commit'])
PY
)
  python3 "$guiones/clon_runtime.py" "$operacion" --mode interno --repo "$repo" --commit "$hash" --state "$estado" --port "$puerto_web" --pg-port "$puerto_pg"
}

comunicaciones() {
  python3 "$guiones/clon_comunicaciones.py" "$1" --repo "$repo" --state "$estado" --container "$nombre" --pg-port "$puerto_pg" --smtp-port "$puerto_smtp" --mailpit-http-port "$puerto_correo_web"
}

registro_propio() {
  [[ -f "$marcador" ]] || { echo 'Falta el registro de propiedad del clon.' >&2; return 1; }
  python3 - "$marcador" "$nombre" "$estado" <<'PY'
import json, sys
v=json.load(open(sys.argv[1]))
assert v['contenedor'] == sys.argv[2] and v['estado'] == sys.argv[3] and v['propietario'] == 'Codex-M'
PY
}

propio() {
  registro_propio
  pg_id=$(python3 - "$marcador" "$nombre" <<'PY'
import json,re,subprocess,sys
v=json.load(open(sys.argv[1]))
result=subprocess.run(['docker','inspect',sys.argv[2]],capture_output=True,check=True)
c=json.loads(result.stdout)[0]; labels=c['Config'].get('Labels') or {}
assert labels.get('vec.recorridos.owner') == v['propietario'] == 'Codex-M'
assert labels.get('vec.recorridos.state') == v['estado']
assert c['Config']['Image'] == 'postgres:18.4'
assert any(m['Type']=='bind' and m['Destination']=='/var/lib/postgresql' and m['Source']==v['pgdata'] for m in c.get('Mounts',[]))
assert re.fullmatch('[0-9a-f]{64}',c['Id'])
print(c['Id'])
PY
)
}

if [[ "$accion" == estado ]]; then
  registro_propio
  if docker inspect "$nombre" >/dev/null 2>&1; then propio; fi
  runtime status
  exit
fi
if [[ "$accion" == parar || "$accion" == retirar ]]; then
  registro_propio
  rm -f -- "$estado/READY.json"
  pg_id=''
  if docker inspect "$nombre" >/dev/null 2>&1; then propio; fi
  runtime stop
  comunicaciones stop
  if [[ -n "$pg_id" ]] && docker inspect "$pg_id" >/dev/null 2>&1; then docker stop "$pg_id" >/dev/null; fi
  rm -f -- "$estado/READY.json"
  if [[ "$accion" == retirar ]]; then
    python3 - "$marcador" <<'PY'
import json, os, pathlib, re, shutil, sys
v=json.load(open(sys.argv[1])); p=pathlib.Path(v['pgdata'])
assert p.parent == pathlib.Path('/dev/shm') and p.name.startswith('vec-recorridos-') and not p.is_symlink()
# PostgreSQL crea ficheros de otro uid: la limpieza usa el mismo contenedor
# efímero que creó el volumen, sobre una ruta de propiedad comprobada.
import subprocess
subprocess.run(['docker','run','--rm','--network','none','-v',str(p)+':/datos','alpine:3.22','sh','-c','find /datos -mindepth 1 -delete'],check=True,stdout=subprocess.DEVNULL)
p.rmdir()
s=pathlib.Path(v['estado'])
# Conservar material y recibos; retirar fuentes, binarios y logs del guion.
directorios={'fuente','usuarios-source','usuarios-h4-source','tmp'}
ficheros={'build.log','runtime.log','usuarios-install.log','usuarios-h4-install.log','source.tar'}
for item in s.iterdir():
    directorio=item.name in directorios or re.fullmatch(r'(?:fuente|source)-[0-9a-f]{40}',item.name)
    fichero=item.name in ficheros or re.fullmatch(r'vec-server-[0-9a-f]{40}',item.name)
    if not (directorio or fichero):
        continue
    if item.is_symlink() or item.stat().st_uid != os.getuid():
        raise SystemExit('Una ruta temporal no pertenece al guion; se conserva para revisión.')
    if directorio and item.is_dir():
        shutil.rmtree(item)
    elif fichero and item.is_file() and item.stat().st_nlink == 1:
        item.unlink()
    else:
        raise SystemExit('Tipo inesperado en una ruta temporal; se conserva para revisión.')
PY
    mv -- "$marcador" "$estado/RETIRADO.json"
  fi
  exit
fi
if [[ "$accion" == reiniciar ]]; then
  propio
  rm -f -- "$estado/READY.json"
  runtime stop
  # Con --rm no se puede hacer stop/start de PostgreSQL; restart conserva el volumen.
  docker restart "$pg_id" >/dev/null
  for ((i=0; i<60; i++)); do docker exec "$pg_id" pg_isready -q -U postgres && break; sleep 1; done
  docker exec "$pg_id" pg_isready -q -U postgres
  finalizar_arranque
  exit
fi

commit=$(git -C "$repo" rev-parse --verify "$referencia^{commit}")
git -C "$repo" merge-base --is-ancestor "$commit" origin/main || { echo 'La fuente debe estar integrada en origin/main.' >&2; exit 2; }
[[ -f "$archivo" && ! -L "$archivo" ]] || { echo 'Falta el estado sintético del hito 1.' >&2; exit 2; }
if [[ "$accion" == plan ]]; then
  python3 "$guiones/clon_sql.py" --repo "$repo" --git-repo "$repo" --source-ref "$commit" --container "$nombre" --state-dir "$estado" --plan
  exit
fi
if [[ -e "$marcador" ]]; then
  registro_propio
  anterior=$(python3 - "$marcador" <<'PY'
import json,sys
print(json.load(open(sys.argv[1]))['commit'])
PY
)
  python3 - "$marcador" "$commit" "$puerto_pg" "$puerto_web" <<'PY'
import json,sys
v=json.load(open(sys.argv[1])); assert (v['puerto_pg'],v['puerto_web']) == (int(sys.argv[3]),int(sys.argv[4])), 'Puertos diferentes: prepare otro estado y otro contenedor.'
PY
  git -C "$repo" merge-base --is-ancestor "$anterior" "$commit" || { echo 'La fuente nueva no continúa la anterior.' >&2; exit 1; }
  if docker inspect "$nombre" >/dev/null 2>&1; then
    propio
  else
    pgdata=$(python3 - "$marcador" <<'PY'
import json,pathlib,sys
p=pathlib.Path(json.load(open(sys.argv[1]))['pgdata'])
assert p.parent == pathlib.Path('/dev/shm') and p.name.startswith('vec-recorridos-') and not p.is_symlink() and p.is_dir()
print(p)
PY
)
    docker run -d --rm --name "$nombre" --label vec.recorridos.owner=Codex-M --label "vec.recorridos.state=$estado" -p "127.0.0.1:$puerto_pg:5432" -v "$pgdata:/var/lib/postgresql" postgres:18.4 >/dev/null
  fi
  if [[ "$(runtime status | python3 -c 'import json,sys; print(json.load(sys.stdin)["running"])')" == True ]]; then
    [[ "$anterior" == "$commit" ]] || { echo 'Detenga el clon antes de actualizar su fuente.' >&2; exit 1; }
    if ! publicar_ready; then
      runtime stop
      rm -f -- "$estado/READY.json"
      exit 1
    fi
    echo 'El clon propio está en marcha y su registro está actualizado.'
    exit
  fi
else
  ! docker inspect "$nombre" >/dev/null 2>&1 || { echo 'El nombre de contenedor ya está ocupado.' >&2; exit 1; }
  [[ ! -e "$estado/fuente" ]] || { echo 'Existe una fuente sin registro de propiedad; revise el estado.' >&2; exit 1; }
  restauracion_nueva=true
  mkdir -m 700 "$estado/fuente"
  git -C "$repo" archive "$commit" | tar -x -C "$estado/fuente"
  pgdata=$(mktemp -d /dev/shm/vec-recorridos-XXXXXXXX)
  python3 - "$archivo" <<'PY'
import pathlib,sys,tarfile
with tarfile.open(sys.argv[1],mode='r|gz') as t:
    for item in t:
        p=pathlib.PurePosixPath(item.name)
        if p.is_absolute() or '..' in p.parts or item.isdev() or item.isfifo():
            raise SystemExit('El archivo de base contiene una entrada no permitida.')
        if item.issym() or item.islnk():
            q=pathlib.PurePosixPath(item.linkname)
            if q.is_absolute() or '..' in q.parts:
                raise SystemExit('El archivo de base contiene un enlace fuera del volumen.')
PY
  docker run --rm --network none -v "$pgdata:/datos" -v "$archivo:/estado.tgz:ro" alpine:3.22 tar -C /datos -xzf /estado.tgz
  python3 - "$marcador" "$estado" "$pgdata" "$nombre" "$commit" "$puerto_pg" "$puerto_web" "$archivo" <<'PY'
import hashlib,json,pathlib,sys
p=pathlib.Path(sys.argv[1]); archivo=pathlib.Path(sys.argv[8])
v=dict(propietario='Codex-M',estado=sys.argv[2],pgdata=sys.argv[3],contenedor=sys.argv[4],commit=sys.argv[5],puerto_pg=int(sys.argv[6]),puerto_web=int(sys.argv[7]),estado_h1_sha256=hashlib.file_digest(archivo.open('rb'),'sha256').hexdigest())
p.write_text(json.dumps(v,indent=2)+'\n'); p.chmod(0o600)
PY
  docker run -d --rm --name "$nombre" --label vec.recorridos.owner=Codex-M --label "vec.recorridos.state=$estado" -p "127.0.0.1:$puerto_pg:5432" -v "$pgdata:/var/lib/postgresql" postgres:18.4 >/dev/null
fi
for ((i=0; i<60; i++)); do docker exec "$nombre" pg_isready -q -U postgres && break; sleep 1; done
docker exec "$nombre" pg_isready -q -U postgres
fuente="$estado/fuente-$commit"
if [[ ! -e "$fuente" ]]; then
  mkdir -m 700 "$fuente"
  git -C "$repo" archive "$commit" | tar -x -C "$fuente"
fi
etapas_sql=$(python3 "$guiones/clon_sql.py" --repo "$fuente" --git-repo "$repo" --source-ref "$commit" --container "$nombre" --state-dir "$estado" --steps)
while IFS= read -r paso_sql; do
  fuente_sql="$estado/fuente-sql-$paso_sql"
  if [[ ! -e "$fuente_sql" ]]; then
    mkdir -m 700 "$fuente_sql"
    git -C "$repo" archive "$paso_sql" | tar -x -C "$fuente_sql"
  fi
  python3 "$guiones/clon_sql.py" --repo "$fuente_sql" --git-repo "$repo" --source-ref "$paso_sql" --container "$nombre" --state-dir "$estado"
done < <(python3 -c 'import json,sys; print("\n".join(json.load(sys.stdin)))' <<<"$etapas_sql")
# La última llamada verifica la fuente de la aplicación, incluso cuando todos
# los prefijos físicos ya estaban instalados.
python3 "$guiones/clon_sql.py" --repo "$fuente" --git-repo "$repo" --source-ref "$commit" --container "$nombre" --state-dir "$estado"
python3 - "$marcador" "$estado" "$commit" <<'PY'
import datetime,json,os,pathlib,sys
p=pathlib.Path(sys.argv[1]); state=pathlib.Path(sys.argv[2]); v=json.loads(p.read_text())
j=json.loads((state/'sql-journal.json').read_text())
actual=j.get('current_source_ref', j['source_ref'])
assert actual == sys.argv[3]
if v['commit'] != actual:
    h=state/'actualizaciones.jsonl'
    fd=os.open(h,os.O_WRONLY|os.O_CREAT|os.O_APPEND|os.O_NOFOLLOW,0o600)
    with os.fdopen(fd,'a') as f:
        f.write(json.dumps(dict(anterior=v['commit'],actual=actual,instante=datetime.datetime.now(datetime.timezone.utc).isoformat()))+'\n')
    v['commit']=actual
    t=state/'clon.json.nuevo'; t.write_text(json.dumps(v,indent=2)+'\n'); t.chmod(0o600); t.replace(p)
v.update(sql_instaladas=len(j['installed']),app_lista=False)
d=state/'DB_READY.json'; d.write_text(json.dumps(v,indent=2)+'\n'); d.chmod(0o600)
PY
runtime build
comunicaciones configure
opciones_material=()
if [[ -f "$estado/material-manifest.json" ]]; then opciones_material+=(--upgrade-source); fi
if [[ "$refrescar_prueba" == true ]]; then opciones_material+=(--refresh-internal-proof); fi
estado_material=0
python3 "$guiones/clon_material.py" --repo "$repo" --source-archive "$estado/source-$commit" --commit "$commit" --container "$nombre" --output "$estado" --port "$puerto_web" --pg-port "$puerto_pg" --repair-coverage-connect --repair-importacion-connect --repair-nominal-connect --complete-profiles "${opciones_material[@]}" || estado_material=$?
[[ "$estado_material" == 0 || "$estado_material" == 3 ]] || exit "$estado_material"
finalizar_arranque

echo 'Clon preparado. Consulte el registro privado de estado antes de ejecutar recorridos.'
