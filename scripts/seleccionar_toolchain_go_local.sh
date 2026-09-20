#!/usr/bin/env bash
set -Eeuo pipefail
IFS=$'\n\t'

# Devuelve en stdout el Go exacto declarado por go.mod, si ya está presente en
# la caché local. No consulta la red ni permite que el Go base cambie de
# toolchain por su cuenta.
fallar() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

DIRECTORIO_SCRIPT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
RAIZ_REPOSITORIO=$(cd -- "$DIRECTORIO_SCRIPT/.." && pwd -P)
MODULO="$RAIZ_REPOSITORIO/go.mod"

[[ -f "$MODULO" ]] || fallar "no existe go.mod: $MODULO"

DIRECTIVAS=$(awk '
  /^[[:space:]]*\/\// { next }
  /^[[:space:]]*toolchain([[:space:]]|$)/ {
    if ($0 !~ /^[[:space:]]*toolchain[[:space:]]+go[0-9]+\.[0-9]+\.[0-9]+[[:space:]]*(\/\/.*)?$/) {
      print "INVALIDA"
      exit 0
    }
    linea = $0
    sub(/^[[:space:]]*toolchain[[:space:]]+/, "", linea)
    sub(/[[:space:]].*$/, "", linea)
    print linea
  }
' "$MODULO")

[[ "$DIRECTIVAS" != 'INVALIDA' ]] || fallar 'la directiva toolchain de go.mod no es exacta'
NUMERO_DIRECTIVAS=$(printf '%s\n' "$DIRECTIVAS" | sed '/^$/d' | wc -l)
[[ "$NUMERO_DIRECTIVAS" -eq 1 ]] || fallar 'go.mod debe contener exactamente una directiva toolchain goX.Y.Z'
VERSION=$(printf '%s\n' "$DIRECTIVAS" | sed '/^$/d')

GO_BASE=$(command -v go 2>/dev/null || true)
[[ -n "$GO_BASE" ]] || fallar 'no existe un Go base en PATH para consultar la caché local'
[[ -x "$GO_BASE" ]] || fallar "Go base no es ejecutable: $GO_BASE"

mapfile -t ENTORNO_GO < <(GOENV=off GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off \
  "$GO_BASE" env GOMODCACHE GOHOSTOS GOHOSTARCH)
[[ "${#ENTORNO_GO[@]}" -eq 3 ]] || fallar 'el Go base no devolvió GOMODCACHE, GOHOSTOS y GOHOSTARCH'
GOMODCACHE=${ENTORNO_GO[0]}
GOHOSTOS=${ENTORNO_GO[1]}
GOHOSTARCH=${ENTORNO_GO[2]}
[[ -n "$GOMODCACHE" && -n "$GOHOSTOS" && -n "$GOHOSTARCH" ]] || fallar 'el Go base devolvió un entorno de toolchain incompleto'

GO_LOCAL="$GOMODCACHE/golang.org/toolchain@v0.0.1-$VERSION.$GOHOSTOS-$GOHOSTARCH/bin/go"
[[ -f "$GO_LOCAL" && -x "$GO_LOCAL" && ! -L "$GO_LOCAL" ]] || \
  fallar "toolchain local requerida no disponible o insegura: $GO_LOCAL"

VERSION_OBSERVADA=$(GOENV=off GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off "$GO_LOCAL" version 2>&1) || \
  fallar "no se pudo ejecutar la toolchain local: $GO_LOCAL"
VERSION_ESPERADA="go version $VERSION $GOHOSTOS/$GOHOSTARCH"
[[ "$VERSION_OBSERVADA" == "$VERSION_ESPERADA" ]] || \
  fallar "toolchain local no coincide: se esperaba '$VERSION_ESPERADA' y se obtuvo '$VERSION_OBSERVADA'"

printf '%s\n' "$GO_LOCAL"
