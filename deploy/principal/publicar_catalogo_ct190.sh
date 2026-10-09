#!/usr/bin/env bash
set -euo pipefail

# Se ejecuta tras CT190/CT191 y antes de reanudar escritores.
# El segundo fichero lo produce el dominio Go con
# reglas.CanonicoCatalogoBaseReglas sobre el catálogo real del primer fichero.
# La conexión usa las variables libpq del operador; aquí no se manejan claves.
if [[ $# -ne 4 ]]; then
  echo 'Uso: publicar_catalogo_ct190.sh <catalogo_fuente.json> <catalogo_canonico.json> <manifiesto.json> <secuencia_esperada>' >&2
  exit 2
fi
fuente=$1
canonico=$2
manifiesto=$3
esperada=$4
if [[ ! -f $fuente || ! -f $canonico || ! -f $manifiesto || ! $esperada =~ ^(0|[1-9][0-9]{0,6})$ ]]; then
  echo 'CT190: fichero o secuencia no válidos' >&2
  exit 2
fi
if [[ $(wc -c < "$fuente") -gt 4194304 || $(wc -c < "$manifiesto") -gt 8192 ]]; then
  echo 'CT190: artefacto demasiado grande' >&2
  exit 2
fi
# Los cotejos y el envío usan las mismas copias privadas, aun si cambian las rutas originales.
directorio=$(mktemp -d "${TMPDIR:-/var/tmp}/vec-ct190.XXXXXXXX")
trap 'rm -rf -- "$directorio"' EXIT
cp -- "$fuente" "$directorio/fuente.json"
cp -- "$canonico" "$directorio/canonico.json"
cp -- "$manifiesto" "$directorio/manifiesto.json"
chmod 600 "$directorio"/*.json
fuente=$directorio/fuente.json
canonico=$directorio/canonico.json
manifiesto=$directorio/manifiesto.json
id=$(jq -er '.id' "$canonico")
version=$(jq -er '.version' "$canonico")
estado=$(jq -er '.estado' "$canonico")
aprobacion=$(jq -er '.aprobacion_ref' "$canonico")
if [[ $id != vec.contratacion_temporal.reglas || $estado != publicado ||
      ! $version =~ ^[1-9][0-9]{0,6}$ || -z $aprobacion ]]; then
  echo 'CT190: catálogo canónico no publicable' >&2
  exit 2
fi
if [[ $(jq -er '.catalogo.id' "$fuente") != "$id" ||
      $(jq -er '.catalogo.version' "$fuente") != "$version" ||
      $(jq -er '.catalogo.estado' "$fuente") != "$estado" ||
      $(jq -er '.catalogo.fuente_ref' "$fuente") != "$(jq -er '.fuente_ref' "$canonico")" ]]; then
  echo 'CT190: el artefacto no corresponde al catálogo fuente' >&2
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
  echo 'CT190: artefacto y fuente divergen del manifiesto' >&2
  exit 2
fi
if [[ ${VEC_CT190_ENTORNO:-} != produccion && ${VEC_CT190_ENTORNO:-} != desarrollo ]]; then
  echo 'CT190: entorno de publicación no indicado' >&2
  exit 2
fi
if [[ $(jq -r '.fuente.demostracion // false' "$fuente") == true ||
      $(jq -er '.catalogo.fuente_ref' "$fuente") == paquete:ejemplo:vec:v1 ||
      $(jq -er '.fuente_ref' "$canonico") == paquete:ejemplo:vec:v1 ||
      $(jq -er '.ejemplo | booleans' "$manifiesto") == true ]]; then
  if [[ ${VEC_CT190_ENTORNO:-} != desarrollo || ${VEC_CT190_PERMITIR_EJEMPLO:-} != 1 ||
        $(jq -er '.fuente.demostracion | booleans' "$fuente") != true ||
        $(jq -er '.ejemplo | booleans' "$manifiesto") != true ||
        $(jq -er '.catalogo.fuente_ref' "$fuente") != paquete:ejemplo:vec:v1 ]]; then
    echo 'CT190: catálogo de ejemplo no publicable en este entorno' >&2
    exit 2
  fi
else
  if [[ $(jq -er '.fuente.demostracion | booleans' "$fuente") != false ||
        $(jq -er '.ejemplo | booleans' "$manifiesto") != false ||
        -z ${VEC_CT190_APROBACION_REF:-} ||
        $aprobacion != "$VEC_CT190_APROBACION_REF" ]]; then
    echo 'CT190: referencia de aprobación divergente' >&2
    exit 2
  fi
fi
if [[ $(wc -c < "$canonico") -gt 90000 ]]; then
  echo 'CT190: artefacto canónico demasiado grande para este publicador' >&2
  exit 2
fi
canonico_base64=$(base64 -w0 < "$canonico")
if [[ $(printf '%s' "$canonico_base64" | base64 -d | sha256sum | cut -d' ' -f1) != "$huella" ]]; then
  echo 'CT190: el artefacto cambió durante la publicación' >&2
  exit 2
fi
psql_bin=${VEC_CT190_PSQL:-psql}
"$psql_bin" -X -v ON_ERROR_STOP=1 -v base64="$canonico_base64" \
  -v version="$version" -v huella="$huella" -v fuente_huella="$fuente_huella" \
  -v aprobacion="$aprobacion" -v esperada="$esperada" <<'SQL'
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(
 pg_catalog.hashtextextended('vec_contratacion_temporal:regla_base_activacion',0));
SELECT 1 / (SELECT CASE WHEN coalesce(max(secuencia),0)=:esperada
 THEN 1 ELSE 0 END FROM vec_contratacion_temporal.regla_base_activacion_v1)
 AS cabeza_comprobada;
INSERT INTO vec_contratacion_temporal.regla_base_publicada_v1
 (catalogo_id,version,huella_sha256,canonico,fuente_sha256,aprobacion_ref)
VALUES ('vec.contratacion_temporal.reglas', :version, :'huella',
 pg_catalog.convert_from(pg_catalog.decode(:'base64','base64'),'UTF8'),
 :'fuente_huella',:'aprobacion')
ON CONFLICT (catalogo_id,version,huella_sha256) DO NOTHING;
SELECT 1 / coalesce((SELECT CASE WHEN fuente_sha256=:'fuente_huella'
 AND aprobacion_ref=:'aprobacion' THEN 1 ELSE 0 END
 FROM vec_contratacion_temporal.regla_base_publicada_v1
 WHERE catalogo_id='vec.contratacion_temporal.reglas'
 AND version=:version AND huella_sha256=:'huella'),0) AS publicacion_comprobada;
INSERT INTO vec_contratacion_temporal.regla_base_activacion_v1
 (secuencia,secuencia_esperada,activa,catalogo_id,version,huella_sha256,aprobacion_ref)
VALUES (:esperada+1,:esperada,true,'vec.contratacion_temporal.reglas',:version,:'huella',:'aprobacion');
COMMIT;
SQL
