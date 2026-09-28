#!/usr/bin/env bash
# Sólo construye un artefacto local. Nunca conecta con la base ni reinicia servicios.
set -Eeuo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
repo=$(git -C "$script_dir" rev-parse --show-toplevel)
base=1433c6a44d757358742dc0932f0ed78b0a95dbb9

fallar() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }
command -v git >/dev/null || fallar 'falta git'
command -v rsync >/dev/null || fallar 'falta rsync'
command -v go >/dev/null || fallar 'falta Go'
command -v sha256sum >/dev/null || fallar 'falta sha256sum'
git -C "$repo" merge-base --is-ancestor "$base" HEAD || fallar 'checkout ajeno al candidato'
git -C "$repo" diff --quiet "$base" HEAD -- deploy/postgresql || fallar 'SQL cambió: revisar y actualizar plan'
[[ -z $(git -C "$repo" status --porcelain) ]] || fallar 'checkout sucio'

# Nunca empaquetar un plan que active el publicador B10 separado sin nuevo GO.
if grep -Eq '^deploy/postgresql/(bolsa_llamamientos/migraciones/000049_publicacion_cese_b10|bolsa_publica/migraciones/000003_publicacion_cese_replay)\.up\.sql$' \
    "$script_dir/migraciones.txt"; then
  fallar 'plan contiene Bolsa B49 o Bolsa pública 000003: NO-GO de publicación B10'
fi

for manifest in produccion.manifest interno.manifest; do
  [[ -f $repo/web/$manifest ]] || fallar "falta $manifest"
  # Importado por aplicacion.js y seguimiento-tramites.js en este candidato.
  grep -Fxq 'static/area-personal/mi-bolsa-historial.js' "$repo/web/$manifest" \
    || fallar "mi-bolsa-historial.js falta en $manifest"
  while IFS= read -r ruta; do
    [[ -n $ruta && $ruta != \#* ]] || continue
    [[ $ruta != /* && $ruta != *../* && -f $repo/web/$ruta ]] \
      || fallar "entrada ausente o insegura en $manifest: $ruta"
  done <"$repo/web/$manifest"
done

destino=$(mktemp -d /tmp/vec-piden-20260928.XXXXXXXX)
trap 'rm -rf -- "$destino"' ERR
mkdir -p -- "$destino/web" "$destino/evidencia"
go -C "$repo" build -buildvcs=false -o "$destino/vec-server" ./cmd/vec-server
rsync -a --delete -- "$repo/web/" "$destino/web/"
cp -- "$script_dir/migraciones.txt" "$destino/evidencia/migraciones.txt"
printf '%s\n' "$(git -C "$repo" rev-parse HEAD)" >"$destino/evidencia/commit.txt"
(
  cd "$destino"
  find vec-server web evidencia -type f -print0 | sort -z | xargs -0 sha256sum >SHA256SUMS
  sha256sum -c SHA256SUMS >/dev/null
)
trap - ERR
printf 'PAQUETE_LOCAL=%s\n' "$destino"
