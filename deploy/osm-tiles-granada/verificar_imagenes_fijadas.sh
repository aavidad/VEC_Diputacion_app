#!/usr/bin/env bash
set -Eeuo pipefail

imagenes=(
  "ghcr.io/systemed/tilemaker:master@sha256:d32505d7827907089c2dd07517524276946d8930b4b82e93cf5c25ec989bbe41"
  "maptiler/tileserver-gl:v5.6.0@sha256:3a9ccdb24820b6814c8119bcc8a4376c39867cb0ffe69d62919ef898b90c2427"
  "nginx:1.27-alpine@sha256:65645c7bb6a0661892a8b03b89d0743208a18dd2f3f17a54ef4b76fb8e2f2a10"
)

if ! command -v docker >/dev/null 2>&1; then
  echo "ERROR: Docker no esta disponible." >&2
  exit 1
fi

for imagen in "${imagenes[@]}"; do
  if ! docker image inspect "$imagen" >/dev/null 2>&1; then
    echo "ERROR: no esta precargada la imagen exacta: $imagen" >&2
    echo "Use el registro interno aprobado; este script no descarga ni sustituye imagenes." >&2
    exit 1
  fi
  echo "OK: imagen exacta presente: $imagen"
done

imagen_teselas="maptiler/tileserver-gl:v5.6.0@sha256:3a9ccdb24820b6814c8119bcc8a4376c39867cb0ffe69d62919ef898b90c2427"
uid_node="$(docker run --rm --network none --entrypoint /usr/bin/id "$imagen_teselas" -u node)"
gid_node="$(docker run --rm --network none --entrypoint /usr/bin/id "$imagen_teselas" -g node)"
if [[ "$uid_node:$gid_node" != "999:999" ]]; then
  echo "ERROR: el usuario node del renderer ya no coincide con el tmpfs gobernado: $uid_node:$gid_node" >&2
  exit 1
fi
echo "OK: usuario efectivo del renderer fijado: 999:999"
