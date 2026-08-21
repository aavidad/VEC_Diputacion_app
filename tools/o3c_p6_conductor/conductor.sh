#!/usr/bin/env bash
set -euo pipefail

[[ $# -eq 2 ]] || { printf 'NO-GO uso: conductor.sh TARGET EVIDENCIA_NUEVA\n' >&2; exit 2; }
go_entrada=$(type -P go)
[[ $go_entrada == /* && -f $go_entrada && ! -L $go_entrada ]] || { printf 'NO-GO go no acreditable\n' >&2; exit 2; }
goroot=$(GOENV=off GOTOOLCHAIN=local "$go_entrada" env GOROOT)
go_bin="$goroot/bin/go"
[[ -f $go_bin && ! -L $go_bin && $(GOENV=off GOTOOLCHAIN=local "$go_bin" version) == 'go version go1.26.5 linux/amd64' ]] || {
  printf 'NO-GO toolchain\n' >&2
  exit 2
}
PATH="$goroot/bin:/usr/bin:/bin"
export PATH GOENV=off GOTOOLCHAIN=local
exec 9>/var/tmp/o3c-p6-conductor.lock
flock -n 9 || { printf 'NO-GO conductor concurrente\n' >&2; exit 2; }
target=$(realpath "$1")
evidencia=$(realpath -m "$2")
raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
unidad="$raiz/tools/o3c_p6_conductor"
fuentes="$unidad/fuentes.tsv"
casos="$unidad/casos.tsv"
publicador_fallo="$unidad/fallo_durable.sh"
base=c0f2a9945ed2fc5648980ee48b91424a04977655

[[ -d $target && ! -L $target && ! -e $evidencia && ! -L $evidencia ]] || { printf 'NO-GO target/evidencia\n' >&2; exit 2; }
[[ -x $publicador_fallo ]] || { printf 'NO-GO publicador_fallo\n' >&2; exit 2; }
git -C "$target" cat-file -e "$base^{commit}" 2>/dev/null || { printf 'NO-GO base ausente\n' >&2; exit 2; }
git -C "$target" merge-base --is-ancestor "$base" HEAD || { printf 'NO-GO ascendencia\n' >&2; exit 2; }
head_inicial=$(git -C "$target" rev-parse HEAD)
tree_inicial=$(git -C "$target" rev-parse 'HEAD^{tree}')
if [[ -n $(git -C "$target" status --porcelain=v1 --untracked-files=all) ]] ||
  ! git -C "$target" diff --quiet || ! git -C "$target" diff --cached --quiet; then
  printf 'NO-GO checkout tracked sucio\n' >&2
  exit 2
fi

staging=$(mktemp -d /var/tmp/o3c-p6.XXXXXX)
temporal_publicacion_go=
helper_bin_noreplace=
sha_helper_noreplace_global=
huella_padre_publicacion=
destino_publicacion_go=
limpiar_temporales_propios() {
  [[ -z $temporal_publicacion_go ]] || rm -rf -- "$temporal_publicacion_go"
  [[ -z $staging ]] || rm -rf -- "$staging"
}
trap limpiar_temporales_propios EXIT
runtime_tmp="$staging/runtime-tmp"
build_tmp="$staging/build-tmp"
snapshot="$staging/snapshot"
mkdir "$staging/cache" "$build_tmp" "$runtime_tmp"
mkdir -m 700 "$snapshot"
destino_evidencia=$evidencia
evidencia="$staging/evidencia"
mkdir -m 700 "$evidencia"

printf 'nombre\truta\tsha256\n' > "$evidencia/utilidades.tsv"
printf 'go\t%s\t%s\n' "$go_bin" "$(sha256sum "$go_bin" | cut -d' ' -f1)" >> "$evidencia/utilidades.tsv"
for utilidad in bash cat chmod cp cut dirname env flock git grep mkdir mktemp realpath rm seq setsid sha256sum sleep sort stat timeout wc; do
  ruta_utilidad=$(type -P "$utilidad")
  [[ $ruta_utilidad == /* && -f $ruta_utilidad && ! -L $ruta_utilidad && -x $ruta_utilidad ]] || {
    printf 'NO-GO utilidad %s\n' "$utilidad" >&2
    exit 1
  }
  printf '%s\t%s\t%s\n' "$utilidad" "$ruta_utilidad" "$(sha256sum "$ruta_utilidad" | cut -d' ' -f1)" >> "$evidencia/utilidades.tsv"
done

archivos=()
declare -A rutas_ledger=()
casos_privados="$staging/casos.tsv"
fuentes_privadas="$staging/fuentes.tsv"
cp -- "$casos" "$casos_privados"
cp -- "$fuentes" "$fuentes_privadas"
chmod 0400 "$casos_privados" "$fuentes_privadas"
[[ -f $casos_privados && ! -L $casos_privados && -f $fuentes_privadas && ! -L $fuentes_privadas ]] || {
  printf 'NO-GO ledger privado no acreditable\n' >&2
  exit 1
}
sha_matriz_privada=$(sha256sum "$casos_privados" | cut -d' ' -f1)
sha_fuentes_privadas=$(sha256sum "$fuentes_privadas" | cut -d' ' -f1)
casos=$casos_privados
fuentes=$fuentes_privadas
{
  printf 'sha256\truta\n'
  while IFS=$'\t' read -r sha ruta; do
    [[ $sha == sha256 ]] && continue
    [[ $ruta != /* && $ruta != ../* && $ruta != */../* ]] || { printf 'NO-GO ruta fuente %s\n' "$ruta" >&2; exit 1; }
    [[ -z ${rutas_ledger[$ruta]+presente} ]] || { printf 'NO-GO ruta duplicada %s\n' "$ruta" >&2; exit 1; }
    rutas_ledger[$ruta]=1
    origen="$target/$ruta"
    destino="$snapshot/$ruta"
    [[ -f $origen && ! -L $origen && $(stat -c '%F' -- "$origen") == 'regular file' ]] || {
      printf 'NO-GO fuente %s\n' "$ruta" >&2
      exit 1
    }
    mkdir -p "${destino%/*}"
    cp --reflink=never -- "$origen" "$destino"
    chmod 0400 "$destino"
    [[ -f $destino && ! -L $destino && $(stat -c '%F' -- "$destino") == 'regular file' ]] || {
      printf 'NO-GO snapshot %s\n' "$ruta" >&2
      exit 1
    }
    [[ $(sha256sum "$destino" | cut -d' ' -f1) == "$sha" ]] || { printf 'NO-GO hash snapshot %s\n' "$ruta" >&2; exit 1; }
    archivos+=("$ruta")
    printf '%s\t%s\n' "$sha" "$ruta"
  done < "$fuentes"
} > "$evidencia/fuentes.tsv"
[[ ${#archivos[@]} -eq 32 ]] || { printf 'NO-GO cardinalidad fuentes\n' >&2; exit 1; }
head_final=$(git -C "$target" rev-parse HEAD)
tree_final=$(git -C "$target" rev-parse 'HEAD^{tree}')
[[ $head_final == "$head_inicial" && $tree_final == "$tree_inicial" &&
  -z $(git -C "$target" status --porcelain=v1 --untracked-files=all) ]] || {
  printf 'NO-GO target mutado durante snapshot\n' >&2
  exit 1
}
if ! git -C "$target" diff --quiet || ! git -C "$target" diff --cached --quiet; then
  printf 'NO-GO target mutado durante snapshot\n' >&2
  exit 1
fi

cd "$snapshot"
entorno=(env -i PATH="$goroot/bin:/usr/bin:/bin" HOME="$staging" TMPDIR="$build_tmp" GOTMPDIR="$build_tmp" GOCACHE="$staging/cache" GOROOT="$goroot" GOENV=off GOTOOLCHAIN=local CGO_ENABLED=0)
"${entorno[@]}" "$go_bin" test -c -buildvcs=false -o "$staging/o3c-normal" "${archivos[@]}"
entorno_race=(env -i PATH="$goroot/bin:/usr/bin:/bin" HOME="$staging" TMPDIR="$build_tmp" GOTMPDIR="$build_tmp" GOCACHE="$staging/cache" GOROOT="$goroot" GOENV=off GOTOOLCHAIN=local CGO_ENABLED=1)
"${entorno_race[@]}" "$go_bin" test -race -c -buildvcs=false -o "$staging/o3c-race" "${archivos[@]}"
while IFS=$'\t' read -r sha ruta; do
  [[ $sha == sha256 ]] && continue
  [[ -f $snapshot/$ruta && ! -L $snapshot/$ruta &&
    $(stat -c '%F' -- "$snapshot/$ruta") == 'regular file' &&
    $(sha256sum "$snapshot/$ruta" | cut -d' ' -f1) == "$sha" ]] || {
    printf 'NO-GO snapshot mutado tras builds %s\n' "$ruta" >&2
    exit 1
  }
done < "$fuentes"

sha_target=$(sha256sum "$evidencia/fuentes.tsv" | cut -d' ' -f1)
printf 'modo\tsha_binario\nnormal\t%s\nrace\t%s\n' "$(sha256sum "$staging/o3c-normal" | cut -d' ' -f1)" "$(sha256sum "$staging/o3c-race" | cut -d' ' -f1)" > "$evidencia/binarios.tsv"
registrar_contexto() {
  local head_publicacion=$1 tree_publicacion=$2 status_vacio=$3 unstaged_vacio=$4 staged_vacio=$5 salida=${6:-$evidencia/contexto.tsv}
  [[ -f $salida && ! -L $salida ]] || return 2
  printf 'head_inicial\thead_snapshot\thead_publicacion\ttree_inicial\ttree_snapshot\ttree_publicacion\tstatus_vacio\tunstaged_vacio\tstaged_vacio\tgo_version\teuid\tsha_conductor\tsha_publicador_fallo\tsha_matriz\tsha_fuentes\tsha_target\n%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
    "$head_inicial" "$head_final" "$head_publicacion" "$tree_inicial" "$tree_final" "$tree_publicacion" \
    "$status_vacio" "$unstaged_vacio" "$staged_vacio" "$($go_bin version)" "$EUID" \
    "$(sha256sum "${BASH_SOURCE[0]}" | cut -d' ' -f1)" \
    "$(sha256sum "$publicador_fallo" | cut -d' ' -f1)" "$sha_matriz_privada" \
    "$sha_fuentes_privadas" "$sha_target" > "$salida"
}
: > "$evidencia/contexto.tsv"
chmod 0600 "$evidencia/contexto.tsv"
registrar_contexto pendiente pendiente pendiente pendiente pendiente "$evidencia/contexto.tsv"
cabecera='id\tmodo\tcomando\tsha_target\testado\tstdout_bytes\tstderr_bytes\tduracion_ms\tfd_inicio\tfd_fin\thijos_inicio\thijos_fin\tzombis_inicio\tzombis_fin\tgrupos_inicio\tgrupos_fin\ttemporales_inicio\ttemporales_fin\ttmpdir_inicio\tresiduos_pre_limpieza\ttmpdir_fin\ttmpdir_acreditable\tcontenedor_retirado\tfd_ambiente_cerrado\tselectores_esperados\tselectores_obtenidos\tselectores_acreditados\tgrupo_ejecucion_esrch\toraculo\tresultado'
printf '%b\n' "$cabecera" > "$evidencia/casos.tsv"
printf '%b\n' "${cabecera/estado\\tstdout_bytes\\tstderr_bytes/estado\\tstdout_bytes\\tstderr_bytes\\tstdout_eof\\tstderr_eof\\tno_retorno}" > "$evidencia/bf_directos.tsv"
printf 'id_ejecucion\tmodo\tselector\tdev\tinode\tuid\tmodo_dir\tentradas_inicio\tidentidad_pre_limpieza\tresiduos_pre_limpieza\tlstat_enoent\traiz_exterior_vacia\tretirada\tresultado\n' > "$evidencia/tmpdir_selectores.tsv"

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
  for temporal in "$runtime_tmp"/*; do [[ -e $temporal ]] && ((nt+=1)); done
  shopt -u nullglob dotglob
  printf -v "$dfd" %d "$nfd"; printf -v "$dh" %d "$nh"; printf -v "$dz" %d "$nz"; printf -v "$dg" %d "${#grupos[@]}"; printf -v "$dt" %d "$nt"
}

contar_entradas_tmpdir() {
  local ruta=$1 salida=$2
  local -a entradas=()
  [[ -d $ruta && ! -L $ruta ]] || return 1
  shopt -s nullglob dotglob
  entradas=("$ruta"/*)
  shopt -u nullglob dotglob
  printf -v "$salida" %d "${#entradas[@]}"
}

retirar_entradas_tmpdir() {
  local ruta=$1 entrada
  local -a entradas=()
  [[ -d $ruta && ! -L $ruta ]] || return 1
  shopt -s nullglob dotglob
  entradas=("$ruta"/*)
  shopt -u nullglob dotglob
  for entrada in "${entradas[@]}"; do
    rm -rf -- "$entrada" || return 1
  done
}

validar_atestaciones_selectores() {
  local id=$1 modo=$2 esperados=$3 huella_esperada=$4 atestacion_fd=$5 cabecera_local linea selector huella_fd
  local -a campos=()
  local -A vistos=()
  selectores_obtenidos_aislado=0
  selectores_acreditados_aislado=no
  huella_fd=$(stat -L -c '%d:%i:%u:%a' -- "/proc/$$/fd/$atestacion_fd") || { exec {atestacion_fd}<&-; return 1; }
  [[ $huella_fd == "$huella_esperada" ]] || { exec {atestacion_fd}<&-; return 1; }
  IFS= read -r cabecera_local <&"$atestacion_fd" || { exec {atestacion_fd}<&-; return 1; }
  [[ $cabecera_local == $'selector\tdev\tinode\tuid\tmodo_dir\tentradas_inicio\tidentidad_pre_limpieza\tresiduos_pre_limpieza\tlstat_enoent\traiz_exterior_vacia\tretirada\tresultado' ]] || { exec {atestacion_fd}<&-; return 1; }
  while IFS= read -r linea <&"$atestacion_fd"; do
    [[ -n $linea ]] || { exec {atestacion_fd}<&-; return 1; }
    IFS=$'\t' read -r -a campos <<< "$linea"
    [[ ${#campos[@]} -eq 12 ]] || { exec {atestacion_fd}<&-; return 1; }
    selector=${campos[0]}
    [[ $selector =~ ^(positivo|retirada|retirada_terminal|reuso|particion|retirada_sin_ref|retirada_plazo)$ &&
      -z ${vistos[$selector]+presente} && ${campos[1]} =~ ^[0-9]+$ && ${campos[2]} =~ ^[0-9]+$ &&
      ${campos[3]} == "$EUID" && ${campos[4]} == 700 && ${campos[5]} == 0 &&
      ${campos[6]} == true && ${campos[7]} =~ ^[0-9]+$ && ${campos[8]} == true &&
      ${campos[9]} == true && ${campos[10]} == true && ${campos[11]} == GO ]] || { exec {atestacion_fd}<&-; return 1; }
    vistos[$selector]=1
    ((selectores_obtenidos_aislado+=1))
    printf '%s\t%s\t%s\n' "$id" "$modo" "$linea" >> "$evidencia/tmpdir_selectores.tsv"
  done
  [[ $selectores_obtenidos_aislado -eq $esperados ]] || { exec {atestacion_fd}<&-; return 1; }
  if [[ $esperados -eq 7 ]]; then
    for selector in positivo retirada retirada_terminal reuso particion retirada_sin_ref retirada_plazo; do
      [[ -n ${vistos[$selector]+presente} ]] || { exec {atestacion_fd}<&-; return 1; }
    done
  elif [[ $esperados -eq 1 ]]; then
    [[ -n ${vistos[particion]+presente} ]] || { exec {atestacion_fd}<&-; return 1; }
  elif [[ $esperados -ne 0 ]]; then
    exec {atestacion_fd}<&-
    return 1
  fi
  selectores_acreditados_aislado=si
  exec {atestacion_fd}<&-
}

escribir_fila() {
  local archivo=$1 primero=$2
  shift 2
  {
    printf '%s' "$primero"
    printf '\t%s' "$@"
    printf '\n'
  } >> "$archivo"
}

# Cada caso vive en una sesión/grupo nuevo. Tras esperar al líder, cualquier
# proceso que aún responda en el grupo es residuo: se mata solo para contener
# el fixture, pero la fila conserva NO-GO y nunca se acepta como evidencia.
ejecutar_aislado() {
  local id=$1 modo=$2 out=$3 err=$4 selectores_esperados=$5; shift 5
  local lider etiqueta=${id,,} runtime_aislado descriptor numero_fd huella_tmpdir_aislado atestacion_selectores atestacion_selectores_huella atestacion_fd fd_marker fd_marker_fd fd_marker_value fd_marker_huella
  [[ $etiqueta =~ ^[a-z0-9_]+$ && $modo =~ ^(normal|race)$ ]] || return 2
  runtime_aislado=$(mktemp -d "$runtime_tmp/${etiqueta}-${modo}.XXXXXX")
  chmod 0700 "$runtime_aislado"
  mkdir -m 0700 "$runtime_aislado/home" "$runtime_aislado/tmp"
  atestacion_selectores="$runtime_aislado/atestacion-selectores.tsv"
  printf 'selector\tdev\tinode\tuid\tmodo_dir\tentradas_inicio\tidentidad_pre_limpieza\tresiduos_pre_limpieza\tlstat_enoent\traiz_exterior_vacia\tretirada\tresultado\n' > "$atestacion_selectores"
  chmod 0600 "$atestacion_selectores"
  atestacion_selectores_huella=$(stat -c '%d:%i:%u:%a' -- "$atestacion_selectores")
  exec {atestacion_fd}< "$atestacion_selectores"
  tmp_inicio_aislado=-1
  residuos_pre_limpieza_aislado=-1
  tmp_fin_aislado=-1
  tmp_acreditable_aislado=no
  contenedor_retirado_aislado=no
  fd_ambiente_cerrado_aislado=no
  selectores_obtenidos_aislado=-1
  selectores_acreditados_aislado=no
  huella_tmpdir_aislado=$(stat -c '%d:%i' -- "$runtime_aislado/tmp")
  if [[ $(stat -c '%u:%a' -- "$runtime_aislado/tmp") == "$EUID:700" ]] &&
    contar_entradas_tmpdir "$runtime_aislado/tmp" tmp_inicio_aislado; then
    tmp_acreditable_aislado=si
  fi
  set +e
  fd_marker="$runtime_aislado/fd-ambiental"
  : > "$fd_marker"
  chmod 0600 "$fd_marker"
  fd_marker_huella=$(stat -c '%d:%i:%u:%a' -- "$fd_marker")
  exec {fd_marker_fd}< "$fd_marker"
  # El proceso intermedio cierra todo descriptor ambiental >=3 antes de
  # convertirse en el líder de sesión. stdout/stderr se fijan externamente.
  (
    for descriptor in /proc/self/fd/*; do
      numero_fd=${descriptor##*/}
      if [[ -e $descriptor && $numero_fd =~ ^[0-9]+$ && $numero_fd -ge 3 ]]; then
        eval "exec ${numero_fd}>&-"
      fi
    done
    fd_ambiental_residual=no
    for descriptor in /proc/self/fd/*; do
      numero_fd=${descriptor##*/}
      if [[ -e $descriptor && $numero_fd =~ ^[0-9]+$ && $numero_fd -ge 3 ]]; then
        fd_ambiental_residual=si
      fi
    done
    if [[ $fd_ambiental_residual == no ]]; then
      printf 'si\n' > "$fd_marker"
    else
      printf 'no\n' > "$fd_marker"
    fi
    if [[ $selectores_esperados -gt 0 ]]; then
      exec setsid timeout --signal=KILL 180 env -i PATH="$goroot/bin:/usr/bin:/bin" HOME="$runtime_aislado/home" \
        TMPDIR="$runtime_aislado/tmp" GOTMPDIR="$runtime_aislado/tmp" GOROOT="$goroot" GOENV=off GOTOOLCHAIN=local \
        O3C_P5_ATESTACION="$atestacion_selectores" "$@"
    fi
    exec setsid timeout --signal=KILL 180 env -i PATH="$goroot/bin:/usr/bin:/bin" HOME="$runtime_aislado/home" \
      TMPDIR="$runtime_aislado/tmp" GOTMPDIR="$runtime_aislado/tmp" GOROOT="$goroot" GOENV=off GOTOOLCHAIN=local "$@"
  ) >"$out" 2>"$err" &
  lider=$!
  wait "$lider"
  estado_aislado=$?
  set -e
  if [[ $(stat -L -c '%d:%i:%u:%a' -- "/proc/$$/fd/$fd_marker_fd") == "$fd_marker_huella" ]] &&
    IFS= read -r fd_marker_value <&"$fd_marker_fd" && [[ $fd_marker_value == si ]] &&
    ! IFS= read -r fd_marker_extra <&"$fd_marker_fd" && [[ -z ${fd_marker_extra:-} ]]; then
    fd_ambiente_cerrado_aislado=si
  fi
  exec {fd_marker_fd}<&-
  rm -f -- "$fd_marker"
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
  validar_atestaciones_selectores "$id" "$modo" "$selectores_esperados" "$atestacion_selectores_huella" "$atestacion_fd" ||
    selectores_acreditados_aislado=no
  if ! contar_entradas_tmpdir "$runtime_aislado/tmp" residuos_pre_limpieza_aislado ||
    ! retirar_entradas_tmpdir "$runtime_aislado/tmp" ||
    ! contar_entradas_tmpdir "$runtime_aislado/tmp" tmp_fin_aislado ||
    [[ $tmp_inicio_aislado -ne 0 || $tmp_fin_aislado -ne 0 ||
      $(stat -c '%d:%i:%u:%a' -- "$runtime_aislado/tmp") != "$huella_tmpdir_aislado:$EUID:700" ]]; then
    tmp_acreditable_aislado=no
  fi
  if ! rm -rf -- "$runtime_aislado" || [[ -e $runtime_aislado || -L $runtime_aislado ]]; then
    contenedor_retirado_aislado=no
  else
    contenedor_retirado_aislado=si
  fi
  return 0
}

