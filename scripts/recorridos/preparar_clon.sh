#!/usr/bin/env bash
# Gestión local del clon; instalación retenida hasta conectar el kit D aprobado.
set -euo pipefail
umask 077

guiones=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repo=$(git -C "$guiones" rev-parse --show-toplevel)
estado=${VEC_RECORRIDOS_ESTADO:-"$HOME/.local/state/vec-recorridos"}
referencia=${VEC_RECORRIDOS_REFERENCIA:-73e56c106d12fdda0bd16d6fe573503c42c5495f}
nombre=${VEC_RECORRIDOS_CONTENEDOR:-vec-recorridos-local}
# Puerto lógico dentro del namespace PG; H6 no publica un puerto PostgreSQL host.
# Los registros históricos necesitan su puerto explícito para estado/parar/retirar.
puerto_pg=${VEC_RECORRIDOS_PUERTO_PG:-5432}
puerto_web=${VEC_RECORRIDOS_PUERTO_WEB:-18531}
puerto_smtp=${VEC_RECORRIDOS_PUERTO_SMTP:-11025}
puerto_correo_web=${VEC_RECORRIDOS_PUERTO_CORREO_WEB:-18532}
accion=${1:-preparar}
if (( $# )); then shift; fi
case "$accion" in preparar|estado|reiniciar|parar|retirar|plan|preparar-sql|verificar-sql|preparar-material-externo|exportar-alias|verificar-alias) ;;
  *) echo 'Uso: preparar_clon.sh [preparar|preparar-sql|verificar-sql|preparar-material-externo|exportar-alias|verificar-alias|plan|estado|reiniciar|parar|retirar]' >&2; exit 2;; esac

