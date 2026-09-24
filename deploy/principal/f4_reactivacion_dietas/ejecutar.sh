#!/usr/bin/env bash
set -euo pipefail
umask 077

modo="${1:-}"
case "$modo" in
  --inventario|--rollback|--commit|--retirar-rollback|--retirar-commit) ;;
  *) echo 'uso: ejecutar.sh --inventario|--rollback|--commit|--retirar-rollback|--retirar-commit' >&2; exit 2 ;;
esac
: "${VEC_F4_EVIDENCIA_DIR:?falta VEC_F4_EVIDENCIA_DIR privado}"
if [[ "$modo" == --commit || "$modo" == --retirar-commit ]] && [[ "${VEC_F4_APLICAR:-}" != 'SI-F4-REVISADO' ]]; then
  echo 'COMMIT exige VEC_F4_APLICAR=SI-F4-REVISADO' >&2
  exit 2
fi

base_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
repo_dir="$(cd "$base_dir/../../.." && pwd -P)"
transporte=servicio
if [[ -n "${VEC_F4_POSTGRES_CONTAINER:-}" ]]; then
  transporte=contenedor
  [[ "$VEC_F4_POSTGRES_CONTAINER" =~ ^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$ ]] || {
    echo 'nombre de contenedor F4 invalido' >&2; exit 2;
  }
  : "${VEC_F4_PG_SOCKET_DIR:?falta socket PostgreSQL privado}"
  : "${VEC_F4_PG_PORT:?falta puerto PostgreSQL privado}"
  [[ "$VEC_F4_PG_SOCKET_DIR" =~ ^/[A-Za-z0-9_./-]+$ \
     && "$VEC_F4_PG_SOCKET_DIR" != *..* \
     && "$VEC_F4_PG_PORT" =~ ^[1-9][0-9]{0,4}$ \
     && "$VEC_F4_PG_PORT" -le 65535 ]] || {
    echo 'socket o puerto PostgreSQL F4 invalidos' >&2; exit 2;
  }
  for nombre in PGSERVICE PGSERVICEFILE PGPASSFILE PGHOST PGHOSTADDR PGPORT PGDATABASE PGUSER PGPASSWORD PGOPTIONS; do
    if [[ -n "${!nombre+x}" ]]; then
      echo "variable PG incompatible con transporte contenedor: $nombre" >&2
      exit 2
    fi
  done
  command -v podman >/dev/null || { echo 'falta podman' >&2; exit 2; }
else
  : "${PGSERVICE:?falta PGSERVICE privado}"
  : "${PGSERVICEFILE:?falta PGSERVICEFILE privado}"
  : "${PGPASSFILE:?falta PGPASSFILE privado}"
  [[ "$PGSERVICE" =~ ^[A-Za-z0-9_-]+$ ]] || { echo 'PGSERVICE invalido' >&2; exit 2; }
  for nombre in PGHOST PGHOSTADDR PGPORT PGDATABASE PGUSER PGPASSWORD PGOPTIONS; do
    if [[ -n "${!nombre+x}" ]]; then
      echo "variable PG incompatible con servicio privado: $nombre" >&2
      exit 2
    fi
  done
  for secreto in "$PGSERVICEFILE" "$PGPASSFILE"; do
    [[ -f "$secreto" && ! -L "$secreto" && "$(stat -c '%a' "$secreto")" == 600
        && "$(stat -c '%u' "$secreto")" == "$(id -u)" ]] || {
      echo 'fichero PG privado ausente, ajeno o sin modo 0600' >&2; exit 2;
    }
    ruta="$(realpath -e "$secreto")"
    case "$ruta" in "$repo_dir/"*) echo 'fichero PG dentro de Git' >&2; exit 2;; esac
  done
  python3 - "$PGSERVICEFILE" "$PGSERVICE" <<'PY'
import configparser
import sys

servicios = configparser.ConfigParser(interpolation=None)
with open(sys.argv[1], encoding="utf-8") as handle:
    servicios.read_file(handle)
nombre = sys.argv[2]
if nombre not in servicios or "password" in servicios[nombre]:
    raise SystemExit("servicio PG ausente o contiene contraseña; usar PGPASSFILE")
for clave in ("host", "port", "dbname", "user"):
    if not servicios[nombre].get(clave):
        raise SystemExit("servicio PG incompleto")
host = servicios[nombre]["host"]
port = servicios[nombre]["port"]
if (not host.startswith("/") or any(x in (".", "..") for x in host.split("/"))
        or not port.isascii() or not port.isdigit() or not 1 <= int(port) <= 65535
        or servicios[nombre]["dbname"] != "postgres"):
    raise SystemExit("servicio PG debe apuntar al socket local de postgres")
PY
fi

psql_ejecutar() {
  if [[ "$transporte" == contenedor ]]; then
    podman exec -i "$VEC_F4_POSTGRES_CONTAINER" \
      env -i PATH=/usr/local/bin:/usr/bin:/bin \
      psql -XAtq -w -v ON_ERROR_STOP=1 \
      -h "$VEC_F4_PG_SOCKET_DIR" -p "$VEC_F4_PG_PORT" \
      -U postgres -d postgres "$@"
  else
    psql -XAtq -v ON_ERROR_STOP=1 "$@"
  fi
}

