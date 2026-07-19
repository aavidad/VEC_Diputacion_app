#!/usr/bin/env bash
set -Eeuo pipefail

imagenes=(
  "ghcr.io/systemed/tilemaker:master@sha256:d32505d7827907089c2dd07517524276946d8930b4b82e93cf5c25ec989bbe41"
  "maptiler/tileserver-gl:v5.6.0@sha256:3a9ccdb24820b6814c8119bcc8a4376c39867cb0ffe69d62919ef898b90c2427"
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
