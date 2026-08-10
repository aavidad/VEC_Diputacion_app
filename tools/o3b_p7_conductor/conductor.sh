#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo 'NO-GO uso: conductor.sh TARGET EVIDENCIA_NUEVA' >&2
  exit 2
fi
target=$(realpath "$1")
evidencia=$(realpath -m "$2")
raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
fuentes="$raiz/tools/o3b_p7_conductor/fuentes.tsv"
casos="$raiz/tools/o3b_p7_conductor/casos.tsv"
base=d9f8aeb547f5d1b3b9ab3eb786382f78ef964e28
sha_conductor=$(sha256sum "${BASH_SOURCE[0]}" | cut -d' ' -f1)
sha_matriz=$(sha256sum "$casos" | cut -d' ' -f1)
sha_fuentes_matriz=$(sha256sum "$fuentes" | cut -d' ' -f1)

[[ -d $target && ! -e $evidencia ]] || { echo 'NO-GO target/evidencia' >&2; exit 2; }
git -C "$target" cat-file -e "$base^{commit}" 2>/dev/null || { echo 'NO-GO base ausente' >&2; exit 2; }
git -C "$target" merge-base --is-ancestor "$base" HEAD || { echo 'NO-GO ascendencia base' >&2; exit 2; }
goroot=$(go env GOROOT)
go_bin="$goroot/bin/go"
[[ -x $go_bin && $($go_bin version) == 'go version go1.26.5 linux/amd64' ]] || { echo 'NO-GO toolchain' >&2; exit 2; }
mkdir -m 700 "$evidencia"
staging=$(mktemp -d "${TMPDIR:-/var/tmp}/o3b-p7.XXXXXX")
trap 'rm -rf -- "$staging"' EXIT
mkdir "$staging/src" "$staging/cache" "$staging/tmp"