evidencia_dir="$(realpath -e "$VEC_F4_EVIDENCIA_DIR")"
case "$evidencia_dir/" in "$repo_dir/"*) echo 'evidencia dentro de Git' >&2; exit 2;; esac
[[ -d "$evidencia_dir" && ! -L "$VEC_F4_EVIDENCIA_DIR" ]] || { echo 'directorio de evidencia invalido' >&2; exit 2; }
[[ "$(stat -c '%a' "$evidencia_dir")" == 700 ]] || { echo 'evidencia debe ser modo 0700' >&2; exit 2; }
[[ "$(stat -c '%u' "$evidencia_dir")" == "$(id -u)" ]] || { echo 'evidencia debe pertenecer al ejecutor' >&2; exit 2; }
if ! puerta="$(printf '%s\n' "SELECT current_setting('server_version_num')||'|'||" \
  "(SELECT rolsuper::text FROM pg_roles WHERE rolname=session_user)||'|'||" \
  "(session_user=current_user)::text||'|'||" \
  "(inet_server_addr() IS NULL)::text||'|'||current_database();" | psql_ejecutar)"; then
  echo 'fallo de transporte al verificar PostgreSQL/DBA' >&2; exit 2
fi
IFS='|' read -r version super mismo socket_local base <<< "$puerta"
[[ "$version" =~ ^[0-9]{6}$ && "$version" -ge 180000 && "$version" -lt 190000
   && "$super" == true && "$mismo" == true && "$socket_local" == true
   && "$base" == postgres ]] || {
  echo 'F4 exige PostgreSQL 18, socket local, base postgres y sesion DBA' >&2; exit 2;
}

marca="$(date -u +%Y%m%dT%H%M%SZ)"
inventario="$evidencia_dir/f4-inventario-$marca-$$.json"
psql_ejecutar < "$base_dir/inventario.sql" > "$inventario"
echo "inventario privado: $inventario"
if [[ "$modo" == --inventario ]]; then exit 0; fi

preimagen="$(mktemp "$evidencia_dir/.f4-preimagen.XXXXXXXX")"
variables="$(mktemp "$evidencia_dir/.f4-variables.XXXXXXXX")"
trap 'rm -f -- "$preimagen" "$variables"' EXIT
retirada=false
[[ "$modo" == --retirar-* ]] && retirada=true
preimagen_sql="$base_dir/preimagen.sql"
transaccion_sql="$base_dir/transaccion.sql"
plan_args=()
finalizar_var=f4_finalizar
if "$retirada"; then
  preimagen_sql="$base_dir/preimagen_retirada.sql"
  transaccion_sql="$base_dir/retirada.sql"
  plan_args=(--retirar)
  finalizar_var=f4_retirada_finalizar
fi
psql_ejecutar < "$preimagen_sql" > "$preimagen"
[[ "$(wc -l < "$preimagen")" -eq 1 && -s "$preimagen" ]] || {
  echo 'preimagen F4 revocada unica ausente o ambigua' >&2; exit 1;
}
(cd "$repo_dir" && go run ./deploy/principal/f4_reactivacion_dietas "${plan_args[@]}" < "$preimagen") > "$variables"

finalizar=ROLLBACK
[[ "$modo" == --commit || "$modo" == --retirar-commit ]] && finalizar=COMMIT
tx_estado=0
if { cat "$variables" "$transaccion_sql"; } | \
    psql_ejecutar -v "$finalizar_var=$finalizar" >/dev/null; then
  :
else
  tx_estado=$?
fi

post="$evidencia_dir/f4-post-$marca-$$.json"
if ! psql_ejecutar < "$base_dir/inventario.sql" > "$post"; then
  echo 'postinventario inaccesible: resultado transaccional incierto; no reintentar sin inspeccion DBA' >&2
  exit 1
fi
if [[ "$tx_estado" -ne 0 ]]; then
  echo "transporte/transaccion falló (código $tx_estado); revisar inventario privado $post antes de cualquier reintento" >&2
  exit 1
fi
python3 - "$post" "$finalizar" "$retirada" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as handle:
    estado = json.load(handle)
roles = estado["roles"]
actuales = [a for a in estado["asignaciones"] if a["puntero"]]
confirmado = sys.argv[2] == "COMMIT"
retirada = sys.argv[3] == "true"
login_esperado = confirmado != retirada
version_esperada = (4 if confirmado else 3) if retirada else (3 if confirmado else 2)
estado_esperado = ("revocada" if confirmado else "activa") if retirada else ("activa" if confirmado else "revocada")
actor_esperado = ("administracion:f4:retirada-dietas-r1d" if confirmado else "administracion:f4:reactivacion-dietas-r1d") if retirada else ("administracion:f4:reactivacion-dietas-r1d" if confirmado else "administracion:p6:retirada-dietas-r1d")
if (len(roles) != 8 or estado["sesiones"] or len(actuales) != 1
        or any(r["login"] != login_esperado for r in roles)
        or actuales[0]["estado"] != estado_esperado
        or actuales[0]["version"] != version_esperada
        or actuales[0]["actualizada_por"] != actor_esperado):
    raise SystemExit("postcondicion F4 fallida; revisar inventario privado (COMMIT puede estar confirmado)")
PY
echo "resultado=$finalizar inventario_post_privado=$post"
