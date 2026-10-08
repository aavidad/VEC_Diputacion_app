#!/usr/bin/env bash
# Sólo construye un artefacto local. Nunca conecta con la base ni reinicia servicios.
set -Eeuo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
repo=$(git -C "$script_dir" rev-parse --show-toplevel)
base=da48a409b7e75249fa1b2378d612a90998aa25bd

fallar() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }
command -v git >/dev/null || fallar 'falta git'
command -v rsync >/dev/null || fallar 'falta rsync'
command -v go >/dev/null || fallar 'falta Go'
command -v sha256sum >/dev/null || fallar 'falta sha256sum'
command -v file >/dev/null || fallar 'falta file'
command -v ldd >/dev/null || fallar 'falta ldd'
[[ $(GOTOOLCHAIN=go1.26.9 go version) == 'go version go1.26.9 linux/amd64' ]] \
  || fallar 'se requiere Go 1.26.9 linux/amd64, igual que el Dockerfile'
git -C "$repo" merge-base --is-ancestor "$base" HEAD || fallar 'checkout ajeno al candidato'
git -C "$repo" diff --quiet "$base" HEAD -- . \
  ':(exclude)deploy/principal/piden_rrhh_corte2_20260928/**' \
  || fallar 'fuente distinta del corte 2 fijado: revisar y actualizar plan'
[[ -z $(git -C "$repo" status --porcelain) ]] || fallar 'checkout sucio'
"$script_dir/validar_plan.sh"

# Nunca empaquetar un plan que active el publicador B10 separado sin nuevo GO.
if grep -Eq '^deploy/postgresql/(bolsa_llamamientos/migraciones/000049_publicacion_cese_b10|bolsa_publica/migraciones/000003_publicacion_cese_replay)\.up\.sql$' \
    "$script_dir/migraciones.txt"; then
  fallar 'plan contiene Bolsa B49 o Bolsa pública 000003: NO-GO de publicación B10'
fi
rol_linea=$(grep -nFx 'deploy/postgresql/contratacion_temporal/roles_registrador_auditoria_up.sql' \
  "$script_dir/migraciones.txt" | cut -d: -f1)
ct136_linea=$(grep -nFx 'deploy/postgresql/contratacion_temporal/migraciones/000136_auditoria_frontera_auditoria_ruta_exacta.up.sql' \
  "$script_dir/migraciones.txt" | cut -d: -f1)
calculador_linea=$(grep -nFx 'deploy/postgresql/bolsa_llamamientos/roles_calculador_politica_up.sql' \
  "$script_dir/migraciones.txt" | cut -d: -f1)
b51_linea=$(grep -nFx 'deploy/postgresql/bolsa_llamamientos/migraciones/000051_consulta_politica_ofertas_v3.up.sql' \
  "$script_dir/migraciones.txt" | cut -d: -f1)
[[ $rol_linea =~ ^[0-9]+$ && $ct136_linea =~ ^[0-9]+$ \
   && $ct136_linea -eq $((rol_linea + 1)) ]] \
  || fallar 'plan CT136 sin delta DBA inmediatamente anterior'
[[ $calculador_linea =~ ^[0-9]+$ && $b51_linea =~ ^[0-9]+$ \
   && $b51_linea -eq $((calculador_linea + 1)) ]] \
  || fallar 'plan B51 sin delta DBA inmediatamente anterior'

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

destino=$(mktemp -d /dev/shm/vec-piden-corte2-20260928.XXXXXXXX)
trap 'rm -rf -- "$destino"' ERR
mkdir -p -- "$destino/web" "$destino/evidencia"
mkdir -p -- "$destino/.go-cache"
GOCACHE="$destino/.go-cache" GOTOOLCHAIN=go1.26.9 GOMAXPROCS=2 \
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go -C "$repo" build -buildvcs=false -trimpath -ldflags='-s -w' \
  -o "$destino/vec-server" ./cmd/vec-server
rm -rf -- "$destino/.go-cache"
file "$destino/vec-server" | grep -Eq 'ELF 64-bit.*x86-64.*statically linked' \
  || fallar 'vec-server no es ELF amd64 enlazado estáticamente'
ldd_salida=$(ldd "$destino/vec-server" 2>&1 || true)
[[ $ldd_salida == *'not a dynamic executable'* ]] \
  || fallar 'vec-server conserva dependencias dinámicas'
rsync -a --delete --files-from="$repo/web/produccion.manifest" \
  -- "$repo/web/" "$destino/web/"
rsync -a --files-from=<(grep -vE '^[[:space:]]*(#|$)' "$script_dir/migraciones.txt") \
  -- "$repo/" "$destino/"
"$repo/scripts/verificar_web_produccion.sh" "$destino/web" \
  "$repo/web/produccion.manifest" >/dev/null
cp -- "$repo/web/interno.manifest" "$repo/web/publico.manifest" \
  "$repo/web/interno.locales.manifest" "$destino/evidencia/"
cp -- "$script_dir/migraciones.txt" "$destino/evidencia/migraciones.txt"
printf '%s\n' "$(git -C "$repo" rev-parse HEAD)" >"$destino/evidencia/commit.txt"
printf '%s\n' "$base" >"$destino/evidencia/fuente_commit.txt"
(
  cd "$destino"
  find vec-server web deploy evidencia -type f -print0 | sort -z | xargs -0 sha256sum >SHA256SUMS
  sha256sum -c SHA256SUMS >/dev/null
)
trap - ERR
printf 'PAQUETE_LOCAL=%s\n' "$destino"