# Fases offline independientes. Rutas absolutas explícitas, fuera de Git y
# canónicas según la autoridad delegada; sus pines permanecen en cada herramienta.
# Material: --fuente RUTA --acuse RUTA --directorio RUTA.
# Alias: --binario RUTA --fuente RUTA --acuse RUTA --material RUTA --salida RUTA.
# Verificación: además --replay-receipt-sha256 SHA observado en la primera salida
# y conservado por Dirección fuera del paquete; nunca calcularlo aquí.
if [[ "$accion" == preparar-material-externo || "$accion" == exportar-alias || "$accion" == verificar-alias ]]; then
  offline_rechazar() { echo 'H6-OFFLINE arguments_invalid' >&2; exit 2; }
  declare -A entradas=()
  opciones=(--fuente --acuse --directorio)
  herramienta=clon_material_externo_offline.py
  if [[ "$accion" != preparar-material-externo ]]; then
    opciones=(--binario --fuente --acuse --material --salida)
    herramienta=clon_alias_export.py
  fi
  while (( $# )); do
    opcion=$1
    (( $# >= 2 )) || offline_rechazar
    case "$opcion" in
      --fuente|--acuse) ;;
      --directorio) [[ "$accion" == preparar-material-externo ]] || offline_rechazar;;
      --binario|--material|--salida) [[ "$accion" != preparar-material-externo ]] || offline_rechazar;;
      --replay-receipt-sha256) [[ "$accion" == verificar-alias ]] || offline_rechazar;;
      *) offline_rechazar;;
    esac
    [[ ! -v 'entradas[$opcion]' ]] || offline_rechazar
    entradas[$opcion]=$2
    shift 2
  done
  argumentos=()
  for opcion in "${opciones[@]}"; do
    [[ "${entradas[$opcion]:-}" == /* ]] || offline_rechazar
    argumentos+=("$opcion" "${entradas[$opcion]}")
  done
  if [[ "$accion" == verificar-alias ]]; then
    [[ "${entradas[--replay-receipt-sha256]:-}" =~ ^[0-9a-f]{64}$ ]] || offline_rechazar
    argumentos+=(--replay-receipt-sha256 "${entradas[--replay-receipt-sha256]}")
  fi
  exec python3 -B "$guiones/$herramienta" "${argumentos[@]}"
fi

# SQL62 requires nominal external pins; no historical45 fallback or inferred approval.
if [[ "$accion" == plan ]]; then
  exec python3 -B "$guiones/clon_h6_orquestador.py" plan --source-ref "$referencia" "$@"
fi
if [[ "$accion" == preparar-sql || "$accion" == verificar-sql ]]; then
  exec python3 -B "$guiones/clon_h6_orquestador.py" "$accion" --source-ref "$referencia" --state-dir "$estado" "$@"
fi
[[ "$#" -eq 0 ]] || exit 2

# Composición documental; nunca importa proveedores ni crea estado privado.
# Los bloqueos de material, transporte, aprobación y red preceden H1 y SQL.
if [[ "$accion" == preparar || "$accion" == reiniciar ]]; then
  exec python3 -B "$guiones/clon_h6_orquestador.py" "$accion" --source-ref "$referencia"
fi

[[ "$nombre" =~ ^vec-[a-z0-9-]+$ ]] || exit 2
for puerto in "$puerto_pg" "$puerto_web" "$puerto_smtp" "$puerto_correo_web"; do
  [[ "$puerto" =~ ^[0-9]+$ ]] || exit 2
  ((puerto > 1024 && puerto < 65536)) || exit 2
done
python3 -B - "$puerto_pg" "$puerto_web" "$puerto_smtp" "$puerto_correo_web" <<'PYTHON'
import sys
if len(set(map(int,sys.argv[1:]))) != 4:
    raise SystemExit('Los cuatro servicios necesitan puertos diferentes.')
PYTHON
# No crear ni normalizar silenciosamente un estado ausente/enlazado.
estado=$(python3 -B - "$estado" <<'PYTHON'
import os, stat, sys
from pathlib import Path
p=Path(sys.argv[1]).expanduser().absolute()
if p == Path('/') or p == Path.home() or '..' in p.parts:
    raise SystemExit('El estado privado necesita una ruta absoluta propia fuera de Git.')
for a in (p, *p.parents):
    if a.is_symlink() or (a/'.git').exists():
        raise SystemExit('El estado privado debe quedar fuera de Git y sin enlaces.')
fd=os.open(p, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
try:
    v=os.fstat(fd)
    if v.st_uid != os.getuid() or stat.S_IMODE(v.st_mode) & 0o077:
        raise SystemExit('El directorio de estado necesita propietario actual y permisos 0700.')
finally:
    os.close(fd)
print(p)
PYTHON
)
marcador="$estado/clon.json"

registro_propio() {
  python3 -B - "$marcador" "$nombre" "$estado" "$puerto_pg" "$puerto_web" <<'PYTHON'
import json, os, re, stat, sys
from pathlib import Path
fd=os.open(sys.argv[1], os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
with os.fdopen(fd, 'rb') as f:
    st=os.fstat(f.fileno())
    if not (stat.S_ISREG(st.st_mode) and st.st_nlink == 1 and st.st_uid == os.getuid() and not st.st_mode & 0o077 and st.st_size <= 65536):
        raise SystemExit('El registro no es un archivo privado propio.')
    v=json.load(f)
if not (v['contenedor'] == sys.argv[2] and v['estado'] == sys.argv[3] and v['propietario'] == 'Codex-M'):
    raise SystemExit('El registro no pertenece a este clon.')
if not re.fullmatch('[0-9a-f]{40}',v['commit']):
    raise SystemExit('El registro no identifica una fuente fija.')
if (v['puerto_pg'],v['puerto_web']) != (int(sys.argv[4]),int(sys.argv[5])):
    raise SystemExit('Los puertos no corresponden al registro del clon.')
p=Path(v['pgdata'])
if not (p.parent == Path('/dev/shm') and re.fullmatch('vec-recorridos-[A-Za-z0-9_-]+',p.name) and not p.is_symlink()):
    raise SystemExit('El volumen no es una ruta propia del clon.')
PYTHON
}

runtime() {
  local operacion=$1 hash
  hash=$(python3 -B - "$marcador" <<'PYTHON'
import json,sys
print(json.load(open(sys.argv[1]))['commit'])
PYTHON
)
  python3 -B "$guiones/clon_runtime.py" "$operacion" --mode interno --repo "$repo" --commit "$hash" --state "$estado" --port "$puerto_web" --pg-port "$puerto_pg"
}

comunicaciones() {
  python3 -B "$guiones/clon_comunicaciones.py" "$1" --repo "$repo" --state "$estado" --container "$nombre" --pg-port "$puerto_pg" --smtp-port "$puerto_smtp" --mailpit-http-port "$puerto_correo_web"
}

propio() {
  registro_propio
  pg_id=$(python3 -B - "$marcador" "$nombre" <<'PY'
import json,re,subprocess,sys
v=json.load(open(sys.argv[1]))
result=subprocess.run(['docker','inspect',sys.argv[2]],capture_output=True,check=True)
c=json.loads(result.stdout)[0]; labels=c['Config'].get('Labels') or {}
if labels.get('vec.recorridos.owner') != v['propietario'] or v['propietario'] != 'Codex-M':
    raise SystemExit('El contenedor pertenece a otro propietario.')
if labels.get('vec.recorridos.state') != v['estado']:
    raise SystemExit('El contenedor pertenece a otro estado privado.')
if c['Config']['Image'] != 'postgres:18.4':
    raise SystemExit('La imagen no corresponde al clon.')
if not any(m['Type']=='bind' and m['Destination']=='/var/lib/postgresql' and m['Source']==v['pgdata'] for m in c.get('Mounts',[])):
    raise SystemExit('El montaje no corresponde al volumen propio.')
if not re.fullmatch('[0-9a-f]{64}',c['Id']):
    raise SystemExit('El identificador del contenedor no es válido.')
print(c['Id'])
PY
)
}

if [[ "$accion" == estado ]]; then
  # El clon H6 nuevo se identifica por su journal; clon.json pertenece solo al
  # runtime anterior. No exigir ese marcador para consultar una fase SQL62.
  if [[ -e "$marcador" || -L "$marcador" ]]; then
    registro_propio
  elif [[ ! -f "$estado/sql-journal.json" || -L "$estado/sql-journal.json" ]]; then
    echo 'El estado no conserva un registro propio ni un journal H6.' >&2
    exit 2
  fi
  python3 -B - "$guiones" "$repo" "$estado" <<'PYTHON'
import json, os, subprocess, sys, uuid
from pathlib import Path
sys.path.insert(0,sys.argv[1])
import clon_sql as sql
s=Path(sys.argv[3])
v={'kit_d':'pendiente', 'ready':False, 'ready_legacy_presente':(s/'READY.json').exists(), 'journal':'ausente'}
try:
    fd=os.open(s/'sql-journal.json',os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
except FileNotFoundError:
    pass
except OSError:
    v.update(journal='bloqueado', motivo='journal privado inválido o enlazado; conservar evidencia')
else:
    try:
        sql.Journal._check_file(fd)
        with os.fdopen(fd,'rb',closefd=False) as f:
            j=json.loads(f.read(sql.MAX_JOURNAL+1))
        if not isinstance(j,dict) or j.get('version') != sql.JOURNAL_VERSION:
            raise sql.Refused('journal antiguo/ajeno; no convertir; reconstruir clon nuevo')
        if j.get('journal_sha') != sql.record_hash(j):
            raise sql.Refused('journal corrupto; conservar y reconstruir clon nuevo')
        uuid.UUID(j['run_id'])
        if j.get('pending') is not None:
            raise sql.Refused(sql.REBUILD)
        if (j.get('file_count') == 62 and
                j.get('source_commit') == j.get('approved_sql_ref') ==
                '73e56c106d12fdda0bd16d6fe573503c42c5495f' and
                j.get('phase') == 'awaiting_ad132' and
                isinstance(j.get('entries'), list) and len(j['entries']) == 62 and
                isinstance(j.get('installed'), list) and len(j['installed']) == 62):
            # El validador legacy solo conoce las 45 SQL H6 originales. Para
            # H1+62 se necesita verificar-sql con pines y acta externos; aquí
            # se muestra exclusivamente el diario local y nunca READY.
            v.update(journal='v2_sin_revalidacion_viva', fase=j['phase'],
                     sql_declaradas=62, requiere='verificar-sql con acta aprobada')
        else:
            plan=sql.validate_git_source(j['source_commit'],Path(sys.argv[2]))
            sql.validate_record(j,plan,sql.context_from_record(j))
            v.update(journal='v2', fase=j['phase'], sql_confirmadas=len(j['installed']))
    except sql.Refused as e:
        v.update(journal='bloqueado', motivo=str(e))
    except (ValueError,KeyError,TypeError,OSError,AttributeError,subprocess.SubprocessError):
        v.update(journal='bloqueado', motivo='journal inválido; conservar evidencia y reconstruir clon nuevo')
    finally:
        os.close(fd)
try:
    import clon_h6_orquestador as orchestrator
    v['composicion_h6'] = orchestrator.composition()
except (ImportError, OSError, ValueError):
    v['composicion_h6'] = {'ready': False, 'executable': False,
                           'blockers': ['orchestrator_unavailable']}
print(json.dumps(v,ensure_ascii=False))
PYTHON
  exit
fi

# Sólo la parada/retirada usa Docker, tras comprobar la propiedad del estado.
registro_propio
[[ ! -e "$estado/RETIRADO.json" && ! -L "$estado/RETIRADO.json" ]] || {
  echo 'Este estado conserva un clon retirado. Elija otro directorio para reconstruirlo.' >&2; exit 2;
}
[[ ! -L "$estado/preparar.lock" ]] || exit 2
exec 9<>"$estado/preparar.lock"
python3 -B - <<'PYTHON'
import os,stat
v=os.fstat(9)
if not stat.S_ISREG(v.st_mode) or v.st_nlink != 1 or v.st_uid != os.getuid() or v.st_mode & 0o077:
    raise SystemExit('El bloqueo no es un archivo privado propio.')
PYTHON
flock -n 9 || { echo 'El clon tiene otra operación en curso.' >&2; exit 1; }

if [[ "$accion" == parar || "$accion" == retirar ]]; then
  registro_propio
  pg_id=''
  if docker inspect "$nombre" >/dev/null 2>&1; then
    propio
  elif [[ "$accion" == retirar ]]; then
    # El marcador histórico no conserva dev/ino/run_id del volumen. Un nombre
    # con prefijo propio no sustituye la comprobación del montaje del contenedor.
    echo 'Retirada bloqueada: no se puede acreditar el volumen sin su contenedor. Conserve la copia para revisión manual.' >&2
    exit 1
  fi
  rm -f -- "$estado/READY.json"
  runtime stop
  comunicaciones stop
  if [[ -n "$pg_id" ]] && docker inspect "$pg_id" >/dev/null 2>&1; then docker stop "$pg_id" >/dev/null; fi
  rm -f -- "$estado/READY.json"
  if [[ "$accion" == retirar ]]; then
    python3 -B - "$marcador" <<'PY'
import json, os, pathlib, re, shutil, sys
v=json.load(open(sys.argv[1])); p=pathlib.Path(v['pgdata'])
if not (p.parent == pathlib.Path('/dev/shm') and re.fullmatch('vec-recorridos-[A-Za-z0-9_-]+',p.name) and not p.is_symlink()):
    raise SystemExit('El volumen no es una ruta propia del clon.')
# PostgreSQL crea ficheros de otro uid: la limpieza usa el mismo contenedor
# efímero que creó el volumen, sobre una ruta de propiedad comprobada.
import subprocess
subprocess.run(['docker','--host','unix:///var/run/docker.sock','run','--rm','--pull=never','--network','none','-v',str(p)+':/datos','alpine:3.22','sh','-c','find /datos -mindepth 1 -delete'],check=True,stdout=subprocess.DEVNULL)
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
    mv -T -- "$marcador" "$estado/RETIRADO.json"
  fi
  exit
fi
