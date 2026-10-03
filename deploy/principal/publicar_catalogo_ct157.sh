#!/usr/bin/env bash
set -euo pipefail

# Se ejecuta durante el despliegue, tras CT157 y antes de reanudar escritores.
# El segundo fichero lo produce el dominio Go con
# reglas.CanonicoCatalogoBaseReglas sobre el catálogo real del primer fichero.
# La conexión usa las variables libpq del operador; aquí no se manejan claves.
if [[ $# -ne 4 ]]; then
  echo 'Uso: publicar_catalogo_ct157.sh <catalogo_fuente.json> <catalogo_canonico.json> <manifiesto.json> <secuencia_esperada>' >&2
  exit 2
fi
fuente=$1
canonico=$2
manifiesto=$3
esperada=$4
if [[ ! -f $fuente || ! -f $canonico || ! -f $manifiesto || ! $esperada =~ ^(0|[1-9][0-9]{0,6})$ ]]; then
  echo 'CT157: fichero o secuencia no válidos' >&2
  exit 2
fi
id=$(jq -er '.id' "$canonico")
version=$(jq -er '.version' "$canonico")
estado=$(jq -er '.estado' "$canonico")
aprobacion=$(jq -er '.aprobacion_ref' "$canonico")
if [[ $id != vec.contratacion_temporal.reglas || $estado != publicado || ! $version =~ ^[1-9][0-9]{0,6}$ ]]; then
  echo 'CT157: catálogo canónico no publicable' >&2
  exit 2
fi
if [[ $(jq -er '.catalogo.id' "$fuente") != "$id" || $(jq -er '.catalogo.version' "$fuente") != "$version" ]]; then
  echo 'CT157: el artefacto no corresponde al catálogo fuente' >&2
  exit 2
fi
huella=$(sha256sum "$canonico" | cut -d' ' -f1)
fuente_huella=$(sha256sum "$fuente" | cut -d' ' -f1)
if [[ $(jq -er '.catalogo_id' "$manifiesto") != "$id" ||
      $(jq -er '.version' "$manifiesto") != "$version" ||
      $(jq -er '.fuente_sha256' "$manifiesto") != "$fuente_huella" ||
      $(jq -er '.canonico_sha256' "$manifiesto") != "$huella" ||
      $(jq -er '.fuente_ref' "$manifiesto") != "$(jq -er '.catalogo.fuente_ref' "$fuente")" ||
      $(jq -er '.fuente_revision' "$manifiesto") != "$(jq -er '.fuente.revision' "$fuente")" ||
      $(jq -er '.catalogo.aprobacion_ref' "$fuente") != "$aprobacion" ||
      $(jq -er '.aprobacion_ref' "$manifiesto") != "$aprobacion" ]]; then
  echo 'CT157: artefacto y fuente divergen del manifiesto' >&2
  exit 2
fi
if [[ ${VEC_CT157_ENTORNO:-} != produccion && ${VEC_CT157_ENTORNO:-} != desarrollo ]]; then
  echo 'CT157: entorno de publicación no indicado' >&2
  exit 2
fi
if [[ $(jq -r '.fuente.demostracion // false' "$fuente") == true ||
      $(jq -er '.catalogo.fuente_ref' "$fuente") == paquete:ejemplo:vec:v1 ||
      $(jq -er '.fuente_ref' "$canonico") == paquete:ejemplo:vec:v1 ||
      $(jq -r '.ejemplo' "$manifiesto") == true ]]; then
  if [[ ${VEC_CT157_ENTORNO:-} != desarrollo || ${VEC_CT157_PERMITIR_EJEMPLO:-} != 1 ]]; then
    echo 'CT157: catálogo de ejemplo no publicable en este entorno' >&2
    exit 2
  fi
else
  if [[ -z ${VEC_CT157_APROBACION_REF:-} ||
        $aprobacion != "$VEC_CT157_APROBACION_REF" ]]; then
    echo 'CT157: referencia de aprobación divergente' >&2
    exit 2
  fi
fi
if [[ $(wc -c < "$canonico") -gt 90000 ]]; then
  echo 'CT157: artefacto canónico demasiado grande para este publicador' >&2
  exit 2
fi
canonico_base64=$(base64 -w0 < "$canonico")
if [[ $(printf '%s' "$canonico_base64" | base64 -d | sha256sum | cut -d' ' -f1) != "$huella" ]]; then
  echo 'CT157: el artefacto cambió durante la publicación' >&2
  exit 2
fi
psql_bin=${VEC_CT157_PSQL:-psql}
"$psql_bin" -X -v ON_ERROR_STOP=1 -v base64="$canonico_base64" \
  -v version="$version" -v huella="$huella" -v fuente_huella="$fuente_huella" \
  -v aprobacion="$aprobacion" -v esperada="$esperada" <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(
 pg_catalog.hashtextextended('vec_contratacion_temporal:regla_base_activacion',0));
INSERT INTO vec_contratacion_temporal.regla_base_publicada_v1
 (catalogo_id,version,huella_sha256,canonico,fuente_sha256,aprobacion_ref)
VALUES ('vec.contratacion_temporal.reglas', :version, :'huella',
 pg_catalog.convert_from(pg_catalog.decode(:'base64','base64'),'UTF8'),
 :'fuente_huella',:'aprobacion')
ON CONFLICT (catalogo_id,version,huella_sha256) DO NOTHING;
WITH cabeza AS (
 SELECT coalesce(max(secuencia),0) AS secuencia
 FROM vec_contratacion_temporal.regla_base_activacion_v1
), activada AS (
 INSERT INTO vec_contratacion_temporal.regla_base_activacion_v1
  (secuencia,secuencia_esperada,activa,catalogo_id,version,huella_sha256,aprobacion_ref)
 SELECT :esperada+1,:esperada,true,'vec.contratacion_temporal.reglas',:version,:'huella',:'aprobacion'
 FROM cabeza WHERE secuencia=:esperada
 RETURNING secuencia
)
SELECT count(*) AS activaciones FROM activada \gset
\if :activaciones
COMMIT;
\else
ROLLBACK;
-- ON_ERROR_STOP hace que psql termine con código distinto de cero.
DO $fallo$ BEGIN
 RAISE EXCEPTION 'CT157: secuencia de activación cambiada' USING ERRCODE='40001';
END $fallo$;
\endif
SQL
