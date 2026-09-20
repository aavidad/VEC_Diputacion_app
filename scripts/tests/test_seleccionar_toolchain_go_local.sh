#!/usr/bin/env bash
set -Eeuo pipefail
IFS=$'\n\t'

DIRECTORIO_TEST=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
RAIZ_REPOSITORIO=$(cd -- "$DIRECTORIO_TEST/../.." && pwd -P)
TEMPORAL=$(mktemp -d)

limpiar() {
  rm -rf -- "$TEMPORAL"
}
trap limpiar EXIT HUP INT TERM

FAKES="$TEMPORAL/fakes"
MODCACHE="$TEMPORAL/modcache"
REPO_PRUEBA="$TEMPORAL/repo"
mkdir -p "$FAKES" "$MODCACHE" "$REPO_PRUEBA/scripts"
cp "$RAIZ_REPOSITORIO/go.mod" "$REPO_PRUEBA/go.mod"
cp "$RAIZ_REPOSITORIO/scripts/seleccionar_toolchain_go_local.sh" \
  "$REPO_PRUEBA/scripts/seleccionar_toolchain_go_local.sh"
chmod 700 "$REPO_PRUEBA/scripts/seleccionar_toolchain_go_local.sh"
SELECTOR="$REPO_PRUEBA/scripts/seleccionar_toolchain_go_local.sh"
cat >"$FAKES/go" <<'FAKE_GO'
#!/usr/bin/env bash
set -Eeuo pipefail
if [[ "$1" == env ]]; then
  printf '%s\n' "$VEC_SELECTOR_TEST_MODCACHE" linux amd64
  exit 0
fi
printf 'invocación base inesperada: %s\n' "$*" >&2
exit 97
FAKE_GO
chmod 700 "$FAKES/go"

crear_toolchain() {
  local version=$1
  local salida=$2
  local ruta="$MODCACHE/golang.org/toolchain@v0.0.1-$version.linux-amd64/bin"
  mkdir -p "$ruta"
  cat >"$ruta/go" <<FAKE_TOOLCHAIN
#!/usr/bin/env bash
set -Eeuo pipefail
if [[ "\$1" == version ]]; then
  printf '%s\\n' '$salida'
  exit 0
fi
exit 98
FAKE_TOOLCHAIN
  chmod 700 "$ruta/go"
  printf '%s/go\n' "$ruta"
}

ejecutar_selector() {
  PATH="$FAKES:$PATH" VEC_SELECTOR_TEST_MODCACHE="$MODCACHE" "$SELECTOR"
}

RUTA_OK=$(crear_toolchain go1.26.6 'go version go1.26.6 linux/amd64')
[[ "$(ejecutar_selector)" == "$RUTA_OK" ]]

mv "$RUTA_OK" "$RUTA_OK.ausente"
if ejecutar_selector >"$TEMPORAL/falta" 2>&1; then
  printf 'el selector aceptó una toolchain ausente\n' >&2
  exit 1
fi
grep -Fq 'toolchain local requerida no disponible o insegura' "$TEMPORAL/falta"
mv "$RUTA_OK.ausente" "$RUTA_OK"

RUTA_INCORRECTA=$(crear_toolchain go1.26.6 'go version go1.25.1 linux/amd64')
if ejecutar_selector >"$TEMPORAL/version-incorrecta" 2>&1; then
  printf 'el selector aceptó una versión de toolchain incorrecta\n' >&2
  exit 1
fi
grep -Fq 'toolchain local no coincide' "$TEMPORAL/version-incorrecta"
rm -f -- "$RUTA_INCORRECTA"

printf '\ntoolchain go1.25.1\n' >>"$REPO_PRUEBA/go.mod"
if ejecutar_selector >"$TEMPORAL/ambigua" 2>&1; then
  printf 'el selector aceptó una directiva toolchain ambigua\n' >&2
  exit 1
fi
grep -Fq 'exactamente una directiva toolchain' "$TEMPORAL/ambigua"

printf 'selector de toolchain Go local: OK\n'
