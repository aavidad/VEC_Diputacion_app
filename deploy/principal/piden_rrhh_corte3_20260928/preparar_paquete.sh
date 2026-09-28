#!/usr/bin/env bash
# Exportación local fail-closed. Nunca conecta con PostgreSQL ni cambia servicios.
set -Eeuo pipefail
script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
repo=$(git -C "$script_dir" rev-parse --show-toplevel)
fallar() { printf 'ERROR: paquete RRHH c3: %s\n' "$*" >&2; exit 1; }
[[ $# == 0 ]] || fallar 'uso: preparar_paquete.sh (sin argumentos)'
for programa in git rsync go sha256sum file ldd; do
  command -v "$programa" >/dev/null || fallar "falta $programa"
done
[[ -f $script_dir/fuente_final.txt ]] || fallar 'falta hash final ratificado; NO EXPORTAR'
fuente=$(cat "$script_dir/fuente_final.txt")
[[ $fuente =~ ^[0-9a-f]{40}$ ]] || fallar 'hash final inválido'
paquete=$(git -C "$repo" rev-parse HEAD)
[[ ${VEC_PIDEN_C3_PUERTA_SHA:-} == "$fuente" \
   && ${VEC_PIDEN_C3_GO_SQL_SHA:-} == "$paquete" \
   && ${VEC_PIDEN_C3_GO_OPS_SHA:-} == "$paquete" ]] \
  || fallar 'faltan puerta verde del hash fuente y dos GO SQL/ops del hash paquete'
[[ -z $(git -C "$repo" status --porcelain) ]] || fallar 'checkout sucio'
"$script_dir/validar_plan.sh"
"$script_dir/validar_web.sh"
[[ $(GOTOOLCHAIN=go1.26.6 go version) == 'go version go1.26.6 linux/amd64' ]] \
  || fallar 'se requiere Go 1.26.6 linux/amd64'

destino=$(mktemp -d /dev/shm/vec-piden-corte3-20260928.XXXXXXXX)
trap 'rm -rf -- "$destino"' ERR
mkdir -p -- "$destino/web" "$destino/evidencia" "$destino/.go-cache"
GOCACHE="$destino/.go-cache" GOTOOLCHAIN=go1.26.6 GOMAXPROCS=2 \
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go -C "$repo" build -buildvcs=false -trimpath -ldflags='-s -w' \
  -o "$destino/vec-server" ./cmd/vec-server
rm -rf -- "$destino/.go-cache"
file "$destino/vec-server" | grep -Eq 'ELF 64-bit.*x86-64.*statically linked' \
  || fallar 'binario no es ELF amd64 estático'
ldd_salida=$(ldd "$destino/vec-server" 2>&1 || true)
[[ $ldd_salida == *'not a dynamic executable'* ]] \
  || fallar 'binario con dependencias dinámicas'
rsync -a --delete --files-from="$repo/web/produccion.manifest" -- "$repo/web/" "$destino/web/"
rsync -a --files-from=<(grep -vE '^[[:space:]]*(#|$)' "$script_dir/migraciones.txt") \
  -- "$repo/" "$destino/"
"$repo/scripts/verificar_web_produccion.sh" "$destino/web" "$repo/web/produccion.manifest" >/dev/null
cp -- "$repo/web/interno.manifest" "$repo/web/publico.manifest" \
  "$repo/web/interno.locales.manifest" "$destino/evidencia/"
cp -- "$script_dir/migraciones.txt" "$destino/evidencia/migraciones.txt"
printf '%s\n' "$fuente" >"$destino/evidencia/fuente_commit.txt"
printf '%s\n' "$paquete" >"$destino/evidencia/paquete_commit.txt"
(
  cd "$destino"
  find vec-server web deploy evidencia -type f -print0 | sort -z | xargs -0 sha256sum >SHA256SUMS
  sha256sum -c SHA256SUMS >/dev/null
)
trap - ERR
printf 'PAQUETE_LOCAL=%s\n' "$destino"
