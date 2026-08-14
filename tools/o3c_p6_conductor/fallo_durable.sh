#!/usr/bin/env bash
set -euo pipefail

raiz_autoprueba=
temporal_publicacion=

limpiar() {
  [[ -z $temporal_publicacion ]] || rm -rf -- "$temporal_publicacion"
  [[ -z $raiz_autoprueba ]] || rm -rf -- "$raiz_autoprueba"
}
trap limpiar EXIT

publicar() {
  local origen=$1 destino=$2 stdout=$3 stderr=$4 id=$5 modo=$6 estado=$7 pgid=$8
  local antes=$9 despues=${10} head=${11} version_go=${12}
  local padre nombre temporal archivo

  [[ -d $origen && -f $stdout && -f $stderr && ! -e $destino ]] || return 2
  [[ $id =~ ^[A-Z0-9_]+$ && $modo =~ ^(normal|race)$ && $estado =~ ^[0-9]+$ ]] || return 2
  [[ $pgid =~ ^(si|no)$ && $antes =~ ^[0-9]+(,[0-9]+){4}$ && $despues =~ ^[0-9]+(,[0-9]+){4}$ ]] || return 2
  [[ $head =~ ^[0-9a-f]{40}$ && $version_go == 'go version go1.26.5 linux/amd64' ]] || return 2
  padre=$(dirname "$destino")
  nombre=$(basename "$destino")
  [[ -d $padre ]] || return 2

  umask 077
  temporal=$(mktemp -d "$padre/.${nombre}.fallo.XXXXXX")
  temporal_publicacion=$temporal
  cp -- "$origen"/* "$temporal/"
  cp -- "$stdout" "$temporal/fallo.stdout"
  cp -- "$stderr" "$temporal/fallo.stderr"
  printf 'id\tmodo\testado\tstdout_bytes\tstderr_bytes\tpgid_ausente\tinventario_antes\tinventario_despues\n' >"$temporal/fallo.tsv"
  printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$id" "$modo" "$estado" \
    "$(wc -c <"$stdout")" "$(wc -c <"$stderr")" "$pgid" "$antes" "$despues" >>"$temporal/fallo.tsv"
  {
    printf 'resultado=NO-GO\n'
    printf 'fallo_id=%s\nmodo=%s\nestado=%s\n' "$id" "$modo" "$estado"
    printf 'head=%s\ngo_version=%s\n' "$head" "$version_go"
    printf 'stdout_bytes=%s\nstderr_bytes=%s\n' "$(wc -c <"$stdout")" "$(wc -c <"$stderr")"
    printf 'pgid_ausente=%s\ninventario_antes=%s\ninventario_despues=%s\n' "$pgid" "$antes" "$despues"
  } >"$temporal/resumen.txt"
  : >"$temporal/SHA256SUMS"
  while IFS= read -r archivo; do
    sha256sum "$archivo"
  done < <(find "$temporal" -maxdepth 1 -type f ! -name SHA256SUMS -print | sort) | \
    sed "s#  $temporal/#  #" >"$temporal/SHA256SUMS"
  (cd "$temporal" && sha256sum -c SHA256SUMS >/dev/null)
  mv -- "$temporal" "$destino"
  temporal_publicacion=
}

autoprueba() {
  local origen destino stdout stderr
  raiz_autoprueba=$(mktemp -d /var/tmp/o3c-fallo-durable-autoprueba.XXXXXX)
  origen="$raiz_autoprueba/origen"
  destino="$raiz_autoprueba/evidencia"
  stdout="$raiz_autoprueba/stdout"
  stderr="$raiz_autoprueba/stderr"
  mkdir -m 700 "$origen"
  printf 'parcial\n' >"$origen/casos.tsv"
  printf 'salida integra\n' >"$stdout"
  printf 'error integro\n' >"$stderr"
  publicar "$origen" "$destino" "$stdout" "$stderr" CAP_NORMAL_021 normal 1 si \
    5,0,0,0,0 5,0,0,0,0 fea52f3ddf796991c93c85cae992ca695db1ae63 'go version go1.26.5 linux/amd64'
  [[ -d $destino && ! -e $origen/SHA256SUMS ]]
  cmp -s "$stdout" "$destino/fallo.stdout"
  cmp -s "$stderr" "$destino/fallo.stderr"
  grep -Fxq 'resultado=NO-GO' "$destino/resumen.txt"
  if grep -Fq 'resultado=GO' "$destino/resumen.txt"; then
    return 1
  fi
  grep -Fq '  fallo.stdout' "$destino/SHA256SUMS"
  grep -Fq '  fallo.stderr' "$destino/SHA256SUMS"
  grep -Fq '  fallo.tsv' "$destino/SHA256SUMS"
  grep -Fq '  resumen.txt' "$destino/SHA256SUMS"
  (cd "$destino" && sha256sum -c SHA256SUMS >/dev/null)
  if publicar "$origen" "$destino" "$stdout" "$stderr" CAP_NORMAL_022 normal 1 si \
    5,0,0,0,0 5,0,0,0,0 fea52f3ddf796991c93c85cae992ca695db1ae63 'go version go1.26.5 linux/amd64'; then
    return 1
  fi
  printf 'GO\n'
}

if [[ ${1:-} == --autoprueba ]]; then
  [[ $# -eq 1 ]] || exit 64
  autoprueba
  exit
fi
[[ ${1:-} == publicar && $# -eq 13 ]] || {
  printf 'uso: %s publicar ORIGEN DESTINO STDOUT STDERR ID MODO ESTADO PGID ANTES DESPUES HEAD GO_VERSION\n' "$0" >&2
  exit 64
}
shift
publicar "$@"