ejecutar() {
  local id=$1 modo=$2 patron=$3 oraculo=$4 bin=$5 out="$staging/out" err="$staging/err"
  local fdi fdf hi hf zi zf gi gf ti tf inicio fin estado resultado=GO selectores_esperados=0
  case "$id" in
    C17_OWNERS|C18_O4A_OPACO|C19_RETIRADA|C20_POST_CONT|C22_RESIDUOS|CAP_*) selectores_esperados=7 ;;
  esac
  inventario fdi hi zi gi ti; inicio=${EPOCHREALTIME/./}
  ejecutar_aislado "$id" "$modo" "$out" "$err" "$selectores_esperados" \
    "$bin" "-test.run=^(${patron})$" -test.count=1; estado=$estado_aislado
  fin=${EPOCHREALTIME/./}; inventario fdf hf zf gf tf
  [[ $estado -eq 0 && $grupo_cero_aislado == si && $tmp_acreditable_aislado == si &&
    $contenedor_retirado_aislado == si && $fd_ambiente_cerrado_aislado == si &&
    $selectores_acreditados_aislado == si && $selectores_obtenidos_aislado -eq $selectores_esperados &&
    $fdi -eq $fdf && $hi -eq $hf && $zi -eq $zf && $gi -eq $gf && $ti -eq $tf ]] || resultado=NO-GO
  escribir_fila "$evidencia/casos.tsv" "$id" "$modo" "testbin -test.run=^(${patron})$ -test.count=1" \
    "$sha_target" "$estado" "$(wc -c <"$out")" "$(wc -c <"$err")" "$(((fin-inicio)/1000))" \
    "$fdi" "$fdf" "$hi" "$hf" "$zi" "$zf" "$gi" "$gf" "$ti" "$tf" \
    "$tmp_inicio_aislado" "$residuos_pre_limpieza_aislado" "$tmp_fin_aislado" "$tmp_acreditable_aislado" \
    "$contenedor_retirado_aislado" "$fd_ambiente_cerrado_aislado" "$selectores_esperados" \
    "$selectores_obtenidos_aislado" "$selectores_acreditados_aislado" "$grupo_cero_aislado" "$oraculo" "$resultado"
  if [[ $resultado != GO ]]; then
    "$publicador_fallo" publicar "$evidencia" "$destino_evidencia" "$out" "$err" "$id" "$modo" "$estado" "$grupo_cero_aislado" \
      "$fdi,$hi,$zi,$gi,$ti" "$fdf,$hf,$zf,$gf,$tf" "$head_inicial" "$($go_bin version)"
    printf 'NO-GO caso=%s modo=%s estado=%d stdout=%d stderr=%d grupo=%s inventario=%d/%d,%d/%d,%d/%d,%d/%d,%d/%d\n' "$id" "$modo" "$estado" "$(wc -c <"$out")" "$(wc -c <"$err")" "$grupo_cero_aislado" "$fdi" "$fdf" "$hi" "$hf" "$zi" "$zf" "$gi" "$gf" "$ti" "$tf" >&2
    return 1
  fi
}

