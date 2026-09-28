#!/usr/bin/env bash
# Sólo construye un artefacto local. Nunca conecta con la base ni reinicia servicios.
set -Eeuo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
repo=$(git -C "$script_dir" rev-parse --show-toplevel)
base=600783c8ee34281ed9e5e98fa6b4c6ec4fab0a2e

fallar() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }
command -v git >/dev/null || fallar 'falta git'
command -v rsync >/dev/null || fallar 'falta rsync'
command -v go >/dev/null || fallar 'falta Go'
command -v sha256sum >/dev/null || fallar 'falta sha256sum'
git -C "$repo" merge-base --is-ancestor "$base" HEAD || fallar 'checkout ajeno al candidato'
git -C "$repo" diff --quiet "$base" HEAD -- . \
  ':(exclude)deploy/principal/piden_rrhh_20260928/**' \
  || fallar 'fuente distinta del stage fijado: revisar y actualizar plan'
[[ -z $(git -C "$repo" status --porcelain) ]] || fallar 'checkout sucio'

# Nunca empaquetar un plan que active el publicador B10 separado sin nuevo GO.
if grep -Eq '^deploy/postgresql/(bolsa_llamamientos/migraciones/000049_publicacion_cese_b10|bolsa_publica/migraciones/000003_publicacion_cese_replay)\.up\.sql$' \
    "$script_dir/migraciones.txt"; then
  fallar 'plan contiene Bolsa B49 o Bolsa pública 000003: NO-GO de publicación B10'
fi
rol_linea=$(grep -nFx 'deploy/postgresql/contratacion_temporal/roles_registrador_auditoria_up.sql' \
  "$script_dir/migraciones.txt" | cut -d: -f1)
ct136_linea=$(grep -nFx 'deploy/postgresql/contratacion_temporal/migraciones/000136_auditoria_frontera_auditoria_ruta_exacta.up.sql' \
  "$script_dir/migraciones.txt" | cut -d: -f1)
[[ $rol_linea =~ ^[0-9]+$ && $ct136_linea =~ ^[0-9]+$ \
   && $ct136_linea -eq $((rol_linea + 1)) ]] \
  || fallar 'plan CT136 sin delta DBA inmediatamente anterior'

for manifest in produccion.manifest interno.manifest; do
  [[ -f $repo/web/$manifest ]] || fallar "falta $manifest"
  while IFS= read -r ruta; do
    [[ -n $ruta && $ruta != \#* ]] || continue
    [[ $ruta != /* && $ruta != *../* && -f $repo/web/$ruta ]] \
      || fallar "entrada ausente o insegura en $manifest: $ruta (para OSM: scripts/aprovisionar_cartografia_osm.sh)"
  done <"$repo/web/$manifest"
done
# El área personal pertenece al inventario productivo, no al interno.
grep -Fxq 'static/area-personal/mi-bolsa-historial.js' "$repo/web/produccion.manifest" \
  || fallar 'mi-bolsa-historial.js falta en produccion.manifest'

destino=$(mktemp -d /tmp/vec-piden-20260928.XXXXXXXX)
trap 'rm -rf -- "$destino"' ERR
mkdir -p -- "$destino/web" "$destino/evidencia"
mkdir -p -- "$destino/.go-cache"
GOCACHE="$destino/.go-cache" GOTOOLCHAIN=auto GOMAXPROCS=2 \
  go -C "$repo" build -buildvcs=false -o "$destino/vec-server" ./cmd/vec-server
rm -rf -- "$destino/.go-cache"
rsync -a --delete --files-from="$repo/web/produccion.manifest" \
  -- "$repo/web/" "$destino/web/"
"$repo/scripts/verificar_web_produccion.sh" "$destino/web" \
  "$repo/web/produccion.manifest" >/dev/null
cp -- "$repo/web/interno.manifest" "$repo/web/publico.manifest" \
  "$repo/web/interno.locales.manifest" "$destino/evidencia/"
cp -- "$script_dir/migraciones.txt" "$destino/evidencia/migraciones.txt"
printf '%s\n' "$(git -C "$repo" rev-parse HEAD)" >"$destino/evidencia/commit.txt"
printf '%s\n' "$base" >"$destino/evidencia/fuente_commit.txt"
(
  cd "$destino"
  find vec-server web evidencia -type f -print0 | sort -z | xargs -0 sha256sum >SHA256SUMS
  sha256sum -c SHA256SUMS >/dev/null
)
trap - ERR
printf 'PAQUETE_LOCAL=%s\n' "$destino"
