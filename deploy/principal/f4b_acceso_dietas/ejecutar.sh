#!/usr/bin/env bash
# F4b: acceso completo de Dietas (ocho cuentas R1D + tres nuevas). Ver README.
set -euo pipefail
umask 077

modo="${1:-}"
case "$modo" in
  --preparar-estado|--inventario|--rollback|--commit|--sonda-tls|--retirar-rollback|--retirar-commit) ;;
  *) echo 'uso: ejecutar.sh --preparar-estado|--inventario|--rollback|--commit|--sonda-tls|--retirar-rollback|--retirar-commit' >&2; exit 2 ;;
esac
if [[ "$modo" == --commit || "$modo" == --retirar-commit ]] && [[ "${VEC_F4B_APLICAR:-}" != 'SI-F4B-REVISADO' ]]; then
  echo 'COMMIT exige VEC_F4B_APLICAR=SI-F4B-REVISADO' >&2
  exit 2
fi

base_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
repo_dir="$(cd "$base_dir/../../.." && pwd -P)"

# Estado privado con las contraseñas de las cuentas nuevas: fichero regular,
# del ejecutor, modo 0600 y fuera de Git. Nunca se imprime ni viaja por argv.
estado_privado() {
  : "${VEC_F4B_ESTADO:?falta VEC_F4B_ESTADO (estado privado 0600 fuera de Git)}"
  local directorio
  directorio="$(realpath -e "$(dirname "$VEC_F4B_ESTADO")")"
  case "$directorio/" in "$repo_dir/"*) echo 'estado F4b dentro de Git' >&2; exit 2;; esac
  estado="$directorio/$(basename "$VEC_F4B_ESTADO")"
}
validar_estado() {
  [[ -f "$estado" && ! -L "$estado" && "$(stat -c '%a' "$estado")" == 600
      && "$(stat -c '%u' "$estado")" == "$(id -u)" ]] || {
    echo 'estado F4b ausente, enlace, ajeno o sin modo 0600' >&2; exit 2;
  }
}

if [[ "$modo" == --preparar-estado ]]; then
  estado_privado
  if [[ -e "$estado" || -L "$estado" ]]; then
    echo 'el estado F4b ya existe: no se sobrescribe ni se rota' >&2; exit 2
  fi
  python3 - "$estado" <<'PY'
import json, os, secrets, sys
ruta = sys.argv[1]
nuevas = ["vec_dietas_f4b_auditoria_frontera_desarrollo", "vec_personal_d7_asignacion",
          "vec_personal_d7_auditoria_frontera"]
datos = {"version": 1, "contrasenas": {n: secrets.token_urlsafe(36) for n in nuevas}}
fd = os.open(ruta, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
with os.fdopen(fd, "w", encoding="utf-8") as salida:
    json.dump(datos, salida, indent=1, sort_keys=True)
    salida.write("\n")
PY
  echo "estado privado creado (0600): $estado"
  exit 0
fi

: "${VEC_F4B_EVIDENCIA_DIR:?falta VEC_F4B_EVIDENCIA_DIR privado}"
evidencia_dir="$(realpath -e "$VEC_F4B_EVIDENCIA_DIR")"
case "$evidencia_dir/" in "$repo_dir/"*) echo 'evidencia dentro de Git' >&2; exit 2;; esac
[[ -d "$evidencia_dir" && ! -L "$VEC_F4B_EVIDENCIA_DIR" ]] || { echo 'directorio de evidencia invalido' >&2; exit 2; }
[[ "$(stat -c '%a' "$evidencia_dir")" == 700 ]] || { echo 'evidencia debe ser modo 0700' >&2; exit 2; }
[[ "$(stat -c '%u' "$evidencia_dir")" == "$(id -u)" ]] || { echo 'evidencia debe pertenecer al ejecutor' >&2; exit 2; }