ejecutar_bf() {
  local id=$1 modo=$2 variable=$3 valor=$4 prueba=$5 oraculo=$6 bin=$7 out="$staging/out" err="$staging/err"
  local fdi fdf hi hf zi zf gi gf ti tf inicio fin estado so se resultado=GO selectores_esperados=0
  [[ $id == C17_BF_PARTICION ]] && selectores_esperados=1
  inventario fdi hi zi gi ti; inicio=${EPOCHREALTIME/./}
  ejecutar_aislado "$id" "$modo" "$out" "$err" "$selectores_esperados" \
    env "$variable=$valor" "$bin" "-test.run=^${prueba}$" -test.count=1; estado=$estado_aislado
  fin=${EPOCHREALTIME/./}; inventario fdf hf zf gf tf; so=$(wc -c <"$out"); se=$(wc -c <"$err")
  [[ $estado -eq 65 && $so -eq 0 && $se -eq 0 && $grupo_cero_aislado == si &&
    $tmp_acreditable_aislado == si && $contenedor_retirado_aislado == si &&
    $fd_ambiente_cerrado_aislado == si && $selectores_acreditados_aislado == si &&
    $selectores_obtenidos_aislado -eq $selectores_esperados && $fdi -eq $fdf && $hi -eq $hf &&
    $zi -eq $zf && $gi -eq $gf && $ti -eq $tf ]] || resultado=NO-GO
  escribir_fila "$evidencia/bf_directos.tsv" "$id" "$modo" \
    "env $variable=$valor testbin -test.run=^${prueba}$ -test.count=1" "$sha_target" "$estado" "$so" "$se" \
    si si si "$(((fin-inicio)/1000))" "$fdi" "$fdf" "$hi" "$hf" "$zi" "$zf" "$gi" "$gf" "$ti" "$tf" \
    "$tmp_inicio_aislado" "$residuos_pre_limpieza_aislado" "$tmp_fin_aislado" "$tmp_acreditable_aislado" \
    "$contenedor_retirado_aislado" "$fd_ambiente_cerrado_aislado" "$selectores_esperados" \
    "$selectores_obtenidos_aislado" "$selectores_acreditados_aislado" "$grupo_cero_aislado" "$oraculo" "$resultado"
  if [[ $resultado != GO ]]; then
    "$publicador_fallo" publicar "$evidencia" "$destino_evidencia" "$out" "$err" "$id" "$modo" "$estado" "$grupo_cero_aislado" \
      "$fdi,$hi,$zi,$gi,$ti" "$fdf,$hf,$zf,$gf,$tf" "$head_inicial" "$($go_bin version)"
    printf 'NO-GO BF=%s modo=%s estado=%d stdout=%d stderr=%d grupo=%s\n' "$id" "$modo" "$estado" "$so" "$se" "$grupo_cero_aislado" >&2
    return 1
  fi
}

