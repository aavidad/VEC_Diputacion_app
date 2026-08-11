#!/usr/bin/env bash
set -euo pipefail

[[ $# -eq 2 ]] || { printf 'NO-GO uso: conductor.sh TARGET EVIDENCIA_NUEVA\n' >&2; exit 2; }
exec 9>/var/tmp/o3c-p6-conductor.lock
flock -n 9 || { printf 'NO-GO conductor concurrente\n' >&2; exit 2; }
target=$(realpath "$1")
evidencia=$(realpath -m "$2")
raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
unidad="$raiz/tools/o3c_p6_conductor"
fuentes="$unidad/fuentes.tsv"
casos="$unidad/casos.tsv"
base=c0f2a9945ed2fc5648980ee48b91424a04977655

[[ -d $target && ! -e $evidencia ]] || { printf 'NO-GO target/evidencia\n' >&2; exit 2; }
git -C "$target" cat-file -e "$base^{commit}" 2>/dev/null || { printf 'NO-GO base ausente\n' >&2; exit 2; }
git -C "$target" merge-base --is-ancestor "$base" HEAD || { printf 'NO-GO ascendencia\n' >&2; exit 2; }
if ! git -C "$target" diff --quiet || ! git -C "$target" diff --cached --quiet; then
  printf 'NO-GO checkout tracked sucio\n' >&2
  exit 2
fi
goroot=$(go env GOROOT)
go_bin="$goroot/bin/go"
[[ -x $go_bin && $($go_bin version) == 'go version go1.26.5 linux/amd64' ]] || { printf 'NO-GO toolchain\n' >&2; exit 2; }

staging=$(mktemp -d /var/tmp/o3c-p6.XXXXXX)
trap 'rm -rf -- "$staging"' EXIT
mkdir "$staging/cache" "$staging/tmp"
destino_evidencia=$evidencia
evidencia="$staging/evidencia"
mkdir -m 700 "$evidencia"
archivos=()
{
  printf 'sha256\truta\n'
  while IFS=$'\t' read -r sha ruta; do
    [[ $sha == sha256 ]] && continue
    [[ $ruta != /* && -f $target/$ruta ]] || { printf 'NO-GO fuente %s\n' "$ruta" >&2; exit 1; }
    [[ $(sha256sum "$target/$ruta" | cut -d' ' -f1) == "$sha" ]] || { printf 'NO-GO hash %s\n' "$ruta" >&2; exit 1; }
    archivos+=("$ruta")
    printf '%s\t%s\n' "$sha" "$ruta"
  done < "$fuentes"
} > "$evidencia/fuentes.tsv"
[[ ${#archivos[@]} -eq 32 ]] || { printf 'NO-GO cardinalidad fuentes\n' >&2; exit 1; }

cd "$target"
entorno=(env -i PATH="$goroot/bin:/usr/bin:/bin" HOME="$staging" TMPDIR="$staging/tmp" GOTMPDIR="$staging/tmp" GOCACHE="$staging/cache" GOROOT="$goroot" GOENV=off GOTOOLCHAIN=local CGO_ENABLED=0)
"${entorno[@]}" "$go_bin" test -c -buildvcs=false -o "$staging/o3c-normal" "${archivos[@]}"
entorno_race=(env -i PATH="$goroot/bin:/usr/bin:/bin" HOME="$staging" TMPDIR="$staging/tmp" GOTMPDIR="$staging/tmp" GOCACHE="$staging/cache" GOROOT="$goroot" GOENV=off GOTOOLCHAIN=local CGO_ENABLED=1)
"${entorno_race[@]}" "$go_bin" test -race -c -buildvcs=false -o "$staging/o3c-race" "${archivos[@]}"

sha_target=$(sha256sum "$evidencia/fuentes.tsv" | cut -d' ' -f1)
printf 'modo\tsha_binario\nnormal\t%s\nrace\t%s\n' "$(sha256sum "$staging/o3c-normal" | cut -d' ' -f1)" "$(sha256sum "$staging/o3c-race" | cut -d' ' -f1)" > "$evidencia/binarios.tsv"
cabecera='id\tmodo\tcomando\tsha_target\testado\tstdout_bytes\tstderr_bytes\tduracion_ms\tfd_inicio\tfd_fin\thijos_inicio\thijos_fin\tzombis_inicio\tzombis_fin\tgrupos_inicio\tgrupos_fin\ttemporales_inicio\ttemporales_fin\tgrupo_ejecucion_esrch\toraculo\tresultado'
printf '%b\n' "$cabecera" > "$evidencia/casos.tsv"
printf '%b\n' "${cabecera/estado\\tstdout_bytes\\tstderr_bytes/estado\\tstdout_bytes\\tstderr_bytes\\tstdout_eof\\tstderr_eof\\tno_retorno}" > "$evidencia/bf_directos.tsv"

inventario() {
  local dfd=$1 dh=$2 dz=$3 dg=$4 dt=$5 fd hijos='' pid stat resto estado grupo temporal
  local nfd=0 nh=0 nz=0 nt=0
  local -A grupos=()
  for fd in "/proc/$$/fd"/*; do [[ -e $fd ]] && ((nfd+=1)); done
  read -r hijos < "/proc/$$/task/$$/children" || true
  for pid in $hijos; do
    [[ $pid =~ ^[0-9]+$ ]] || continue
    IFS= read -r stat < "/proc/$pid/stat" || continue
    resto=${stat##*) }; read -ra campos <<< "$resto"
    [[ ${#campos[@]} -ge 3 ]] || continue
    estado=${campos[0]}; grupo=${campos[2]}; ((nh+=1)); [[ $estado == Z ]] && ((nz+=1)); grupos[$grupo]=1
  done
  shopt -s nullglob dotglob
  for temporal in "$staging/tmp"/*; do [[ -e $temporal ]] && ((nt+=1)); done
  shopt -u nullglob dotglob
  printf -v "$dfd" %d "$nfd"; printf -v "$dh" %d "$nh"; printf -v "$dz" %d "$nz"; printf -v "$dg" %d "${#grupos[@]}"; printf -v "$dt" %d "$nt"
}

# Cada caso vive en una sesión/grupo nuevo. Tras esperar al líder, cualquier
# proceso que aún responda en el grupo es residuo: se mata solo para contener
# el fixture, pero la fila conserva NO-GO y nunca se acepta como evidencia.
ejecutar_aislado() {
  local out=$1 err=$2; shift 2
  local lider
  set +e
  # El lock pertenece solo al conductor; nunca se hereda como sexto FD del
  # target ni contamina el inventario que se pretende acreditar.
  setsid timeout --signal=KILL 180 "$@" 9>&- >"$out" 2>"$err" &
  lider=$!
  wait "$lider"
  estado_aislado=$?
  set -e
  grupo_cero_aislado=si
  if kill -0 -- "-$lider" 2>/dev/null; then
    grupo_cero_aislado=no
    kill -KILL -- "-$lider" 2>/dev/null || true
    wait "$lider" 2>/dev/null || true
  fi
  # La contención se acredita hasta ESRCH, no con un único sondeo sujeto a
  # carrera. Haber necesitado contener sigue siendo NO-GO aunque converja.
  for _ in {1..100}; do
    kill -0 -- "-$lider" 2>/dev/null || break
    sleep 0.01
  done
  kill -0 -- "-$lider" 2>/dev/null && grupo_cero_aislado=no
  return 0
}

ejecutar() {
  local id=$1 modo=$2 patron=$3 oraculo=$4 bin=$5 out="$staging/out" err="$staging/err"
  local fdi fdf hi hf zi zf gi gf ti tf inicio fin estado resultado=GO
  inventario fdi hi zi gi ti; inicio=${EPOCHREALTIME/./}
  ejecutar_aislado "$out" "$err" "$bin" "-test.run=^(${patron})$" -test.count=1; estado=$estado_aislado
  fin=${EPOCHREALTIME/./}; inventario fdf hf zf gf tf
  [[ $estado -eq 0 && $grupo_cero_aislado == si && $fdi -eq $fdf && $hi -eq $hf && $zi -eq $zf && $gi -eq $gf && $ti -eq $tf ]] || resultado=NO-GO
  printf '%s\t%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%s\t%s\t%s\n' "$id" "$modo" "testbin -test.run=^(${patron})$ -test.count=1" "$sha_target" "$estado" "$(wc -c <"$out")" "$(wc -c <"$err")" "$(((fin-inicio)/1000))" "$fdi" "$fdf" "$hi" "$hf" "$zi" "$zf" "$gi" "$gf" "$ti" "$tf" "$grupo_cero_aislado" "$oraculo" "$resultado" >> "$evidencia/casos.tsv"
  [[ $resultado == GO ]] || { printf 'NO-GO caso=%s modo=%s estado=%d stdout=%d stderr=%d grupo=%s inventario=%d/%d,%d/%d,%d/%d,%d/%d,%d/%d\n' "$id" "$modo" "$estado" "$(wc -c <"$out")" "$(wc -c <"$err")" "$grupo_cero_aislado" "$fdi" "$fdf" "$hi" "$hf" "$zi" "$zf" "$gi" "$gf" "$ti" "$tf" >&2; return 1; }
}

ejecutar_bf() {
  local id=$1 modo=$2 variable=$3 valor=$4 prueba=$5 oraculo=$6 bin=$7 out="$staging/out" err="$staging/err"
  local fdi fdf hi hf zi zf gi gf ti tf inicio fin estado so se resultado=GO
  inventario fdi hi zi gi ti; inicio=${EPOCHREALTIME/./}
  ejecutar_aislado "$out" "$err" env "$variable=$valor" "$bin" "-test.run=^${prueba}$" -test.count=1; estado=$estado_aislado
  fin=${EPOCHREALTIME/./}; inventario fdf hf zf gf tf; so=$(wc -c <"$out"); se=$(wc -c <"$err")
  [[ $estado -eq 65 && $so -eq 0 && $se -eq 0 && $grupo_cero_aislado == si && $fdi -eq $fdf && $hi -eq $hf && $zi -eq $zf && $gi -eq $gf && $ti -eq $tf ]] || resultado=NO-GO
  printf '%s\t%s\t%s\t%s\t%d\t%d\t%d\tsi\tsi\tsi\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%s\t%s\t%s\n' "$id" "$modo" "env $variable=$valor testbin -test.run=^${prueba}$ -test.count=1" "$sha_target" "$estado" "$so" "$se" "$(((fin-inicio)/1000))" "$fdi" "$fdf" "$hi" "$hf" "$zi" "$zf" "$gi" "$gf" "$ti" "$tf" "$grupo_cero_aislado" "$oraculo" "$resultado" >> "$evidencia/bf_directos.tsv"
  [[ $resultado == GO ]] || { printf 'NO-GO BF=%s modo=%s estado=%d stdout=%d stderr=%d grupo=%s\n' "$id" "$modo" "$estado" "$so" "$se" "$grupo_cero_aislado" >&2; return 1; }
}

for modo in normal race; do
  bin="$staging/o3c-$modo"
  while IFS=$'\t' read -r id prueba oraculo; do
    [[ $id == id ]] && continue
    ejecutar "$id" "$modo" "$prueba" "$oraculo" "$bin"
  done < "$casos"
  ejecutar_bf C01_BF_AUTO "$modo" O3C_P1_FATAL auto TestAutoridadO3cRechazosFatales 'CF entrada: 65 EOF no retorno stdout stderr cero' "$bin"
  ejecutar_bf C08_BF_LEASE "$modo" O3C_P2_FATAL 1 TestRevalidacionO3cLeaseInacreditableEsFatal 'CF lease: 65 EOF no retorno stdout stderr cero' "$bin"
  ejecutar_bf C17_BF_PARTICION "$modo" O3C_P5_CASO particion TestHandoffO3cP5CasosAislados 'CF particion owners: 65 EOF no retorno stdout stderr cero' "$bin"
  for numero in $(seq -w 1 100); do ejecutar "CAP_${modo^^}_$numero" "$modo" TestHandoffO3cP5CasosAislados 'captura completa con inventarios y residuos delta cero' "$bin"; done
done

filas=$(( $(wc -l < "$evidencia/casos.tsv") - 1 )); [[ $filas -eq 244 ]] || { printf 'NO-GO filas %d\n' "$filas" >&2; exit 1; }
filas_bf=$(( $(wc -l < "$evidencia/bf_directos.tsv") - 1 )); [[ $filas_bf -eq 6 ]] || { printf 'NO-GO BF %d\n' "$filas_bf" >&2; exit 1; }
! grep -q $'\tNO-GO$' "$evidencia/casos.tsv" "$evidencia/bf_directos.tsv" || { printf 'NO-GO resultados\n' >&2; exit 1; }
: > "$evidencia/residuos.txt"
cat > "$evidencia/resumen.txt" <<EOF
resultado=GO
base=$base
go_version=$($go_bin version)
sha_conductor=$(sha256sum "${BASH_SOURCE[0]}" | cut -d' ' -f1)
sha_matriz=$(sha256sum "$casos" | cut -d' ' -f1)
sha_fuentes=$(sha256sum "$fuentes" | cut -d' ' -f1)
sha_target=$sha_target
fuentes=32
oraculos=22_por_modo
capturas_normal=100
capturas_race=100
casos_totales=$filas
bf_directos=$filas_bf
bf_estado_eof_no_retorno_0_0=si
residuos=cero
EOF
(cd "$evidencia" && sha256sum bf_directos.tsv binarios.tsv casos.tsv fuentes.tsv residuos.txt resumen.txt | sort -k2) > "$staging/SHA256SUMS"
mv "$staging/SHA256SUMS" "$evidencia/SHA256SUMS"
(cd "$evidencia" && sha256sum -c SHA256SUMS >/dev/null)
mv "$evidencia" "$destino_evidencia"
printf 'GO\n'
