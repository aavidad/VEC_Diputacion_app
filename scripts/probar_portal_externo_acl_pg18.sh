#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
ensayo_dir=$(mktemp -d /var/tmp/vec-portal-acl-XXXXXX)
contenedor="vec-portal-acl-$$"
limpiar() {
  docker stop "$contenedor" >/dev/null 2>&1 || true
  docker run --rm --network none -v "$ensayo_dir:/ensayo" alpine:3.22 rm -rf /ensayo/datos /ensayo/tls
  rmdir "$ensayo_dir"
}
trap limpiar EXIT
mkdir "$ensayo_dir/tls" "$ensayo_dir/datos"
openssl req -x509 -newkey rsa:2048 -nodes -days 1 -subj /CN=localhost \
  -addext 'subjectAltName=DNS:localhost,IP:127.0.0.1' \
  -keyout "$ensayo_dir/tls/server.key" -out "$ensayo_dir/tls/server.crt" >/dev/null 2>&1
chmod 600 "$ensayo_dir/tls/server.key"
docker run --rm --network none -v "$ensayo_dir/tls:/tls" alpine:3.22 chown 999:999 /tls/server.key
docker run -d --rm --restart=no --memory 2g --name "$contenedor" \
  -p 127.0.0.1::5432 -e POSTGRES_HOST_AUTH_METHOD=trust \
  -v "$ensayo_dir/datos:/var/lib/postgresql" -v "$ensayo_dir/tls:/tls:ro" \
  postgres:18.4 -c ssl=on -c ssl_cert_file=/tls/server.crt -c ssl_key_file=/tls/server.key >/dev/null
for ((i=0; i<60; i++)); do
  if docker exec "$contenedor" pg_isready -U postgres >/dev/null 2>&1; then break; fi
  sleep 1
done
puerto=$(docker port "$contenedor" 5432/tcp)
puerto=${puerto##*:}
export VEC_PORTAL_ACL_PG18_DESECHABLE=1
export VEC_PORTAL_ACL_PG18_ADMIN_DSN="host=127.0.0.1 port=$puerto user=postgres dbname=postgres sslmode=verify-full sslrootcert=$ensayo_dir/tls/server.crt"
export VEC_PORTAL_ACL_PG18_EXTERNO_DSN="host=127.0.0.1 port=$puerto user=vec_externo_bolsa_desarrollo dbname=postgres sslmode=verify-full sslrootcert=$ensayo_dir/tls/server.crt"
export GOCACHE="$HOME/.cache/go-build"
go test -p 6 ./internal/app/bootstrap -run '^TestPortalExternoACLPostgreSQL18$' -count=1 -v
