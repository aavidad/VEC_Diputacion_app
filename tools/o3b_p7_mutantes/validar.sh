#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
v3a="$raiz/tools/o3b_p7_mutantes_v3a"
v3b="$raiz/tools/o3b_p7_mutantes_v3b"

esperar_sha() {
  local esperado=$1 ruta=$2 actual
  actual=$(sha256sum "$ruta" | cut -d ' ' -f 1)
  [[ $actual == "$esperado" ]] || {
    echo "NO-GO huella $ruta: $actual" >&2
    exit 1
  }
}

esperar_sha 8e3095f45793ac3d64e2f8cfeebae291b26a6aa01515971fe1538e2edcc44714 "$v3a/main.go"
esperar_sha 394521aab8277c7b497b82866dfdde51356c0e0fff8f43ec9717962f5d725cc4 "$v3a/evidencia/resultados.tsv"
esperar_sha e415b818e1fe9652ef4c33e271127e3f7d7742f1c045d03c9dfe78811e2cda2d "$v3b/main.go"
esperar_sha 20a9e2d5f208337774b3d67d6e3def4c5247ed8c87215c299e18947827091b13 "$v3b/evidencia-v3/resultados.tsv"

(cd "$raiz" && sha256sum -c "$v3a/evidencia/SHA256SUMS" >/dev/null)
(cd "$v3b/evidencia-v3" && sha256sum -c SHA256SUMS >/dev/null)

awk -F '\t' '
  FNR == NR {
    if (NF != 8 || $1 !~ /^M[0-9][0-9][0-9]$/ || visto[$1]++) exit 1
    familias[$2] = 1
    if ($4 != "COMPILA" || $5 !~ /^MUERTO-/ || $0 ~ /SKIP|SUPERVIV/) exit 1
    total++
    next
  }
  FNR == 1 { next }
  {
    if (NF != 12 || $1 !~ /^V3B[0-9][0-9][0-9]$/ || visto[$1]++) exit 1
    familias[$2] = 1
    if ($7 != "0" || $8 !~ /^(CONDUCTUAL|AST-CFG-SEMANTICO|META-B30)$/ || $8 ~ /SKIP|SUPERVIV/) exit 1
    total++
  }
  END {
    if (total != 131 || length(familias) != 32) exit 1
  }
' "$v3a/evidencia/resultados.tsv" "$v3b/evidencia-v3/resultados.tsv" || {
  echo "NO-GO catálogo/resultados conjuntos" >&2
  exit 1
}

echo "o3b_p7_mutantes=GO alternativas=131 familias=32 supervivientes=0"
