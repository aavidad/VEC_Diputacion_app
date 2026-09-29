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
accion=${1:-preparar}
case "$accion" in preparar|estado|reiniciar|parar|retirar|plan) ;; *) echo 'Uso: preparar_clon.sh [preparar|plan|estado|reiniciar|parar|retirar]' >&2; exit 2;; esac
[[ "$nombre" =~ ^vec-[a-z0-9-]+$ ]] || exit 2
[[ "$puerto_pg" =~ ^[0-9]+$ && "$puerto_web" =~ ^[0-9]+$ ]] || exit 2
((puerto_pg > 1024 && puerto_pg < 65536 && puerto_web > 1024 && puerto_web < 65536 && puerto_pg != puerto_web)) || exit 2
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

runtime() {
  local operacion=$1 hash
  hash=$(python3 - "$marcador" <<'PY'
import json,sys
print(json.load(open(sys.argv[1]))['commit'])
PY
)
  python3 "$guiones/clon_runtime.py" "$operacion" --repo "$repo" --commit "$hash" --state "$estado" --port "$puerto_web" --pg-port "$puerto_pg"
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
  [[ "$(docker inspect -f '{{ index .Config.Labels "vec.recorridos.owner" }}' "$nombre")" == Codex-M ]]
  [[ "$(docker inspect -f '{{ index .Config.Labels "vec.recorridos.state" }}' "$nombre")" == "$estado" ]]
}

if [[ "$accion" == estado ]]; then
  propio
  runtime status
  exit
fi
if [[ "$accion" == parar || "$accion" == retirar ]]; then
  registro_propio
  if docker inspect "$nombre" >/dev/null 2>&1; then propio; fi
  runtime stop
  if docker inspect "$nombre" >/dev/null 2>&1; then docker stop "$nombre" >/dev/null; fi
  rm -f -- "$estado/READY.json"
  if [[ "$accion" == retirar ]]; then
    python3 - "$marcador" <<'PY'
import json, pathlib, shutil, sys
v=json.load(open(sys.argv[1])); p=pathlib.Path(v['pgdata'])
assert p.parent == pathlib.Path('/dev/shm') and p.name.startswith('vec-recorridos-') and not p.is_symlink()
# PostgreSQL crea ficheros de otro uid: la limpieza usa el mismo contenedor
# efímero que creó el volumen, sobre una ruta de propiedad comprobada.
import subprocess
subprocess.run(['docker','run','--rm','--network','none','-v',str(p)+':/datos','alpine:3.22','sh','-c','find /datos -mindepth 1 -delete'],check=True,stdout=subprocess.DEVNULL)
p.rmdir()
PY
    rm -f -- "$marcador"
  fi
  exit
fi
if [[ "$accion" == reiniciar ]]; then
  propio
  runtime stop
  # Con --rm no se puede hacer stop/start de PostgreSQL; restart conserva el volumen.
  docker restart "$nombre" >/dev/null
  for ((i=0; i<60; i++)); do docker exec "$nombre" pg_isready -q -U postgres && break; sleep 1; done
  docker exec "$nombre" pg_isready -q -U postgres
  runtime start
  exit
fi

commit=$(git -C "$repo" rev-parse --verify "$referencia^{commit}")
git -C "$repo" merge-base --is-ancestor "$commit" origin/main || { echo 'La fuente debe estar integrada en origin/main.' >&2; exit 2; }
[[ -f "$archivo" && ! -L "$archivo" ]] || { echo 'Falta el estado sintético del hito 1.' >&2; exit 2; }
if [[ "$accion" == plan ]]; then
  python3 "$guiones/clon_sql.py" --repo "$repo" --source-ref "$commit" --container "$nombre" --state-dir "$estado" --plan
  exit
fi
if [[ -e "$marcador" ]]; then
  registro_propio
  python3 - "$marcador" "$commit" "$puerto_pg" "$puerto_web" <<'PY'
import json,sys
v=json.load(open(sys.argv[1])); assert (v['commit'],v['puerto_pg'],v['puerto_web']) == (sys.argv[2],int(sys.argv[3]),int(sys.argv[4])), 'Fuente/puertos diferentes: prepare otro estado y otro contenedor.'
PY
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
    echo 'El clon propio ya está en marcha; no se cambió su material.'
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
python3 "$guiones/clon_sql.py" --repo "$estado/fuente" --source-ref "$commit" --container "$nombre" --state-dir "$estado"
python3 "$guiones/clon_material.py" --repo "$estado/fuente" --container "$nombre" --output "$estado" --port "$puerto_web" --pg-port "$puerto_pg"
runtime build
runtime start
echo 'Clon preparado. Consulte el registro privado de estado antes de ejecutar recorridos.'
