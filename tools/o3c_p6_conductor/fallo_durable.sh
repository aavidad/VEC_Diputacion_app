#!/usr/bin/env bash
set -euo pipefail

raiz_autoprueba=
temporal_publicacion=

limpiar() {
  [[ -z $temporal_publicacion ]] || rm -rf -- "$temporal_publicacion"
  [[ -z $raiz_autoprueba ]] || rm -rf -- "$raiz_autoprueba"
}
trap limpiar EXIT

renombrar_sin_reemplazo() {
  local origen=$1 destino=$2 huella_origen huella_destino

  [[ -d $origen && ! -L $origen ]] || return 2
  huella_origen=$(stat -c '%d:%i' -- "$origen") || return 2
  mv -n -T -- "$origen" "$destino" || return 2
  [[ ! -e $origen && ! -L $origen && -d $destino && ! -L $destino ]] || return 2
  huella_destino=$(stat -c '%d:%i' -- "$destino") || return 2
  [[ $huella_destino == "$huella_origen" ]] || return 2
}

publicar() {
  local origen=$1 destino=$2 stdout=$3 stderr=$4 id=$5 modo=$6 estado=$7 pgid=$8
  local antes=$9 despues=${10} head=${11} version_go=${12}
  local padre nombre temporal archivo

  [[ -d $origen && ! -L $origen && -f $stdout && -f $stderr ]] || return 2
  [[ ! -e $destino && ! -L $destino ]] || return 2
  [[ $id =~ ^[A-Z0-9_]+$ && $modo =~ ^(normal|race)$ && $estado =~ ^[0-9]+$ ]] || return 2
  [[ $pgid =~ ^(si|no)$ && $antes =~ ^[0-9]+(,[0-9]+){4}$ && $despues =~ ^[0-9]+(,[0-9]+){4}$ ]] || return 2
  [[ $head =~ ^[0-9a-f]{40}$ && $version_go == 'go version go1.26.5 linux/amd64' ]] || return 2
  padre=$(dirname "$destino")
  nombre=$(basename "$destino")
  [[ -d $padre && ! -L $padre ]] || return 2

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
  (
    local -a archivos=()
    cd "$temporal"
    mapfile -d '' -t archivos < <(find . -maxdepth 1 -type f ! -name SHA256SUMS -printf '%P\0' | sort -z)
    : >SHA256SUMS
    for archivo in "${archivos[@]}"; do
      sha256sum -- "$archivo"
    done >>SHA256SUMS
    sha256sum -c SHA256SUMS >/dev/null
  )
  renombrar_sin_reemplazo "$temporal" "$destino" || return 2
  temporal_publicacion=
}

autoprueba() {
  local origen destino stdout stderr colision_origen colision_destino redireccion huella_guardia
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
  colision_origen="$raiz_autoprueba/colision-origen"
  colision_destino="$raiz_autoprueba/colision-destino"
  mkdir -m 700 "$colision_origen" "$colision_destino"
  printf 'origen\n' >"$colision_origen/paquete"
  huella_guardia=$(stat -c '%d:%i' -- "$colision_destino")
  if renombrar_sin_reemplazo "$colision_origen" "$colision_destino"; then
    return 1
  fi
  [[ -d $colision_origen && -f $colision_origen/paquete ]]
  [[ -d $colision_destino ]]
  [[ $(stat -c '%d:%i' -- "$colision_destino") == "$huella_guardia" ]]
  [[ -z $(find "$colision_destino" -mindepth 1 -maxdepth 1 -print -quit) ]]
  redireccion="$raiz_autoprueba/redireccion"
  mkdir -m 700 "$redireccion"
  ln -s -- "$redireccion" "$raiz_autoprueba/destino-enlace"
  if renombrar_sin_reemplazo "$colision_origen" "$raiz_autoprueba/destino-enlace"; then
    return 1
  fi
  [[ -L $raiz_autoprueba/destino-enlace && -d $colision_origen ]]
  [[ -z $(find "$redireccion" -mindepth 1 -maxdepth 1 -print -quit) ]]
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