preparar_paquete_go() {
  local origen=$1 destino=$2 padre nombre propietario modo
  padre=${destino%/*}
  nombre=${destino##*/}
  [[ $destino == /* && $nombre != "$destino" && -d $padre && ! -L $padre &&
    $(stat -c '%u:%a' -- "$padre") == "$EUID:700" &&
    -d $origen && ! -L $origen && ! -e $destino && ! -L $destino ]] || return 2
  huella_padre_publicacion=$(stat -c '%d:%i' -- "$padre") || return 2
  destino_publicacion_go=$destino

  temporal_publicacion_go=$(mktemp -d "$padre/.${nombre}.go.XXXXXX")
  chmod 0700 "$temporal_publicacion_go"
  cp -- "$origen"/bf_directos.tsv "$origen"/binarios.tsv "$origen"/casos.tsv \
    "$origen"/contexto.tsv "$origen"/fuentes.tsv "$origen"/residuos.txt \
    "$origen"/resumen.txt "$origen"/tmpdir_selectores.tsv "$origen"/utilidades.tsv \
    "$temporal_publicacion_go/"
  huella_origen=$(stat -c '%d:%i' -- "$temporal_publicacion_go") || return 2
  propietario=$(stat -c '%u' -- "$temporal_publicacion_go") || return 2
  modo=$(stat -c '%a' -- "$temporal_publicacion_go") || return 2
  [[ $propietario == "$EUID" && $modo == 700 ]] || return 2
  [[ $(stat -c '%d:%i:%u:%a' -- "$padre") == "$huella_padre_publicacion:$EUID:700" ]] || return 2
}

preparar_helper_noreplace() {
  local helper_src helper_cache propietario modo huella_origen
  [[ -d $temporal_publicacion_go && ! -L $temporal_publicacion_go ]] || return 2
  huella_origen=$(stat -c '%d:%i' -- "$temporal_publicacion_go") || return 2
  propietario=$(stat -c '%u' -- "$temporal_publicacion_go") || return 2
  modo=$(stat -c '%a' -- "$temporal_publicacion_go") || return 2
  [[ $propietario == "$EUID" && $modo == 700 ]] || return 2
  helper_src="$staging/rename_noreplace.go"
  helper_cache="$staging/rename-noreplace-cache"
  helper_bin_noreplace="$temporal_publicacion_go/rename_noreplace"
  mkdir -m 700 "$helper_cache"
  cat > "$helper_src" <<'EOF'
package main

import (
	"os"
	"syscall"
	"unsafe"
)

func main() {
	if len(os.Args) != 3 {
		os.Exit(2)
	}
	oldPath, err := syscall.BytePtrFromString(os.Args[1])
	if err != nil {
		os.Exit(2)
	}
	newPath, err := syscall.BytePtrFromString(os.Args[2])
	if err != nil {
		os.Exit(2)
	}
	const (
		sysRenameat2    = 316
		renameNoreplace = 1
	)
	atFdcwd := uintptr(^uint(99))
	_, _, errno := syscall.Syscall6(sysRenameat2,
		atFdcwd, uintptr(unsafe.Pointer(oldPath)),
		atFdcwd, uintptr(unsafe.Pointer(newPath)),
		renameNoreplace, 0)
	if errno != 0 {
		os.Exit(int(errno))
	}
}
EOF
  (
    cd "$staging"
    GOENV=off GOTOOLCHAIN=local GOROOT="$goroot" PATH="$goroot/bin:/usr/bin:/bin" \
      HOME="$staging" GOCACHE="$helper_cache" \
      "$go_bin" build -trimpath -o "$helper_bin_noreplace" "$helper_src"
  ) || return 2
  chmod 0700 "$helper_bin_noreplace"
  sha_helper_noreplace_global=$(sha256sum "$helper_bin_noreplace" | cut -d' ' -f1)
  rm -f -- "$helper_src"
  rm -rf -- "$helper_cache"
}

sellar_paquete_go() {
  local huella_origen propietario modo
  huella_origen=$(stat -c '%d:%i' -- "$temporal_publicacion_go") || return 2
  propietario=$(stat -c '%u' -- "$temporal_publicacion_go") || return 2
  modo=$(stat -c '%a' -- "$temporal_publicacion_go") || return 2
  [[ $propietario == "$EUID" && $modo == 700 ]] || return 2
  printf 'metodo\thuella_origen\teuid\tmodo\tsha_helper_noreplace\tpostcondiciones\n' > "$temporal_publicacion_go/publicacion.tsv"
  printf 'renameat2_RENAME_NOREPLACE\t%s\t%s\t%s\t%s\torigen_ausente+destino_real+identidad_conservada\n' \
    "$huella_origen" "$EUID" "$modo" "$sha_helper_noreplace_global" >> "$temporal_publicacion_go/publicacion.tsv"
  (
    cd "$temporal_publicacion_go"
    sha256sum bf_directos.tsv binarios.tsv casos.tsv contexto.tsv fuentes.tsv publicacion.tsv \
      residuos.txt resumen.txt rename_noreplace tmpdir_selectores.tsv utilidades.tsv | sort -k2 > SHA256SUMS
    sha256sum -c SHA256SUMS >/dev/null
  )
}

publicar_go_sin_reemplazo() {
  local destino=$1 padre nombre huella_origen huella_destino helper_bin
  padre=${destino%/*}
  nombre=${destino##*/}
  [[ $destino == "$destino_publicacion_go" && $destino == /* && $nombre != "$destino" &&
    -d $padre && ! -L $padre && ! -e $destino && ! -L $destino &&
    -d $temporal_publicacion_go && ! -L $temporal_publicacion_go ]] || return 2
  huella_origen=$(stat -c '%d:%i' -- "$temporal_publicacion_go") || return 2
  helper_bin="$temporal_publicacion_go/rename_noreplace"
  [[ -f $helper_bin && ! -L $helper_bin && $(stat -c '%F:%u:%a' -- "$helper_bin") == "regular file:$EUID:700" &&
    $(sha256sum "$helper_bin" | cut -d' ' -f1) == "$sha_helper_noreplace_global" ]] || return 2
  [[ -d $padre && ! -L $padre && $(stat -c '%d:%i:%u:%a' -- "$padre") == "$huella_padre_publicacion:$EUID:700" ]] || return 2
  "$helper_bin" "$temporal_publicacion_go" "$destino" || return 2
  [[ ! -e $temporal_publicacion_go && ! -L $temporal_publicacion_go && -d $destino && ! -L $destino ]] || return 2
  huella_destino=$(stat -c '%d:%i' -- "$destino") || return 2
  [[ $huella_destino == "$huella_origen" && $(stat -c '%u:%a' -- "$destino") == "$EUID:700" &&
    $(stat -c '%d:%i:%u:%a' -- "$padre") == "$huella_padre_publicacion:$EUID:700" ]] || return 2
  (cd "$destino" && sha256sum -c SHA256SUMS >/dev/null)
  temporal_publicacion_go=
}

for modo in normal race; do
  bin="$staging/o3c-$modo"
  while IFS=$'\t' read -r id prueba oraculo; do
    [[ $id == id ]] && continue
    ejecutar "$id" "$modo" "$prueba" "$oraculo" "$bin"
  done < "$casos"
  ejecutar_bf C01_BF_AUTO "$modo" O3C_P1_FATAL auto TestAutoridadO3cRechazosFatales 'CF entrada: 65 EOF no retorno stdout stderr cero' "$bin"
  ejecutar_bf C08_BF_LEASE "$modo" O3C_P2_FATAL 1 TestRevalidacionO3cLeaseInacreditableEsFatal 'CF lease: 65 EOF no retorno stdout stderr cero' "$bin"
  ejecutar_bf C17_BF_PARTICION "$modo" O3C_P5_BF_DIRECTO particion TestHandoffO3cP5CasosAislados 'CF particion owners: 65 EOF no retorno stdout stderr cero' "$bin"
  for numero in $(seq -w 1 100); do ejecutar "CAP_${modo^^}_$numero" "$modo" TestHandoffO3cP5CasosAislados 'captura completa con inventarios y residuos delta cero' "$bin"; done
done

filas=$(( $(wc -l < "$evidencia/casos.tsv") - 1 )); [[ $filas -eq 244 ]] || { printf 'NO-GO filas %d\n' "$filas" >&2; exit 1; }
filas_bf=$(( $(wc -l < "$evidencia/bf_directos.tsv") - 1 )); [[ $filas_bf -eq 6 ]] || { printf 'NO-GO BF %d\n' "$filas_bf" >&2; exit 1; }
filas_selectores=$(( $(wc -l < "$evidencia/tmpdir_selectores.tsv") - 1 ));
[[ $filas_selectores -eq 1472 ]] || { printf 'NO-GO selectores %d\n' "$filas_selectores" >&2; exit 1; }
! grep -q $'\tNO-GO$' "$evidencia/casos.tsv" "$evidencia/bf_directos.tsv" || { printf 'NO-GO resultados\n' >&2; exit 1; }
head_publicacion=$(git -C "$target" rev-parse HEAD)
tree_publicacion=$(git -C "$target" rev-parse 'HEAD^{tree}')
status_vacio=no
unstaged_vacio=no
staged_vacio=no
[[ -z $(git -C "$target" status --porcelain=v1 --untracked-files=all) ]] && status_vacio=si
git -C "$target" diff --quiet && unstaged_vacio=si
git -C "$target" diff --cached --quiet && staged_vacio=si
[[ $head_publicacion == "$head_inicial" && $tree_publicacion == "$tree_inicial" &&
  $status_vacio == si && $unstaged_vacio == si && $staged_vacio == si ]] || {
  printf 'NO-GO target mutado antes de publicación\n' >&2
  exit 1
}
registrar_contexto "$head_publicacion" "$tree_publicacion" "$status_vacio" "$unstaged_vacio" "$staged_vacio"
: > "$evidencia/residuos.txt"
cat > "$evidencia/resumen.txt" <<EOF
resultado=GO
base=$base
go_version=$($go_bin version)
sha_conductor=$(sha256sum "${BASH_SOURCE[0]}" | cut -d' ' -f1)
sha_publicador_fallo=$(sha256sum "$publicador_fallo" | cut -d' ' -f1)
sha_matriz=$sha_matriz_privada
sha_fuentes=$sha_fuentes_privadas
sha_target=$sha_target
fuentes=32
oraculos=22_por_modo
capturas_normal=100
capturas_race=100
casos_totales=$filas
bf_directos=$filas_bf
bf_estado_eof_no_retorno_0_0=si
snapshot_fuentes_regulares_sin_symlink=si
target_head_tree_limpieza_revalidados=si
tmpdir_exacto_inicio_preconteo_fin_cero=si
tmpdir_selectores=1472
contenedor_runtime_retirado=si
fd_ambiente_cerrado=si
publicacion_go_no_replace=si
residuos=cero
EOF
preparar_paquete_go "$evidencia" "$destino_evidencia" || {
  printf 'NO-GO preparacion paquete GO no acreditada\n' >&2
  exit 1
}
preparar_helper_noreplace || {
  printf 'NO-GO helper no acreditado\n' >&2
  exit 1
}
staging_preparado=$staging
cd "$raiz"
rm -rf -- "$staging_preparado"
[[ ! -e $staging_preparado && ! -L $staging_preparado ]] || {
  printf 'NO-GO staging no retirado\n' >&2
  exit 1
}
staging=
head_post_staging=$(git -C "$target" rev-parse HEAD)
tree_post_staging=$(git -C "$target" rev-parse 'HEAD^{tree}')
status_post_staging=no
unstaged_post_staging=no
staged_post_staging=no
[[ -z $(git -C "$target" status --porcelain=v1 --untracked-files=all) ]] && status_post_staging=si
git -C "$target" diff --quiet && unstaged_post_staging=si
git -C "$target" diff --cached --quiet && staged_post_staging=si
[[ $head_post_staging == "$head_publicacion" && $tree_post_staging == "$tree_publicacion" &&
  $status_post_staging == "$status_vacio" && $unstaged_post_staging == "$unstaged_vacio" &&
  $staged_post_staging == "$staged_vacio" ]] || {
  printf 'NO-GO target mutado tras retirada staging\n' >&2
  exit 1
}
registrar_contexto "$head_publicacion" "$tree_publicacion" "$status_vacio" "$unstaged_vacio" "$staged_vacio" \
  "$temporal_publicacion_go/contexto.tsv"
printf 'staging_retirado=si\n' >> "$temporal_publicacion_go/resumen.txt"
sellar_paquete_go || {
  printf 'NO-GO sello de publicacion no acreditado\n' >&2
  exit 1
}
publicar_go_sin_reemplazo "$destino_evidencia" || {
  printf 'NO-GO publicacion GO no acreditada\n' >&2
  exit 1
}
printf 'GO\n'
