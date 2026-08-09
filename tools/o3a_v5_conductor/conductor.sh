#!/usr/bin/env bash
set -euo pipefail

LC_ALL=C
export LC_ALL

uso() {
  printf 'uso: %s [TARGET] [DIRECTORIO_EVIDENCIA]\n' "$0" >&2
  exit 64
}

(($# <= 2)) || uso
directorio=$(cd -- "$(dirname -- "$0")" && pwd -P)
target=${1:-/srv/fabrica/proyectos/VEC_Diputacion_app/.worktrees/o3a-v5-auditoria-20260809}
destino=${2:-$directorio/evidencia-durable}
ledger=${CND_LEDGER:-$directorio/fuentes_v5.tsv}
base_rel=deploy/postgresql/autorizacion_atestada_v3/pruebas_sql
base="$target/$base_rel"
prefijo=supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1
watchdog=${CND_WATCHDOG_SEGUNDOS:-300}

[[ -d $target && -d $base && -f $ledger ]] || { printf 'NO-GO entrada_ausente\n' >&2; exit 2; }
[[ $watchdog =~ ^[1-9][0-9]*$ ]] || { printf 'NO-GO watchdog_invalido\n' >&2; exit 2; }
version_go=$(go version)
[[ $version_go == 'go version go1.26.5 '* ]] || { printf 'NO-GO version_go=%s\n' "$version_go" >&2; exit 2; }

umask 077
raiz=$(mktemp -d "${TMPDIR:-/var/tmp}/o3a-cnd-v5-11.XXXXXX")
# shellcheck disable=SC2329
limpiar() { rm -rf -- "$raiz"; }
trap limpiar EXIT HUP INT TERM
staging="$raiz/target"
mkdir -p "$staging/$base_rel"

verificado="$raiz/fuentes-verificadas.tsv"
printf 'archivo\tlineas\tsha256\n' >"$verificado"
declare -A declaradas=()
while IFS=$'\t' read -r archivo lineas sha; do
  [[ $archivo == archivo ]] && continue
  [[ -n $archivo && $lineas =~ ^[0-9]+$ && $sha =~ ^[0-9a-f]{64}$ && -z ${declaradas[$archivo]+x} ]] || {
    printf 'NO-GO ledger_invalido=%s\n' "$archivo" >&2; exit 2;
  }
  declaradas[$archivo]=1
  fuente="$base/$archivo"
  [[ -f $fuente ]] || { printf 'NO-GO fuente_ausente=%s\n' "$archivo" >&2; exit 2; }
  lineas_reales=$(wc -l <"$fuente")
  sha_real=$(sha256sum "$fuente" | awk '{print $1}')
  [[ $lineas_reales -eq $lineas && $sha_real == "$sha" ]] || {
    printf 'NO-GO huella=%s lineas=%s/%s sha=%s/%s\n' "$archivo" "$lineas_reales" "$lineas" "$sha_real" "$sha" >&2
    exit 2
  }
  printf '%s\t%s\t%s\n' "$archivo" "$lineas" "$sha" >>"$verificado"
  cp -- "$fuente" "$staging/$base_rel/$archivo"
done <"$ledger"

[[ ${#declaradas[@]} -eq 10 ]] || { printf 'NO-GO fuentes_declaradas=%s esperado=10\n' "${#declaradas[@]}" >&2; exit 2; }
mapfile -t encontradas < <(find "$base" -maxdepth 1 -type f -name "$prefijo*.go" -printf '%f\n' | sort)
[[ ${#encontradas[@]} -eq 10 ]] || { printf 'NO-GO fuentes_target=%s esperado=10\n' "${#encontradas[@]}" >&2; exit 2; }
for archivo in "${encontradas[@]}"; do
  [[ -n ${declaradas[$archivo]+x} ]] || { printf 'NO-GO fuente_no_declarada=%s\n' "$archivo" >&2; exit 2; }
done

mapfile -t copia < <(find "$staging/$base_rel" -maxdepth 1 -type f -name "$prefijo*.go" -print | sort)
if ! gofmt -d "${copia[@]}" >"$raiz/gofmt.diff"; then
  printf 'NO-GO gofmt_error\n' >&2
  exit 2
fi
[[ ! -s $raiz/gofmt.diff ]] || { printf 'NO-GO gofmt_diff\n' >&2; exit 2; }

mkdir -p "$destino"
[[ -z $(find "$destino" -mindepth 1 -maxdepth 1 -print -quit) ]] || {
  printf 'NO-GO evidencia_no_vacia=%s\n' "$destino" >&2; exit 2;
}

ndjson="$raiz/casos.ndjson"
manifiesto="$raiz/manifiesto.tsv"
printf 'bloque\tmodo\tsha_script\tinicio_utc\tduracion_ms\testado_script\tfilas\tresultado\n' >"$manifiesto"
: >"$ndjson"

json_tsv() {
  local bloque=$1 modo=$2 sha_script=$3 tsv=$4 duracion=$5
  awk -F '\t' -v bloque="$bloque" -v modo="$modo" -v sha_script="$sha_script" -v duracion="$duracion" '
    function esc(s) { gsub(/\\/, "\\\\", s); gsub(/"/, "\\\"", s); gsub(/\r/, "\\r", s); gsub(/\n/, "\\n", s); return s }
    NR==1 { for(i=1;i<=NF;i++) h[i]=$i; next }
    NF {
      printf "{\"bloque\":\"%s\",\"modo\":\"%s\",\"sha_script\":\"%s\",\"duracion_bloque_ms\":%s,\"campos\":{", esc(bloque), esc(modo), sha_script, duracion
      for(i=1;i<=NF;i++) printf "%s\"%s\":\"%s\"", (i==1?"":","), esc(h[i]), esc($i)
      print "}}"
    }
  ' "$tsv" >>"$ndjson"
}

ejecutar_bloque_aislado() {
  local modo=$1 ruta=$2 staging=$3 salida=$4 fd ruta_fd
  for ruta_fd in /proc/self/fd/*; do
    fd=${ruta_fd##*/}
    [[ $fd =~ ^[0-9]+$ && $fd -ge 3 ]] || continue
    eval "exec ${fd}>&-" 2>/dev/null || true
  done
  if [[ $modo == race ]]; then
    exec env CND_RUNTIME_TARGET="$target" CND_RACE=1 CND_CGO_ENABLED=1 \
      timeout --foreground -k 10s "${watchdog}s" "$ruta" "$staging" "$salida"
  fi
  exec env CND_RUNTIME_TARGET="$target" CND_RACE=0 CND_CGO_ENABLED=0 \
    timeout --foreground -k 10s "${watchdog}s" "$ruta" "$staging" "$salida"
}

scripts=(
  conductor_c01_c07.sh
  conductor_c02_interleavings.sh
  conductor_c08_c11_c14.sh
  conductor_c10_c13.sh
  conductor_c15_c21.sh
  conductor_c19.sh
  conductor_c20.sh
)
fd_inicio=$(find "/proc/$$/fd" -mindepth 1 -maxdepth 1 -type l 2>/dev/null | wc -l)
fallos=0
for modo in normal race; do
  for script in "${scripts[@]}"; do
    ruta="$directorio/$script"
    [[ -x $ruta ]] || { printf 'NO-GO script_ausente=%s\n' "$script" >&2; exit 2; }
    bloque=${script#conductor_}; bloque=${bloque%.sh}
    salida="$raiz/${bloque}_${modo}.tsv"
    log="$raiz/${bloque}_${modo}.log"
    sha_script=$(sha256sum "$ruta" | awk '{print $1}')
    inicio=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    t0=$(date +%s%N)
    set +e
    (ejecutar_bloque_aislado "$modo" "$ruta" "$staging" "$salida") >"$log" 2>&1
    estado=$?
    set -e
    t1=$(date +%s%N); duracion=$(((t1-t0)/1000000))
    filas=0; resultado=NO-GO
    if [[ -f $salida ]]; then
      filas=$(awk 'END{print (NR > 0 ? NR - 1 : 0)}' "$salida")
      if [[ $estado -eq 0 && $filas -gt 0 ]] && ! grep -Eq '(^|[[:space:]])(NO-GO|SKIP)([[:space:]]|$)' "$salida"; then resultado=GO; fi
      json_tsv "$bloque" "$modo" "$sha_script" "$salida" "$duracion"
    fi
    printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$bloque" "$modo" "$sha_script" "$inicio" "$duracion" "$estado" "$filas" "$resultado" >>"$manifiesto"
    [[ $resultado == GO ]] || fallos=1
  done
done

residuos="$raiz/residuos.txt"
pgrep -af "$raiz|$staging" >"$residuos" || true
[[ ! -s $residuos ]] || fallos=1
fd_fin=$(find "/proc/$$/fd" -mindepth 1 -maxdepth 1 -type l 2>/dev/null | wc -l)

sha_conductor=$(sha256sum "$directorio/conductor.sh" | awk '{print $1}')
sha_fuentes=$(sha256sum "$verificado" | awk '{print $1}')
sha_casos=$(sha256sum "$ndjson" | awk '{print $1}')
sha_manifest=$(sha256sum "$manifiesto" | awk '{print $1}')
casos=$(wc -l <"$ndjson")
go=$(awk 'NR > 1 && $8 != "GO" {n++} END{print n+0}' "$manifiesto")
[[ $casos -gt 0 && $go -eq 0 ]] || fallos=1
resultado_global=GO; [[ $fallos -eq 0 ]] || resultado_global=NO-GO

resumen="$raiz/resumen.txt"
{
  printf 'resultado=%s\n' "$resultado_global"
  printf 'go_version=%s\n' "$version_go"
  printf 'target=%s\n' "$(cd "$target" && pwd -P)"
  printf 'head=%s\n' "$(git -C "$target" rev-parse HEAD 2>/dev/null || printf no_git)"
  printf 'sha_conductor=%s\n' "$sha_conductor"
  printf 'fuentes=10\nsha_fuentes=%s\n' "$sha_fuentes"
  printf 'bloques=14\ncasos_registrados=%s\n' "$casos"
  printf 'sha_casos=%s\nsha_manifiesto=%s\n' "$sha_casos" "$sha_manifest"
  printf 'watchdog_segundos=%s\nfd_conductor_inicio=%s\nfd_conductor_fin=%s\n' "$watchdog" "$fd_inicio" "$fd_fin"
  printf 'residuos=%s\n' "$([[ -s $residuos ]] && printf presentes || printf cero)"
} >"$resumen"

cp -- "$verificado" "$destino/fuentes-verificadas.tsv"
cp -- "$manifiesto" "$destino/manifiesto.tsv"
cp -- "$ndjson" "$destino/casos.ndjson"
cp -- "$resumen" "$destino/resumen.txt"
cp -- "$residuos" "$destino/residuos.txt"
for f in "$raiz"/*.log "$raiz"/*.tsv; do
  [[ -f $f ]] && cp -- "$f" "$destino/"
done
sha256sum "$destino"/* | sort -k2 >"$destino/SHA256SUMS"
printf '%s\n' "$resultado_global"
exit "$fallos"
