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
probe='/api/vec/contratacion-temporal/expedientes/comunicaciones?expediente_ref=expediente:ct:ensayo'
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
expect_error 'alcance SQL desconocido' --scope desconocido --plan "$tmp/vacio" --runtime "$tmp/runtime" --probe "$probe"
expect_error 'la sonda HTTP local no supera' \
  --plan "$tmp/vacio" --runtime "$tmp/runtime" --probe http://example.test/
expect_error 'la sonda HTTP local no supera' \
  --plan "$tmp/vacio" --runtime "$tmp/runtime" --probe '/api/vec/contratacion-temporal/expedientes/comunicaciones?expediente_ref=a&expediente_ref=b'
expect_error 'la sonda HTTP local no supera' \
  --plan "$tmp/vacio" --runtime "$tmp/runtime" --probe '/api/vec/contratacion-temporal/expedientes/comunicaciones?expediente_ref=%0A'
expect_error 'la sonda HTTP local no supera' \
  --plan "$tmp/vacio" --runtime "$tmp/runtime" --probe /api/vec/contratacion-temporal/cuadro/consultas --probe-method POST
expect_error 'la sonda HTTP local no supera' \
  --plan "$tmp/vacio" --runtime "$tmp/runtime" --probe /api/vec/contratacion-temporal/cuadro/consultas \
  --probe-method POST --probe-json '{"filtros":{"texto":"https://externo.invalid"},"paginacion":{"limite":1,"cursor":""}}'
expect_error 'el directorio TLS debe ser absoluto' \
  --plan "$tmp/vacio" --runtime "$tmp/runtime" --probe "$probe" --tls-material relativo
expect_error 'falta material TLS local: ca/ca.crt' \
  --plan "$tmp/vacio" --runtime "$tmp/runtime" --probe "$probe" --tls-material "$tmp/tls"
expect_error 'faltan ' --plan "$tmp/vacio" --runtime "$tmp/runtime" --probe "$probe"

git -C "$repo" ls-files 'deploy/postgresql' | grep -E '/roles[^/]*_up[.]sql$|[.]up[.]sql$' \
  > "$tmp/completo"
dba_b1='deploy/postgresql/bolsa_llamamientos/dba/20260928_b1_rls_propietario/01_cerrar_politicas.sql'
git -C "$repo" ls-files --error-unmatch "$dba_b1" >/dev/null
printf '%s\n' "$dba_b1" >> "$tmp/completo"
primero=$(head -1 "$tmp/completo")
printf '%s\n' "$primero" >> "$tmp/completo"
expect_error 'repite archivos' --plan "$tmp/completo" --runtime "$tmp/runtime" --probe "$probe"
sed -i '$d' "$tmp/completo"
grep -Fxv "$dba_b1" "$tmp/completo" > "$tmp/sin-dba"
expect_error "falta: $dba_b1" --plan "$tmp/sin-dba" --runtime "$tmp/runtime" --probe "$probe"
otro_sql='deploy/postgresql/bolsa_llamamientos/roles_down.sql'
printf '%s\n' "$otro_sql" >> "$tmp/completo"
expect_error "sobra: $otro_sql" --plan "$tmp/completo" --runtime "$tmp/runtime" --probe "$probe"
sed -i '$d' "$tmp/completo"
expect_error 'Docker local inaccesible' --scope ct-bolsa --repo "$repo" --plan "$tmp/completo" --runtime "$tmp/runtime" --probe "$probe"
expect_error 'Docker local inaccesible' --scope ct-bolsa --repo "$repo" --plan "$tmp/completo" --runtime "$tmp/runtime" \
  --probe /api/vec/contratacion-temporal/cuadro/consultas --probe-method POST \
  --probe-json '{"filtros":{},"paginacion":{"limite":1,"cursor":""}}'
printf 'OK guardas del plan CT/Bolsa y Docker inaccesible\n'
