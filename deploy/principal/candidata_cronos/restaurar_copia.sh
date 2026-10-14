#!/usr/bin/env bash
set -Eeuo pipefail
umask 077

# Copia local desechable. Nunca apunta a la principal ni instala migraciones.
state_dir=${VEC_CRONOS_STATE_DIR:?Indique una carpeta privada nueva para esta copia}
subnet=${VEC_CRONOS_SUBNET:-10.234.51.0/28}
container=vec-codexc-cronos-pg-20261014
network=vec-codexc-cronos-net-20261014
port=55451
modo=${1:-restaurar}

arrancar_contenedor() {
  chmod 644 "$state_dir/postgres_password"
  trap 'chmod 600 "$state_dir/postgres_password"' EXIT
  docker run -d --name "$container" --rm --restart=no --memory=2g \
    --network "$network" -p "127.0.0.1:$port:5432" \
    -v "$state_dir/pgdata:/var/lib/postgresql" \
    -v "$state_dir/postgres_password:/run/secrets/postgres_password:ro" \
    -e POSTGRES_PASSWORD_FILE=/run/secrets/postgres_password postgres:18 >/dev/null
  for _ in {1..60}; do
    docker exec -u postgres "$container" pg_isready -U postgres -d postgres >/dev/null 2>&1 && break
    sleep 1
  done
  docker exec -u postgres "$container" pg_isready -U postgres -d postgres >/dev/null
  chmod 600 "$state_dir/postgres_password"
  trap - EXIT
}

