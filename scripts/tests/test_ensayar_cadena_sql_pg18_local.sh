#!/usr/bin/env bash
set -Eeuo pipefail
repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
tmp=$(mktemp -d /tmp/vec-puerta-sql-test.XXXXXXXX)
trap 'rm -rf -- "$tmp"' EXIT
cat > "$tmp/runtime" <<'SH'
#!/usr/bin/env bash
exit 0
SH
cat > "$tmp/docker" <<'SH'
#!/usr/bin/env bash
exit 1
SH
chmod +x "$tmp/runtime" "$tmp/docker"
script="$repo/scripts/ensayar_cadena_sql_pg18_local.sh"
export PATH="$tmp:$PATH"

expect_error() {
  local texto=$1
  shift
  if "$script" "$@" >"$tmp/out" 2>"$tmp/err"; then
    printf 'La puerta aceptó un caso que debía fallar: %s\n' "$texto" >&2
    exit 1
  fi
  if ! grep -Fq -- "$texto" "$tmp/err"; then
    printf 'Error esperado ausente: %s\n' "$texto" >&2
    sed -n '1,25p' "$tmp/err" >&2
    exit 1
  fi
}

"$script" --help | grep -Fq -- '--plan ORDEN'
expect_error 'falta valor para --plan' --plan
touch "$tmp/vacio"
expect_error 'la sonda debe ser una ruta HTTP local' \
  --plan "$tmp/vacio" --runtime "$tmp/runtime" --probe http://example.test/
expect_error 'faltan ' --plan "$tmp/vacio" --runtime "$tmp/runtime" --probe /api/vec/contratacion-temporal/cuadro/consultas

git -C "$repo" ls-files 'deploy/postgresql' | grep -E '/roles[^/]*_up[.]sql$|[.]up[.]sql$' \
  > "$tmp/completo"
primero=$(head -1 "$tmp/completo")
printf '%s\n' "$primero" >> "$tmp/completo"
expect_error 'repite archivos' --plan "$tmp/completo" --runtime "$tmp/runtime" --probe /api/vec/contratacion-temporal/cuadro/consultas
sed -i '$d' "$tmp/completo"
expect_error 'Docker local inaccesible' --plan "$tmp/completo" --runtime "$tmp/runtime" --probe /api/vec/contratacion-temporal/cuadro/consultas
printf 'OK guardas del plan CT/Bolsa y Docker inaccesible\n'
