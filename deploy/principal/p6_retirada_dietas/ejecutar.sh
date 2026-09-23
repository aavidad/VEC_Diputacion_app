#!/usr/bin/env bash
set -euo pipefail
umask 077

modo="${1:-}"
case "$modo" in
  --inventario|--rollback|--commit) ;;
  *) echo 'uso: ejecutar.sh --inventario|--rollback|--commit' >&2; exit 2 ;;
esac
: "${VEC_P6_EVIDENCIA_DIR:?falta VEC_P6_EVIDENCIA_DIR privado}"
if [[ "$modo" == --commit && "${VEC_P6_APLICAR:-}" != 'SI-P6-REVISADO' ]]; then
  echo 'COMMIT exige VEC_P6_APLICAR=SI-P6-REVISADO' >&2
  exit 2
fi

base_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
repo_dir="$(cd "$base_dir/../../.." && pwd -P)"
transporte=servicio
if [[ -n "${VEC_P6_POSTGRES_CONTAINER:-}" ]]; then
  transporte=contenedor
  [[ "$VEC_P6_POSTGRES_CONTAINER" =~ ^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$ ]] || {
    echo 'nombre de contenedor P6 invalido' >&2; exit 2;
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
PY
fi

psql_ejecutar() {
  if [[ "$transporte" == contenedor ]]; then
    podman exec -i "$VEC_P6_POSTGRES_CONTAINER" \
      psql -XAtq -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"
  else
    psql -XAtq -v ON_ERROR_STOP=1 "$@"
  fi
}

evidencia_dir="$(realpath -e "$VEC_P6_EVIDENCIA_DIR")"
case "$evidencia_dir/" in "$repo_dir/"*) echo 'evidencia dentro de Git' >&2; exit 2;; esac
[[ -d "$evidencia_dir" && ! -L "$VEC_P6_EVIDENCIA_DIR" ]] || { echo 'directorio de evidencia invalido' >&2; exit 2; }
[[ "$(stat -c '%a' "$evidencia_dir")" == 700 ]] || { echo 'evidencia debe ser modo 0700' >&2; exit 2; }
[[ "$(stat -c '%u' "$evidencia_dir")" == "$(id -u)" ]] || { echo 'evidencia debe pertenecer al ejecutor' >&2; exit 2; }
if ! puerta="$(printf '%s\n' "SELECT current_setting('server_version_num')||'|'||" \
  "(SELECT rolsuper::text FROM pg_roles WHERE rolname=session_user)||'|'||" \
  "(session_user=current_user)::text;" | psql_ejecutar)"; then
  echo 'fallo de transporte al verificar PostgreSQL/DBA' >&2; exit 2
fi
IFS='|' read -r version super mismo <<< "$puerta"
[[ "$version" =~ ^[0-9]{6}$ && "$version" -ge 180000 && "$version" -lt 190000
   && "$super" == true && "$mismo" == true ]] || {
  echo 'P6 exige PostgreSQL 18 y sesion DBA sin SET ROLE' >&2; exit 2;
}

marca="$(date -u +%Y%m%dT%H%M%SZ)"
inventario="$evidencia_dir/p6-inventario-$marca-$$.json"
psql_ejecutar < "$base_dir/inventario.sql" > "$inventario"
echo "inventario privado: $inventario"
if [[ "$modo" == --inventario ]]; then exit 0; fi

preimagen="$(mktemp "$evidencia_dir/.p6-preimagen.XXXXXXXX")"
variables="$(mktemp "$evidencia_dir/.p6-variables.XXXXXXXX")"
trap 'rm -f -- "$preimagen" "$variables"' EXIT
psql_ejecutar < "$base_dir/preimagen.sql" > "$preimagen"
[[ "$(wc -l < "$preimagen")" -eq 1 && -s "$preimagen" ]] || {
  echo 'preimagen activa unica ausente o ambigua' >&2; exit 1;
}
(cd "$repo_dir" && go run ./deploy/principal/p6_retirada_dietas < "$preimagen") > "$variables"

finalizar=ROLLBACK
[[ "$modo" == --commit ]] && finalizar=COMMIT
tx_estado=0
if { cat "$variables" "$base_dir/transaccion.sql"; } | \
    psql_ejecutar -v "p6_finalizar=$finalizar" >/dev/null; then
  :
else
  tx_estado=$?
fi

post="$evidencia_dir/p6-post-$marca-$$.json"
if ! psql_ejecutar < "$base_dir/inventario.sql" > "$post"; then
  echo 'postinventario inaccesible: resultado transaccional incierto; no reintentar sin inspeccion DBA' >&2
  exit 1
fi
if [[ "$tx_estado" -ne 0 ]]; then
  echo "transporte/transaccion falló (código $tx_estado); revisar inventario privado $post antes de cualquier reintento" >&2
  exit 1
fi
python3 - "$post" "$finalizar" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as handle:
    estado = json.load(handle)
roles = estado["roles"]
actuales = [a for a in estado["asignaciones"] if a["puntero"]]
confirmado = sys.argv[2] == "COMMIT"
if (len(roles) != 8 or estado["sesiones"] or len(actuales) != 1
        or any(r["login"] == confirmado for r in roles)
        or (confirmado and any(r["grupo"] in {"vec_dietas_ejecutor", "vec_dietas_registrador_frontera"}
                               for r in estado["login_con_grupo"]))
        or actuales[0]["estado"] != ("revocada" if confirmado else "activa")):
    raise SystemExit("postcondicion P6 fallida; revisar inventario privado (COMMIT puede estar confirmado)")
PY
echo "resultado=$finalizar inventario_post_privado=$post"