[[ $state_dir = /* && ! -L $state_dir ]] || { echo 'Destino privado no válido' >&2; exit 2; }
if [[ $modo == reanudar ]]; then
  [[ -d $state_dir/pgdata && -f $state_dir/postgres_password ]] || { echo 'La copia privada está incompleta' >&2; exit 2; }
  docker container inspect "$container" >/dev/null 2>&1 && { echo 'El contenedor propio ya existe' >&2; exit 2; }
  docker network inspect "$network" >/dev/null 2>&1 || { echo 'Falta la red propia' >&2; exit 2; }
  arrancar_contenedor
  echo 'PostgreSQL de la copia reanudado; comprobar TLS y el recorrido antes de darla por apta.'
  exit 0
fi
[[ $modo == restaurar ]] || { echo 'Modo no admitido' >&2; exit 2; }
backup_dir=${VEC_CRONOS_BACKUP_DIR:?Indique la carpeta privada de la copia fuente}
[[ $backup_dir = /* ]] || { echo 'La carpeta de origen debe ser absoluta' >&2; exit 2; }
[[ ! -e $state_dir && ! -L $state_dir ]] || { echo 'La carpeta de destino ya existe' >&2; exit 2; }
for file in principal-hz37-20261014.dump principal-globales-20261014.sql acl-tipos-20261014.sql acl-base-20261014.sql material-cidonia-20261014.tgz; do
  [[ -f $backup_dir/$file ]] || { echo "Falta el artefacto: $file" >&2; exit 2; }
done
python3 - "$backup_dir" <<'PY'
from pathlib import Path
import hashlib, sys
root = Path(sys.argv[1])
expected = {
    'principal-hz37-20261014.dump': '7ae0613fa174aebc8aa36c079a6c4b268b771df41c9310129ee9dfae9650cc45',
    'principal-globales-20261014.sql': '95d943a3aab12dfe07895bce4b8b6928457cd6896b128537c53794a787d597ec',
    'acl-tipos-20261014.sql': 'e058a00b9e0c8c9f06c2fc76d33882637ab7fb715618282305237c1ea959ea71',
    'acl-base-20261014.sql': 'ab21b92e6f81a838c8c502d431de66454e8c47e7d7fca0becc01882d6d3584aa',
    'material-cidonia-20261014.tgz': '699f7c9ff2d800003ccd088167d835261da2abda981e59fdea0745f810cda8e9',
}
for name, digest in expected.items():
    h = hashlib.sha256()
    with (root / name).open('rb') as source:
        for block in iter(lambda: source.read(1024 * 1024), b''):
            h.update(block)
    if h.hexdigest() != digest:
        raise SystemExit(f'La huella no coincide: {name}')
PY
docker container inspect "$container" >/dev/null 2>&1 && { echo 'El contenedor propio ya existe' >&2; exit 2; }
docker network inspect "$network" >/dev/null 2>&1 && { echo 'La red propia ya existe' >&2; exit 2; }
python3 - "$port" <<'PY'
import socket, sys
with socket.socket() as sock:
    try:
        sock.bind(('127.0.0.1', int(sys.argv[1])))
    except OSError:
        raise SystemExit('El puerto asignado está ocupado')
PY

install -d -m 700 "$state_dir" "$state_dir/pgdata" "$state_dir/material-restaurado"
# La imagen PostgreSQL ajusta los permisos internos; la carpeta padre sigue privada.
chmod 777 "$state_dir/pgdata"
openssl rand -base64 36 > "$state_dir/postgres_password"
# El proceso postgres del contenedor debe leer el fichero montado en solo lectura.
# La carpeta padre tiene modo 0700 en el host.
chmod 644 "$state_dir/postgres_password"
python3 - "$backup_dir/principal-globales-20261014.sql" "$state_dir/globales-restauracion.sql" <<'PY'
from pathlib import Path
import sys
source = Path(sys.argv[1]).read_text()
line = 'CREATE ROLE postgres;\n'
if source.count(line) != 1:
    raise SystemExit('No se encontró una única creación del rol postgres')
Path(sys.argv[2]).write_text(source.replace(line, '-- Rol postgres creado por la imagen.\n'))
PY

# El material de la copia se extrae solo dentro del destino privado.
python3 - "$backup_dir/material-cidonia-20261014.tgz" "$state_dir/material-restaurado" <<'PY'
from pathlib import Path
import sys, tarfile
root = Path(sys.argv[2]).resolve()
with tarfile.open(sys.argv[1], 'r:gz') as archive:
    for member in archive.getmembers():
        target = (root / member.name).resolve()
        if not target.is_relative_to(root) or not (member.isfile() or member.isdir()):
            raise SystemExit('El archivo de material contiene una entrada no admitida')
    archive.extractall(root, filter='data')
PY

python3 - "$state_dir" <<'PY'
from pathlib import Path
from urllib.parse import parse_qsl, urlencode, urlsplit
import json, shlex, shutil, sys

state = Path(sys.argv[1])
sources = list((state / 'material-restaurado').rglob('vec-desarrollo-20260906'))
if len(sources) != 1:
    raise SystemExit('La copia no contiene una única raíz de material de desarrollo')
source = sources[0]
runtime = state / 'runtime'
if not source.is_dir():
    raise SystemExit('Falta la estructura esperada del material privado')
shutil.copytree(source, runtime)
certs = runtime / 'configuracion-postgresql'
tls = state / 'pg-tls'
tls.mkdir(mode=0o700)
for name in ('server.crt', 'server.key', 'o207-ca.crt'):
    shutil.copy2(certs / name, tls / name)
    (tls / name).chmod(0o600)

old_root = '/' + str(source.relative_to(state / 'material-restaurado'))
def replace(value):
    if isinstance(value, dict):
        return {key: replace(item) for key, item in value.items()}
    if isinstance(value, list):
        return [replace(item) for item in value]
    if not isinstance(value, str):
        return value
    value = value.replace(old_root, str(runtime))
    if not value.startswith(('postgresql://', 'postgres://')):
        return value
    url = urlsplit(value)
    if url.hostname not in ('localhost', '127.0.0.1') or url.port != 5432 or url.path != '/postgres':
        raise SystemExit('Conexión privada con destino inesperado')
    query = dict(parse_qsl(url.query, keep_blank_values=True))
    if query.get('sslmode') != 'verify-full':
        raise SystemExit('Conexión privada sin verify-full')
    query['sslrootcert'] = str(tls / 'o207-ca.crt')
    userinfo = url.netloc.rpartition('@')[0]
    return url._replace(netloc=userinfo + '@localhost:55451', query=urlencode(query)).geturl()

for path in runtime.rglob('*.json'):
    try:
        original = json.loads(path.read_text())
    except (UnicodeDecodeError, json.JSONDecodeError):
        continue
    path.write_text(json.dumps(replace(original), ensure_ascii=False, indent=2) + '\n')
for path in runtime.rglob('*.dsn'):
    path.write_text(replace(path.read_text().strip()) + '\n')
for path in runtime.rglob('*.env'):
    path.write_text(path.read_text().replace(old_root, str(runtime)))
for path in runtime.rglob('*'):
    path.chmod(0o600 if path.is_file() else 0o700)

material = runtime / 'material/desarrollo-fiscalizacion-20260904'
env = {}
for line in (material / 'desarrollo.env').read_text().splitlines():
    if '=' in line and not line.lstrip().startswith('#'):
        key, value = line.split('=', 1)
        env[key.strip()] = value.strip()
env['VEC_DEVELOPMENT_MATERIAL_DIR'] = str(material)
env['VEC_TLS_CERT_FILE'] = str(material / 'tls/servidor.crt')
env['VEC_TLS_KEY_FILE'] = str(material / 'tls/servidor.key')
env['VEC_HTTP_ADDR'] = '127.0.0.1:18651'
for key in ('VEC_DEVELOPMENT_MATERIAL_DIR', 'VEC_TLS_CERT_FILE', 'VEC_TLS_KEY_FILE'):
    if not Path(env[key]).exists():
        raise SystemExit('Falta un artefacto privado de arranque')
output = state / 'arranque-base-ct.env'
output.write_text(''.join(f'{key}={shlex.quote(value)}\n' for key, value in env.items()))
output.chmod(0o600)
PY

docker network create --subnet "$subnet" "$network" >/dev/null
arrancar_contenedor
docker exec -i -u postgres "$container" psql -X -v ON_ERROR_STOP=1 -U postgres -d postgres \
  < "$state_dir/globales-restauracion.sql" > "$state_dir/restaurar-globales.log" 2>&1
docker exec -i -u postgres "$container" pg_restore -U postgres -d postgres --exit-on-error \
  < "$backup_dir/principal-hz37-20261014.dump" > "$state_dir/restaurar-dump.log" 2>&1
docker exec -i -u postgres "$container" psql -X -v ON_ERROR_STOP=1 -U postgres -d postgres \
  < "$backup_dir/acl-tipos-20261014.sql" > "$state_dir/restaurar-acl.log" 2>&1
docker exec -i -u postgres "$container" psql -X -v ON_ERROR_STOP=1 -U postgres -d postgres \
  < "$backup_dir/acl-base-20261014.sql" > "$state_dir/restaurar-acl-base.log" 2>&1
docker cp "$state_dir/pg-tls/server.crt" "$container:/var/lib/postgresql/18/docker/server.crt" >/dev/null
docker cp "$state_dir/pg-tls/server.key" "$container:/var/lib/postgresql/18/docker/server.key" >/dev/null
docker exec -u 0 "$container" chown postgres:postgres /var/lib/postgresql/18/docker/server.crt /var/lib/postgresql/18/docker/server.key
docker exec -u 0 "$container" chmod 600 /var/lib/postgresql/18/docker/server.key
docker exec -u postgres "$container" sh -c "printf '\nssl = on\nssl_cert_file = '\''server.crt'\''\nssl_key_file = '\''server.key'\''\n' >> /var/lib/postgresql/18/docker/postgresql.conf"
docker exec -u postgres "$container" pg_ctl -D /var/lib/postgresql/18/docker reload >/dev/null
(cd "$backup_dir" && sha256sum principal-hz37-20261014.dump principal-globales-20261014.sql acl-tipos-20261014.sql acl-base-20261014.sql material-cidonia-20261014.tgz) > "$state_dir/hashes-fuente.txt"
echo 'Copia local restaurada; los registros y las huellas quedan en la carpeta privada indicada.'