archivos=()
{
  printf 'sha256\truta\n'
  while IFS=$'\t' read -r sha ruta; do
    [[ $sha == sha256 ]] && continue
    [[ $ruta != /* && -f $target/$ruta ]] || { echo "NO-GO fuente $ruta" >&2; exit 1; }
    [[ $(sha256sum "$target/$ruta" | cut -d' ' -f1) == "$sha" ]] || { echo "NO-GO hash $ruta" >&2; exit 1; }
    cp -- "$target/$ruta" "$staging/src/$(basename "$ruta")"
    archivos+=("$ruta")
    printf '%s\t%s\n' "$sha" "$ruta"
  done < "$fuentes"
} > "$evidencia/fuentes.tsv"
[[ ${#archivos[@]} -eq 22 ]] || { echo 'NO-GO cardinalidad fuentes' >&2; exit 1; }
cd "$target"

entorno=(env -i PATH="$goroot/bin:$PATH" HOME="$staging" TMPDIR="$staging/tmp" GOTMPDIR="$staging/tmp" GOCACHE="$staging/cache" GOROOT="$goroot" GOENV=off GOTOOLCHAIN=local CGO_ENABLED=0)
"${entorno[@]}" "$go_bin" test -c -buildvcs=false -o "$staging/o3b-normal" "${archivos[@]}"
env_race=(env -i PATH="$goroot/bin:$PATH" HOME="$staging" TMPDIR="$staging/tmp" GOTMPDIR="$staging/tmp" GOCACHE="$staging/cache" GOROOT="$goroot" GOENV=off GOTOOLCHAIN=local CGO_ENABLED=1)
"${env_race[@]}" "$go_bin" test -race -c -buildvcs=false -o "$staging/o3b-race" "${archivos[@]}"

sha_target=$(sha256sum "$evidencia/fuentes.tsv" | cut -d' ' -f1)
printf 'modo\tsha_binario\nnormal\t%s\nrace\t%s\n' \
  "$(sha256sum "$staging/o3b-normal" | cut -d' ' -f1)" \
  "$(sha256sum "$staging/o3b-race" | cut -d' ' -f1)" > "$evidencia/binarios.tsv"
printf 'id\tmodo\tcomando\tsha_target\testado\tstdout_bytes\tstderr_bytes\tduracion_ms\tfd_inicio\tfd_fin\thijos_inicio\thijos_fin\tzombis_inicio\tzombis_fin\tgrupos_inicio\tgrupos_fin\ttemporales_inicio\ttemporales_fin\toraculo\tresultado\n' > "$evidencia/casos.tsv"

inventario() {
  local destino_fd=$1 destino_hijos=$2 destino_zombis=$3 destino_grupos=$4 destino_temporales=$5
  local fd total_fd=0 hijos_crudos='' pid stat resto estado grupo temporal
  local total_hijos=0 total_zombis=0 total_grupos=0 total_temporales=0
  local -A grupos=()
  for fd in /proc/"$$"/fd/*; do
    [[ -e $fd ]] && ((total_fd += 1))
  done
  read -r hijos_crudos < "/proc/$$/task/$$/children" || true
  for pid in $hijos_crudos; do
    [[ $pid =~ ^[0-9]+$ ]] || continue
    stat=''
    IFS= read -r stat < "/proc/$pid/stat" || continue
    resto=${stat##*) }
    read -ra campos <<<"$resto"
    [[ ${#campos[@]} -ge 3 ]] || continue
    estado=${campos[0]}
    grupo=${campos[2]}
    ((total_hijos += 1))
    [[ $estado == Z ]] && ((total_zombis += 1))
    grupos[$grupo]=1
  done
  total_grupos=${#grupos[@]}
  shopt -s nullglob dotglob
  for temporal in "$staging/tmp"/*; do
    [[ -e $temporal ]] || continue
    ((total_temporales += 1))
  done
  shopt -u nullglob dotglob
  printf -v "$destino_fd" '%d' "$total_fd"
  printf -v "$destino_hijos" '%d' "$total_hijos"
  printf -v "$destino_zombis" '%d' "$total_zombis"
  printf -v "$destino_grupos" '%d' "$total_grupos"
  printf -v "$destino_temporales" '%d' "$total_temporales"
}

ejecutar() {
  local id=$1 modo=$2 patron=$3 oraculo=$4 bin=$5
  local out="$staging/$id.$modo.out" err="$staging/$id.$modo.err"
  local fdi fdf hi hf zi zf gi gf ti tf inicio fin estado
  inventario fdi hi zi gi ti
  inicio=${EPOCHREALTIME/./}
  set +e
  (cd "$target" && timeout --signal=KILL 180 "$bin" "-test.run=^(${patron})$" -test.count=1) >"$out" 2>"$err"
  estado=$?
  set -e
  fin=${EPOCHREALTIME/./}
  inventario fdf hf zf gf tf
  local resultado=GO
  [[ $estado -eq 0 && $fdi -eq $fdf && $hi -eq $hf && $zi -eq $zf && $gi -eq $gf && $ti -eq $tf ]] || resultado=NO-GO
  printf '%s\t%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%s\t%s\n' \
    "$id" "$modo" "testbin -test.run=^(${patron})$ -test.count=1" "$sha_target" "$estado" \
    "$(wc -c <"$out")" "$(wc -c <"$err")" "$(((fin-inicio)/1000))" "$fdi" "$fdf" "$hi" "$hf" "$zi" "$zf" "$gi" "$gf" "$ti" "$tf" "$oraculo" "$resultado" >> "$evidencia/casos.tsv"
  [[ $resultado == GO ]] || { cp "$out" "$evidencia/$id.$modo.stdout"; cp "$err" "$evidencia/$id.$modo.stderr"; return 1; }
}

for modo in normal race; do
  bin="$staging/o3b-$modo"
  while IFS=$'\t' read -r id prueba oraculo; do
    [[ $id == id || $id == O18_CIEN_CAPTURAS ]] && continue
    ejecutar "$id" "$modo" "$prueba" "$oraculo" "$bin"
  done < "$casos"
  for numero in $(seq -w 1 100); do
    ejecutar "CAP_${modo^^}_$numero" "$modo" 'TestHandoffO3bNominalConjuntoYConsumido' \
      'captura completa; inventario hijos zombis grupos temporales y FD sin delta' "$bin"
  done
done

filas=$(( $(wc -l < "$evidencia/casos.tsv") - 1 ))
[[ $filas -eq 234 ]] || { echo "NO-GO filas $filas" >&2; exit 1; }
! grep -q $'\tNO-GO$' "$evidencia/casos.tsv" || { echo 'NO-GO caso' >&2; exit 1; }
cat > "$evidencia/resumen.txt" <<EOF
resultado=GO
base=$base
go_version=$($go_bin version)
sha_conductor=$sha_conductor
sha_matriz=$sha_matriz
sha_fuentes_matriz=$sha_fuentes_matriz
fuentes=22
sha_target=$sha_target
oraculos=17_por_modo
capturas_normal=100
capturas_race=100
casos_totales=$filas
residuos=cero
EOF
: > "$evidencia/residuos.txt"
sumas="$staging/SHA256SUMS"
(cd "$evidencia" && sha256sum binarios.tsv casos.tsv fuentes.tsv residuos.txt resumen.txt | sort -k2) > "$sumas"
mv "$sumas" "$evidencia/SHA256SUMS"
(cd "$evidencia" && sha256sum -c SHA256SUMS >/dev/null)
echo GO