# Lectura y validación del estado; imprime JSON {nombre: contraseña} a stdout
# solo hacia otra tubería del propio script, nunca al terminal.
leer_estado_py='
import json, re, sys
cuentas = {"vec_dietas_r1d_registro_identidad_desarrollo","vec_dietas_r1d_revalidacion_identidad_desarrollo",
 "vec_dietas_r1d_contexto_desarrollo","vec_dietas_r1d_fuente_autorizacion_desarrollo",
 "vec_dietas_r1d_registro_autorizacion_desarrollo","vec_dietas_r1d_motivos_desarrollo",
 "vec_dietas_r1d_dietas_desarrollo","vec_dietas_r1d_personal_desarrollo",
 "vec_dietas_f4b_auditoria_frontera_desarrollo","vec_personal_d7_asignacion","vec_personal_d7_auditoria_frontera"}
nuevas = {"vec_dietas_f4b_auditoria_frontera_desarrollo","vec_personal_d7_asignacion","vec_personal_d7_auditoria_frontera"}
with open(sys.argv[1], encoding="utf-8") as f:
    datos = json.load(f)
c = datos.get("contrasenas") if isinstance(datos, dict) else None
if (datos.get("version") != 1 or set(datos) != {"version", "contrasenas"} or not isinstance(c, dict)
        or not nuevas <= set(c) or not set(c) <= cuentas
        or any(not isinstance(v, str) or not re.fullmatch(r"[A-Za-z0-9_-]{32,128}", v) for v in c.values())):
    raise SystemExit("estado F4b invalido: version 1, tres cuentas nuevas y contraseñas [A-Za-z0-9_-]{32,128}")
'

if [[ "$modo" == --sonda-tls ]]; then
  estado_privado
  validar_estado
  : "${VEC_F4B_TLS_HOST:?falta VEC_F4B_TLS_HOST (nombre del certificado del servidor)}"
  : "${VEC_F4B_TLS_PORT:?falta VEC_F4B_TLS_PORT}"
  : "${VEC_F4B_TLS_CA:?falta VEC_F4B_TLS_CA (CA privada del servidor)}"
  [[ "$VEC_F4B_TLS_HOST" =~ ^[A-Za-z0-9][A-Za-z0-9.-]{0,252}$ && "$VEC_F4B_TLS_PORT" =~ ^[1-9][0-9]{0,4}$ \
     && -f "$VEC_F4B_TLS_CA" && ! -L "$VEC_F4B_TLS_CA" ]] || { echo 'parametros TLS F4b invalidos' >&2; exit 2; }
  command -v psql >/dev/null || { echo 'falta psql para la sonda TLS' >&2; exit 2; }
  for nombre in PGSERVICE PGSERVICEFILE PGPASSFILE PGHOST PGHOSTADDR PGPORT PGDATABASE PGUSER PGPASSWORD PGOPTIONS PGSSLMODE PGSSLROOTCERT; do
    if [[ -n "${!nombre+x}" ]]; then echo "variable PG incompatible con la sonda: $nombre" >&2; exit 2; fi
  done
  pgpass="$(mktemp "$evidencia_dir/.f4b-pgpass.XXXXXXXX")"
  cuentas_sonda="$(mktemp "$evidencia_dir/.f4b-cuentas.XXXXXXXX")"
  trap 'rm -f -- "$pgpass" "$cuentas_sonda"' EXIT
  # shellcheck disable=SC2016 # código Python literal, sin expansión de shell
  python3 -c "$leer_estado_py"'
host, puerto = sys.argv[2], sys.argv[3]
with open(sys.argv[4], "w", encoding="utf-8") as p, open(sys.argv[5], "w", encoding="utf-8") as n:
    for nombre in sorted(c):
        p.write(f"{host}:{puerto}:postgres:{nombre}:{c[nombre]}\n")
        n.write(nombre + "\n")
