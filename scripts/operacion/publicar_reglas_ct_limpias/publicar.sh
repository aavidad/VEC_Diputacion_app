#!/usr/bin/env bash
set -euo pipefail
umask 077

carpeta=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
raiz=$(cd -- "$carpeta/../../.." && pwd)
case ${LC_ALL:-${LANG:-es}} in en*) mensajes=$carpeta/mensajes.en.json ;; *) mensajes=$carpeta/mensajes.es.json ;; esac
mensaje() { jq -r --arg clave "$1" '.[$clave] // $clave' "$mensajes"; }
error() { mensaje "$1" >&2; }
ct=$raiz/data/catalogos/contratacion-temporal/reglas.v2.json
bolsa=$raiz/data/catalogos/bolsa/reglas.v6.json
firma=$raiz/data/catalogos/contratacion-temporal/circuito_firma.v2.json
aprobacion=operacion:datos-inventados:20261012
ct_canonico=6ecf24d17b889c8bbd4400580977ac8b1c2c863c824f09ac5498d70a3d4a7141
ct_anterior=5530995222b852169848d7729a024cdc30b631fbde48608b1329e517f1d50572

case "${1:-}" in
  --comprobar) [[ $# == 1 ]] || { error comprobar_argumentos; exit 2; } ;;
  --aplicar) [[ $# == 2 && ${2:-} == PUBLICA-REGLAS-CT-LIMPIAS ]] || { error aplicar_confirmacion; exit 2; } ;;
  *) error uso; exit 2 ;;
esac

exigir_fuente() {
  local ruta=$1 huella=$2 id=$3 version=$4 entradas=$5
  [[ -f $ruta && ! -L $ruta && $(sha256sum -- "$ruta" | cut -d' ' -f1) == "$huella" ]] || {
    error fuente_modificada; exit 1;
  }
  jq -e --arg id "$id" --argjson version "$version" --argjson entradas "$entradas" --arg aprobacion "$aprobacion" \
    '.version_esquema == 1 and .fuente.demostracion == false and .catalogo.id == $id and .catalogo.version == $version and .catalogo.estado == "publicado" and .catalogo.aprobacion_ref == $aprobacion and (.catalogo.entradas | length) == $entradas' \
    "$ruta" >/dev/null || { error metadatos_incompatibles; exit 1; }
}

exigir_fuente "$ct" a9192d1a68e79a21f17650be6d57632ad3520fa91cfeb820fee84e5d6b89768a vec.contratacion_temporal.reglas 2 36
exigir_fuente "$bolsa" f6a9e52603d2c6ca8469ca753a53b17d142c5da9f16cf05a9ffd731cf0186f6e vec.bolsa.reglas 6 50
exigir_fuente "$firma" 4bbd801bb689c5f378c8a63307db1ce070566e4d4db65598af97fcf36e928abf vec.contratacion_temporal.circuito_firma 2 11

psql_bin=${VEC_REGLAS_CT_PSQL:-psql}
consultar() {
  # Una transacción READ ONLY también impide escrituras imprevistas desde
  # funciones, reglas de psql o extensiones del servidor.
  "$psql_bin" -X -qAt -v ON_ERROR_STOP=1 <<'SQL'
BEGIN TRANSACTION ISOLATION LEVEL REPEATABLE READ READ ONLY;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL statement_timeout='10s';
SELECT pg_catalog.json_build_object(
 'secuencia', (SELECT a.secuencia FROM vec_contratacion_temporal.regla_base_activacion_v1 a ORDER BY a.secuencia DESC LIMIT 1),
 'activa', (SELECT a.activa FROM vec_contratacion_temporal.regla_base_activacion_v1 a ORDER BY a.secuencia DESC LIMIT 1),
 'version', (SELECT a.version FROM vec_contratacion_temporal.regla_base_activacion_v1 a ORDER BY a.secuencia DESC LIMIT 1),
 'huella', (SELECT a.huella_sha256 FROM vec_contratacion_temporal.regla_base_activacion_v1 a ORDER BY a.secuencia DESC LIMIT 1),
 'aprobacion', (SELECT a.aprobacion_ref FROM vec_contratacion_temporal.regla_base_activacion_v1 a ORDER BY a.secuencia DESC LIMIT 1),
 'fuente', (SELECT b.fuente_sha256 FROM vec_contratacion_temporal.regla_base_publicada_v1 b JOIN vec_contratacion_temporal.regla_base_activacion_v1 a ON (b.catalogo_id,b.version,b.huella_sha256)=(a.catalogo_id,a.version,a.huella_sha256) ORDER BY a.secuencia DESC LIMIT 1),
 'ajustes', (SELECT pg_catalog.count(*) FROM vec_contratacion_temporal.regla_ajuste_version_v1),
 'ajustes_version', (SELECT v.version FROM vec_contratacion_temporal.regla_ajuste_version_v1 v ORDER BY v.version DESC LIMIT 1),
 'ajustes_huella', (SELECT v.huella_sha256 FROM vec_contratacion_temporal.regla_ajuste_version_v1 v ORDER BY v.version DESC LIMIT 1),
 'ajustes_clave', (SELECT v.clave_idempotencia FROM vec_contratacion_temporal.regla_ajuste_version_v1 v ORDER BY v.version DESC LIMIT 1),
 'ajustes_recibo', (SELECT v.recibo_ref FROM vec_contratacion_temporal.regla_ajuste_version_v1 v ORDER BY v.version DESC LIMIT 1),
 'ajustes_evento', (SELECT EXISTS (SELECT 1 FROM (SELECT * FROM vec_contratacion_temporal.regla_ajuste_version_v1 ORDER BY version DESC LIMIT 1) v JOIN vec_contratacion_temporal.regla_ajuste_outbox_v1 o ON (o.catalogo_id,o.version,o.huella_sha256)=(v.catalogo_id,v.version,v.huella_sha256))),
 'ajustes_rebase_integro', (SELECT EXISTS (
   SELECT 1 FROM (SELECT * FROM vec_contratacion_temporal.regla_ajuste_version_v1 ORDER BY version DESC LIMIT 1) v
   JOIN vec_contratacion_temporal.regla_ajuste_version_v1 p ON p.catalogo_id=v.catalogo_id AND p.version=v.version-1
   JOIN vec_contratacion_temporal.regla_ajuste_outbox_v1 o ON (o.catalogo_id,o.version,o.huella_sha256)=(v.catalogo_id,v.version,v.huella_sha256)
   WHERE 'c418dad9-6857-4c7c-a5f9-39ab47cfba16'::uuid=v.clave_idempotencia
     AND v.base_version=2 AND v.base_huella_sha256='6ecf24d17b889c8bbd4400580977ac8b1c2c863c824f09ac5498d70a3d4a7141'
     AND p.base_version=1 AND p.base_huella_sha256='5530995222b852169848d7729a024cdc30b631fbde48608b1329e517f1d50572'
     AND v.ajustes=p.ajustes AND v.ajustes_canonico=p.ajustes_canonico AND v.huella_sha256=p.huella_sha256
     AND v.decision_ref='operacion:datos-inventados:20261012' AND v.referencia=v.decision_ref
     AND v.consumo_huella_sha256=v.solicitud_huella_sha256 AND v.auditoria_ref LIKE 'provision:%'
     AND o.tipo='contratacion_temporal.reglas.ajustadas' AND o.creada_en=v.vigente_desde
     AND NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.regla_ajuste_cambio_v1 c
       WHERE c.catalogo_id=v.catalogo_id AND c.version=v.version))),
 'ajustes_base_version', (SELECT v.base_version FROM vec_contratacion_temporal.regla_ajuste_version_v1 v ORDER BY v.version DESC LIMIT 1),
 'ajustes_base_huella', (SELECT v.base_huella_sha256 FROM vec_contratacion_temporal.regla_ajuste_version_v1 v ORDER BY v.version DESC LIMIT 1));
COMMIT;
SQL
}

if ! estado=$(consultar 2>/dev/null); then
  error base_no_disponible
  exit 1
fi
estado=$(printf '%s\n' "$estado" | jq -ec 'select(type == "object")') || { error respuesta_invalida; exit 1; }
secuencia=$(jq -r '.secuencia // ""' <<<"$estado")
if [[ $(jq -r '.activa' <<<"$estado") != true || ! $secuencia =~ ^[1-9][0-9]{0,6}$ ]]; then
  error base_inactiva; exit 1
fi
version=$(jq -r '.version' <<<"$estado")
huella=$(jq -r '.huella' <<<"$estado")
ajustes=$(jq -r '.ajustes' <<<"$estado")
ajustes_version=$(jq -r '.ajustes_version // 0' <<<"$estado")
ajustes_huella=$(jq -r '.ajustes_huella // ""' <<<"$estado")
ajustes_base_version=$(jq -r '.ajustes_base_version // 0' <<<"$estado")
ajustes_base_huella=$(jq -r '.ajustes_base_huella // ""' <<<"$estado")
if [[ $secuencia == 2 && $version == 2 && $huella == "$ct_canonico" && $(jq -r '.fuente' <<<"$estado") == a9192d1a68e79a21f17650be6d57632ad3520fa91cfeb820fee84e5d6b89768a && $(jq -r '.aprobacion' <<<"$estado") == "$aprobacion" ]]; then
  resultado=replay
elif [[ $secuencia == 1 && $version == 1 && $huella == "$ct_anterior" && $(jq -r '.aprobacion' <<<"$estado") == sin_aprobacion_rrhh:ejemplo_desarrollo ]]; then
  resultado=preparado
else
  error base_incompatible; exit 1
fi
if [[ $ajustes != 0 ]]; then
  if [[ ! $ajustes =~ ^[1-9][0-9]{0,6}$ || ! $ajustes_version =~ ^[1-9][0-9]{0,6}$ || ! $ajustes_huella =~ ^[0-9a-f]{64}$ || $(jq -r '.ajustes_evento' <<<"$estado") != true ]]; then
    error ajustes_incompatibles; exit 1
  fi
  if [[ $ajustes_base_version == 1 && $ajustes_base_huella == "$ct_anterior" ]]; then
    if [[ $resultado == replay ]]; then resultado=rebase_pendiente; else resultado=preparado_con_rebase; fi
  elif [[ $resultado == replay && $ajustes_base_version == 2 && $ajustes_base_huella == "$ct_canonico" && $(jq -r '.ajustes_rebase_integro' <<<"$estado") == true ]]; then
    resultado=replay
  else
    error ajustes_incompatibles; exit 1
  fi
fi
if [[ $1 == --comprobar || $resultado == replay ]]; then
  jq -nc --arg resultado "$resultado" --argjson secuencia "$secuencia" --argjson ajustes "$ajustes" --arg huella "$ct_canonico" \
    '{resultado:$resultado,version:2,huella_sha256:$huella,secuencia_actual:$secuencia,ajustes:$ajustes,canal:"datos_operacion"}'
  exit 0
fi

if [[ $version == 1 ]]; then
  preparador=${VEC_REGLAS_CT_PREPARADOR:-}
  if [[ -z $preparador || ! -x $preparador ]]; then
    error preparador_ausente; exit 2
  fi
  privado=$(mktemp -d "${TMPDIR:-/var/tmp}/vec-reglas-ct.XXXXXXXX")
  trap 'rm -rf -- "$privado"' EXIT
  chmod 700 "$privado"
  cp -- "$ct" "$privado/fuente.json"
  chmod 600 "$privado/fuente.json"
  [[ $(sha256sum "$privado/fuente.json" | cut -d' ' -f1) == a9192d1a68e79a21f17650be6d57632ad3520fa91cfeb820fee84e5d6b89768a ]] || {
    error fuente_cambio; exit 1;
  }
  "$preparador" -fuente "$privado/fuente.json" -salida "$privado/canonico.json" -manifiesto "$privado/manifiesto.json" -aprobacion-ref "$aprobacion" >/dev/null
  [[ $(sha256sum "$privado/canonico.json" | cut -d' ' -f1) == "$ct_canonico" ]] || { error canonico_divergente; exit 1; }
  VEC_CT190_ENTORNO=desarrollo VEC_CT190_APROBACION_REF="$aprobacion" VEC_CT190_PSQL="$psql_bin" \
    "$raiz/deploy/principal/publicar_catalogo_ct190.sh" "$privado/fuente.json" "$privado/canonico.json" "$privado/manifiesto.json" "$secuencia" >/dev/null
  if ! posterior=$(consultar 2>/dev/null); then error publicacion_incierta; exit 1; fi
  posterior=$(printf '%s\n' "$posterior" | jq -ec 'select(type == "object")') || exit 1
  if [[ $(jq -r '.version' <<<"$posterior") != 2 || $(jq -r '.huella' <<<"$posterior") != "$ct_canonico" || $(jq -r '.fuente' <<<"$posterior") != a9192d1a68e79a21f17650be6d57632ad3520fa91cfeb820fee84e5d6b89768a || $(jq -r '.secuencia' <<<"$posterior") != "$((secuencia+1))" || $(jq -r '.ajustes_version // 0' <<<"$posterior") != "$ajustes_version" || $(jq -r '.ajustes_huella // ""' <<<"$posterior") != "$ajustes_huella" ]]; then
    error publicacion_no_confirmada; exit 1
  fi
  secuencia=$((secuencia+1))
fi
if [[ $ajustes != 0 ]]; then
  if ! "$psql_bin" -X -qAt -v ON_ERROR_STOP=1 < "$carpeta/rebase_ajustes.sql" >/dev/null 2>&1; then
    error rebase_incierto; exit 1
  fi
fi
if ! posterior=$(consultar 2>/dev/null); then error publicacion_incierta; exit 1; fi
posterior=$(printf '%s\n' "$posterior" | jq -ec 'select(type == "object")') || exit 1
if [[ $(jq -r '.version' <<<"$posterior") != 2 || $(jq -r '.huella' <<<"$posterior") != "$ct_canonico" || $(jq -r '.secuencia' <<<"$posterior") != "$secuencia" ]]; then
  error publicacion_no_confirmada; exit 1
fi
if [[ $ajustes != 0 && ( $(jq -r '.ajustes' <<<"$posterior") != "$((ajustes+1))" || $(jq -r '.ajustes_version' <<<"$posterior") != "$((ajustes_version+1))" || $(jq -r '.ajustes_huella' <<<"$posterior") != "$ajustes_huella" || $(jq -r '.ajustes_rebase_integro' <<<"$posterior") != true ) ]]; then
  error rebase_no_confirmado; exit 1
fi
jq -nc --arg huella "$ct_canonico" --argjson secuencia "$secuencia" \
  '{resultado:"registrado",version:2,huella_sha256:$huella,secuencia_actual:$secuencia,canal:"datos_operacion"}'
