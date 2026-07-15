#!/usr/bin/env bash
set -Eeuo pipefail

: "${VEC_CEPH_ENDPOINT:?falta VEC_CEPH_ENDPOINT}"
: "${VEC_CEPH_REGION:?falta VEC_CEPH_REGION}"
: "${VEC_CEPH_BUCKET_CUARENTENA:?falta VEC_CEPH_BUCKET_CUARENTENA}"
: "${VEC_CEPH_BUCKET_ADMITIDA:?falta VEC_CEPH_BUCKET_ADMITIDA}"
: "${AWS_ACCESS_KEY_ID:?falta AWS_ACCESS_KEY_ID}"
: "${AWS_SECRET_ACCESS_KEY:?falta AWS_SECRET_ACCESS_KEY}"

aws_ceph=(aws --no-cli-pager --endpoint-url "${VEC_CEPH_ENDPOINT}" --region "${VEC_CEPH_REGION}")

crear_bucket() {
  local bucket="$1"
  if "${aws_ceph[@]}" s3api head-bucket --bucket "${bucket}" >/dev/null 2>&1; then
    return
  fi
  if [[ "${VEC_CEPH_REGION}" == "us-east-1" ]]; then
    "${aws_ceph[@]}" s3api create-bucket --bucket "${bucket}" --object-lock-enabled-for-bucket >/dev/null
  else
    "${aws_ceph[@]}" s3api create-bucket --bucket "${bucket}" --object-lock-enabled-for-bucket \
      --create-bucket-configuration "LocationConstraint=${VEC_CEPH_REGION}" >/dev/null
  fi
}

configurar_bucket() {
  local bucket="$1"
  "${aws_ceph[@]}" s3api put-bucket-versioning --bucket "${bucket}" \
    --versioning-configuration Status=Enabled
  # Sin regla de retencion predeterminada. La version admitida recibe
  # COMPLIANCE en su mismo PutObject; cuarentena debe permitir compensacion.
  "${aws_ceph[@]}" s3api put-object-lock-configuration --bucket "${bucket}" \
    --object-lock-configuration ObjectLockEnabled=Enabled
  "${aws_ceph[@]}" s3api get-bucket-versioning --bucket "${bucket}"
  "${aws_ceph[@]}" s3api get-object-lock-configuration --bucket "${bucket}"
}

for bucket in "${VEC_CEPH_BUCKET_CUARENTENA}" "${VEC_CEPH_BUCKET_ADMITIDA}"; do
  crear_bucket "${bucket}"
  configurar_bucket "${bucket}"
done