' "$estado" "$VEC_F4B_TLS_HOST" "$VEC_F4B_TLS_PORT" "$pgpass" "$cuentas_sonda"
  # require_auth (libpq >= 16) impide que la sonda positiva se dé por buena si
  # el servidor autentica por trust, password o md5 en vez de SCRAM. La sonda
  # negativa no lo lleva: cualquier conexión sin TLS, sea cual sea el método,
  # debe contar como fallo y no quedar oculta por un rechazo del cliente.
  libpq_prueba="$(LC_ALL=C LANG=C psql -XAtq -w 'dbname=postgres require_auth=scram-sha-256 host=/nonexistent' -c '' </dev/null 2>&1 || true)"
  if [[ "$libpq_prueba" == *'invalid connection option'* || "$libpq_prueba" == *'opción de conexión'* ]]; then
    echo 'libpq sin require_auth (>= 16) para la sonda TLS' >&2; exit 2
  fi
  fallos=0
  while IFS= read -r cuenta; do
    base="host=$VEC_F4B_TLS_HOST port=$VEC_F4B_TLS_PORT dbname=postgres user=$cuenta connect_timeout=5"
    if ! salida="$(PGPASSFILE="$pgpass" psql -XAtq -w "$base sslmode=verify-full sslrootcert=$VEC_F4B_TLS_CA require_auth=scram-sha-256" \
         -c "SELECT session_user||'|'||(SELECT ssl::text FROM pg_stat_ssl WHERE pid=pg_backend_pid())" </dev/null 2>/dev/null)" \
       || [[ "$salida" != "$cuenta|true" ]]; then
      echo "sonda TLS: $cuenta no conecta con verify-full" >&2; fallos=1; continue
    fi
    if PGPASSFILE="$pgpass" psql -XAtq -w "$base sslmode=disable" -c 'SELECT 1' </dev/null >/dev/null 2>&1; then
      echo "sonda TLS: $cuenta conecta SIN TLS; pg_hba debe exigir hostssl" >&2; fallos=1; continue
    fi
    echo "sonda TLS: $cuenta verify-full OK y sin TLS rechazado"
  done < "$cuentas_sonda"
  exit "$fallos"
fi

transporte=servicio
if [[ -n "${VEC_F4B_POSTGRES_CONTAINER:-}" ]]; then
  transporte=contenedor
  [[ "$VEC_F4B_POSTGRES_CONTAINER" =~ ^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$ ]] || {
    echo 'nombre de contenedor F4b invalido' >&2; exit 2;
  }
  : "${VEC_F4B_PG_SOCKET_DIR:?falta socket PostgreSQL privado}"
  : "${VEC_F4B_PG_PORT:?falta puerto PostgreSQL privado}"
  [[ "$VEC_F4B_PG_SOCKET_DIR" =~ ^/[A-Za-z0-9_./-]+$ \
     && "$VEC_F4B_PG_SOCKET_DIR" != *..* \
     && "$VEC_F4B_PG_PORT" =~ ^[1-9][0-9]{0,4}$ \
     && "$VEC_F4B_PG_PORT" -le 65535 ]] || {
    echo 'socket o puerto PostgreSQL F4b invalidos' >&2; exit 2;
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
    podman exec -i "$VEC_F4B_POSTGRES_CONTAINER" \
      env -i PATH=/usr/local/bin:/usr/bin:/bin \
      psql -XAtq -w -v ON_ERROR_STOP=1 \
      -h "$VEC_F4B_PG_SOCKET_DIR" -p "$VEC_F4B_PG_PORT" \
      -U postgres -d postgres "$@"
  else
    psql -XAtq -v ON_ERROR_STOP=1 "$@"
  fi
}

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
  echo 'F4b exige PostgreSQL 18, socket local, base postgres y sesion DBA' >&2; exit 2;
}

marca="$(date -u +%Y%m%dT%H%M%SZ)"
inventario="$evidencia_dir/f4b-inventario-$marca-$$.json"
psql_ejecutar < "$base_dir/inventario.sql" > "$inventario"
echo "inventario privado: $inventario"
if [[ "$modo" == --inventario ]]; then exit 0; fi

