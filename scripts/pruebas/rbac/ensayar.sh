#!/usr/bin/env bash
set -euo pipefail
umask 077
here=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
fail() { printf '%s\n' "$*" >&2; exit 1; }
[[ $# -ge 2 ]] || fail 'Uso: ensayar.sh ROOT init SOURCE | snapshot DEST | apply SQL SHA PRE POST | status'
root=$(realpath -m -- "$1"); action=$2; shift 2
[[ $root == /dev/shm/vec-rbac-* || $root == /dev/shm/vec-codexk-rbac-* ]] || fail 'ROOT debe ser un directorio propio vec-rbac-* en /dev/shm'
name=$(basename -- "$root")
[[ $name =~ ^[a-z0-9-]+$ ]] || fail 'Nombre de clon inválido'
psql_clone() { docker exec -i "$expected_id" psql -X -q -h /tmp -U postgres -d postgres -At -v ON_ERROR_STOP=1 "$@"; }
probe_clone() { { printf 'BEGIN READ ONLY;\n'; cat -- "$1"; printf '\nCOMMIT;\n'; } | psql_clone; }
verify_clone() {
  [[ -d $root && $(stat -c '%u' -- "$root") == "$(id -u)" ]] || fail 'ROOT no pertenece al ejecutor'
  [[ -f $root/container.id ]] || fail 'Falta el ID del contenedor creado'
  expected_id=$(cat -- "$root/container.id")
  [[ $expected_id =~ ^[a-f0-9]{64}$ ]] || fail 'ID registrado inválido'
  [[ $(docker inspect -f '{{.Id}}' "$name") == "$expected_id" ]] || fail 'Contenedor distinto del creado para este clon'
  [[ $(docker inspect -f '{{.HostConfig.NetworkMode}}' "$expected_id") == none ]] || fail 'El clon tiene red'
  [[ -f $root/mounts.json && $(docker inspect -f '{{json .Mounts}}' "$expected_id") == "$(cat -- "$root/mounts.json")" ]] || fail 'Los montajes difieren de los registrados al crear el clon'
  [[ $(docker inspect -f '{{range .Mounts}}{{if eq .Type "bind"}}{{.Type}}|{{.Source}}|{{.Destination}}|{{.RW}}{{end}}{{end}}' "$expected_id") == "bind|$root|/ensayo|true" ]] || fail 'Montaje distinto del clon propio'
  [[ $(docker inspect -f '{{.HostConfig.AutoRemove}}' "$expected_id") == true ]] || fail 'Contenedor sin --rm'
  [[ $(psql_clone -c 'SHOW data_directory') == /ensayo/vec-desarrollo-20260906/pgdata ]] || fail 'Directorio de datos distinto del clon propio'
  [[ $(psql_clone -c "SELECT current_setting('server_version_num')::integer / 10000") == 18 ]] || fail 'Se requiere PostgreSQL 18'
}
case $action in
init)
  [[ $# == 1 && -f $1 && ! -e $root ]] || fail 'Se requiere SOURCE regular y ROOT nuevo'
  mkdir -m 751 -- "$root"
  python3 - "$1" "$root" <<'PY'
import hashlib,pathlib,sys,tarfile
source=pathlib.Path(sys.argv[1]); root=pathlib.Path(sys.argv[2])
prefix='vec-desarrollo-20260906/pgdata/'
with tarfile.open(source) as archive:
    members=archive.getmembers()
    for m in members:
        if m.name.startswith('/') or '..' in pathlib.PurePosixPath(m.name).parts or not (m.isfile() or m.isdir()):
            raise SystemExit('Archivo TAR no admitido')
    if archive.extractfile(prefix+'PG_VERSION').read().strip()!=b'18':
        raise SystemExit('La fuente no es PG18')
    for m in members:
        if m.name.startswith(prefix) or m.name==prefix.rstrip('/'):
            archive.extract(m,root,filter='data')
(root/'source.sha256').write_text(hashlib.file_digest(source.open('rb'),'sha256').hexdigest()+'\n')
(root/'postgresql.conf').write_text("listen_addresses=''\nport=5432\nunix_socket_directories='/tmp'\nssl=off\nhba_file='/ensayo/pg_hba.conf'\nident_file='/ensayo/pg_ident.conf'\nmax_connections=20\nshared_buffers='128MB'\n")
(root/'pg_hba.conf').write_text('local all all trust\n')
(root/'pg_ident.conf').write_text('')
for filename in ('postgresql.conf','pg_hba.conf','pg_ident.conf'):
    (root/filename).chmod(0o644)
PY
  docker run -d --rm --name "$name" --network none --memory 2g --cpus 2 --pids-limit 128 \
    -v "$root:/ensayo" --entrypoint sh postgres:18.4 \
    -c 'chown -R 999:999 /ensayo/vec-desarrollo-20260906/pgdata && exec gosu postgres postgres -D /ensayo/vec-desarrollo-20260906/pgdata -c config_file=/ensayo/postgresql.conf' > "$root/container.id"
  docker inspect -f '{{json .Mounts}}' "$(cat -- "$root/container.id")" > "$root/mounts.json"
  for ((i=0;i<30;i++)); do
    if docker exec "$name" pg_isready -h /tmp >/dev/null 2>&1; then verify_clone; printf 'CLON-OK\n'; exit; fi
    sleep 1
  done
  fail 'Clon no disponible en 30 segundos; consultar logs de este contenedor'
  ;;
status)
  [[ $# == 0 ]] || fail 'status no admite argumentos'
  verify_clone; printf 'CLON-OK PG18 sin red\n'
  ;;
snapshot)
  [[ $# == 1 ]] || fail 'snapshot requiere DEST nuevo'
  verify_clone
  [[ ! -e $1 ]] || fail 'DEST ya existe'
  (set -o noclobber; psql_clone < "$here/preimagen.sql" > "$1")
  sha256sum -- "$1"
  ;;
apply)
  [[ $# == 4 ]] || fail 'apply requiere SQL SHA PRE POST'
  sql=$1; expected=$2; pre=$3; post=$4
  [[ -f $sql && $sql == *.up.sql && $expected =~ ^[a-f0-9]{64}$ && -f $pre && -f $post ]] || fail 'Artefactos de migración inválidos'
  [[ $(sha256sum -- "$sql" | cut -d ' ' -f1) == "$expected" ]] || fail 'Huella SQL distinta'
  verify_clone
  mkdir -p -- "$root/intentos"
  key=$(basename -- "$sql")
  [[ ! -e $root/intentos/$key ]] || fail 'Migración ya intentada en este clon; conservar fallo'
  [[ $(probe_clone "$pre") == missing ]] || fail 'Precondición no acredita SQL faltante'
  (set -o noclobber; printf '%s\n' "$expected" > "$root/intentos/$key")
  cp -- "$sql" "$root/intentos/$key.sql"
  [[ $(sha256sum -- "$root/intentos/$key.sql" | cut -d ' ' -f1) == "$expected" ]] || fail 'La fuente SQL cambió; conservar intento sin ejecutar'
  psql_clone < "$root/intentos/$key.sql" > "$root/intentos/$key.log" 2>&1 || fail 'SQL falló; consultar log privado propio, no repetir sobre este clon'
  [[ $(probe_clone "$post") == installed ]] || fail 'Postcondición no acredita instalación'
  printf 'installed\n' > "$root/intentos/$key.result"
  printf 'ENSAYO-OK %s\n' "$key"
  ;;
*) fail 'Acción desconocida';;
esac
