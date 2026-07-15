#!/usr/bin/env bash
set -Eeuo pipefail

: "${VEC_CEPH_INTEGRACION:?falta VEC_CEPH_INTEGRACION}"
: "${VEC_CEPH_ENDPOINT:?falta VEC_CEPH_ENDPOINT}"
: "${VEC_CEPH_BUCKET_CUARENTENA:?falta VEC_CEPH_BUCKET_CUARENTENA}"
: "${VEC_CEPH_BUCKET_ADMITIDA:?falta VEC_CEPH_BUCKET_ADMITIDA}"
: "${VEC_CEPH_CLAVE_DERIVACION_BASE64URL:?falta VEC_CEPH_CLAVE_DERIVACION_BASE64URL}"

if [[ "${VEC_CEPH_INTEGRACION}" != "1" ]]; then
  echo "VEC_CEPH_INTEGRACION debe valer 1" >&2
  exit 2
fi

raiz="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${raiz}"
go test -tags=integracion_ceph ./internal/vec/adapters/almacen/s3 \
  -run '^TestIntegracionCephPerfilFuerte$' -count=1 -timeout=3m