preimagen="$(mktemp "$evidencia_dir/.f4b-preimagen.XXXXXXXX")"
variables="$(mktemp "$evidencia_dir/.f4b-variables.XXXXXXXX")"
trap 'rm -f -- "$preimagen" "$variables"' EXIT
retirada=false
[[ "$modo" == --retirar-* ]] && retirada=true
preimagen_sql="$base_dir/preimagen.sql"
transaccion_sql="$base_dir/transaccion.sql"
plan_args=()
finalizar_var=f4b_finalizar
if "$retirada"; then
  preimagen_sql="$base_dir/preimagen_retirada.sql"
  transaccion_sql="$base_dir/retirada.sql"
  plan_args=(--retirar)
  finalizar_var=f4b_retirada_finalizar
fi
psql_ejecutar < "$preimagen_sql" > "$preimagen"
[[ "$(wc -l < "$preimagen")" -eq 1 && -s "$preimagen" ]] || {
  echo 'preimagen F4b unica ausente o ambigua' >&2; exit 1;
}
(cd "$repo_dir" && go run ./deploy/principal/f4b_acceso_dietas "${plan_args[@]}" < "$preimagen") > "$variables"

if ! "$retirada"; then
  # Verificadores SCRAM-SHA-256 (RFC 7677, 4096 iteraciones) de las tres
  # cuentas nuevas: el servidor recibe el verificador, nunca la contraseña.
  estado_privado
  validar_estado
  # shellcheck disable=SC2016 # código Python literal, sin expansión de shell
  python3 -c "$leer_estado_py"'
import base64, hashlib, hmac, os
def scram(clave):
    sal = os.urandom(16)
    salada = hashlib.pbkdf2_hmac("sha256", clave.encode(), sal, 4096)
    cliente = hmac.new(salada, b"Client Key", "sha256").digest()
    servidor = hmac.new(salada, b"Server Key", "sha256").digest()
    b = lambda x: base64.b64encode(x).decode()
    return f"SCRAM-SHA-256$4096:{b(sal)}${b(hashlib.sha256(cliente).digest())}:{b(servidor)}"
verificadores = {n: scram(c[n]) for n in sorted(nuevas)}
print("\\set f4b_verificadores_b64 " + base64.b64encode(json.dumps(verificadores).encode()).decode())
' "$estado" >> "$variables"
fi

finalizar=ROLLBACK
[[ "$modo" == --commit || "$modo" == --retirar-commit ]] && finalizar=COMMIT
tx_estado=0
if { cat "$variables" "$base_dir/comun.sql" "$transaccion_sql"; } | \
    psql_ejecutar -v "$finalizar_var=$finalizar" >/dev/null; then
  :
else
  tx_estado=$?
fi

post="$evidencia_dir/f4b-post-$marca-$$.json"
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
actor_esperado = ("administracion:f4b:retirada-acceso-dietas" if confirmado else "administracion:f4b:acceso-dietas") if retirada else ("administracion:f4b:acceso-dietas" if confirmado else "administracion:p6:retirada-dietas-r1d")
# Tras ROLLBACK de la activación pueden faltar las tres cuentas nuevas.
esperadas = 11 if (confirmado or retirada) else len(roles)
if (len(roles) != esperadas or not 8 <= len(roles) <= 11 or estado["sesiones"] or len(actuales) != 1
        or estado["r1d_prefijo"] != 8
        or any(r["login"] != login_esperado for r in roles)
        or actuales[0]["estado"] != estado_esperado
        or actuales[0]["version"] != version_esperada
        or actuales[0]["actualizada_por"] != actor_esperado):
    raise SystemExit("postcondicion F4b fallida; revisar inventario privado (COMMIT puede estar confirmado)")
PY
echo "resultado=$finalizar inventario_post_privado=$post"
