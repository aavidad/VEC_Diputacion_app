#!/usr/bin/bash -p
set -euo pipefail
umask 077
[[ $- == *p* ]] || { printf 'NO-GO intérprete no aislado\n' >&2; exit 2; }
[[ -z ${BASH_ENV+x} && -z ${ENV+x} && -z ${CDPATH+x} ]] || { printf 'NO-GO entorno heredado\n' >&2; exit 2; }
[[ -z $(compgen -A function) && -z $(compgen -A alias) ]] || { printf 'NO-GO funciones/alias heredados\n' >&2; exit 2; }
for variable in ${!GIT_@}; do unset "$variable"; done
hash -r; if ! builtin cd / || [[ $PWD != / ]]; then printf 'NO-GO cwd inicial\n' >&2; exit 2; fi
[[ $# -eq 2 ]] || { printf 'NO-GO uso: conductor.sh TARGET EVIDENCIA_NUEVA\n' >&2; exit 2; }
PATH=/usr/bin:/bin LC_ALL=C; export PATH LC_ALL; readonly pid_conductor=$$
cerrar_fd_excepto() {
  local permitidos=,$1, descriptor numero
  for descriptor in /proc/self/fd/*; do [[ -L $descriptor ]] || continue; numero=${descriptor##*/}; [[ $numero =~ ^[0-9]+$ && $numero -ge 3 && $permitidos != *",$numero,"* ]] || continue; eval "exec ${numero}>&-" || return 2; done
}
hijo_fd() ( cerrar_fd_excepto "$1" || exit 2; shift; exec "$@" )
realpath_bootstrap=/usr/bin/realpath stat_bootstrap=/usr/bin/stat sha256sum_bootstrap=/usr/bin/sha256sum mktemp_bootstrap=/usr/bin/mktemp install_bootstrap=/usr/bin/install rm_bootstrap=/usr/bin/rm
acreditar_cadena_bootstrap() {
  local stat_cmd=$1 directorio=$2 limite=${3:-/} huella uid modo
  while :; do
    [[ $directorio == /* && -d $directorio && ! -L $directorio ]] || return 2
    huella=$(hijo_fd '' "$stat_cmd" -c '%u:%a' -- "$directorio") || return 2
    uid=${huella%%:*}; modo=${huella##*:}
    (( uid == 0 && (8#$modo & 022) == 0 )) || return 2
    [[ $directorio == "$limite" ]] && return 0
    [[ $directorio != / ]] || return 2
    directorio=${directorio%/*}
    [[ -n $directorio ]] || directorio=/
  done
}
for bootstrap in realpath stat sha256sum mktemp install rm; do
  bootstrap_alias="/usr/bin/$bootstrap"
  bootstrap_alias_huella=$(hijo_fd '' "$stat_bootstrap" -c '%u:%F' -- "$bootstrap_alias") || exit 2
  [[ ${bootstrap_alias_huella%%:*} == 0 && ( ${bootstrap_alias_huella#*:} == 'regular file' || ${bootstrap_alias_huella#*:} == 'symbolic link' ) ]] || { printf 'NO-GO alias bootstrap %s\n' "$bootstrap" >&2; exit 2; }
  acreditar_cadena_bootstrap "$stat_bootstrap" "${bootstrap_alias%/*}" || { printf 'NO-GO padres bootstrap %s\n' "$bootstrap" >&2; exit 2; }
  bootstrap_canon=$(hijo_fd '' "$realpath_bootstrap" -e "$bootstrap_alias") || exit 2
  [[ $bootstrap_canon == /* && -f $bootstrap_canon && ! -L $bootstrap_canon ]] || exit 2
  bootstrap_canon_huella=$(hijo_fd '' "$stat_bootstrap" -c '%u:%a:%F' -- "$bootstrap_canon") || exit 2
  bootstrap_canon_uid=${bootstrap_canon_huella%%:*}
  bootstrap_canon_resto=${bootstrap_canon_huella#*:}
  bootstrap_canon_modo=${bootstrap_canon_resto%%:*}
  [[ ${bootstrap_canon_huella##*:} == 'regular file' ]] || exit 2
  (( bootstrap_canon_uid == 0 && (8#$bootstrap_canon_modo & 022) == 0 && (8#$bootstrap_canon_modo & 0111) != 0 )) || exit 2
  acreditar_cadena_bootstrap "$stat_bootstrap" "${bootstrap_canon%/*}" || { printf 'NO-GO padres canon bootstrap %s\n' "$bootstrap" >&2; exit 2; }
done
tool_runtime=$(hijo_fd '' "$mktemp_bootstrap" -d /var/tmp/o3c-p6-tools.XXXXXX)
staging='' git_home='' temporal_publicacion_go='' helper_bin_noreplace='' sha_helper_noreplace_global='' sha_sha256sums_global='' huella_padre_publicacion='' destino_publicacion_go='' publicador_fallo_fd='' go_fd=''
# shellcheck disable=SC2329  # invocada indirectamente por trap EXIT.
limpiar_temporales_propios() {
  local estado_original=$1 estado_limpieza=0; trap - EXIT; [[ -z $temporal_publicacion_go ]] || hijo_fd '' "$rm_bootstrap" -rf -- "$temporal_publicacion_go" || estado_limpieza=1; [[ -z $tool_runtime ]] || hijo_fd '' "$rm_bootstrap" -rf -- "$tool_runtime" || estado_limpieza=1; [[ -z $staging ]] || hijo_fd '' "$rm_bootstrap" -rf -- "$staging" || estado_limpieza=1; (( estado_original != 0 || estado_limpieza == 0 )) || estado_original=2; exit "$estado_original"
}
trap 'limpiar_temporales_propios "$?"' EXIT
[[ $(hijo_fd '' "$stat_bootstrap" -c '%F:%u:%a' -- "$tool_runtime") == "directory:$EUID:700" && ! -L $tool_runtime ]] || exit 2
declare -A utilidad_ruta=() utilidad_sha=() utilidad_canon=() utilidad_uid=() utilidad_modo=()
utilidades=(basename bash cat chmod cmp cp cut dirname env find flock git grep install ln mkdir mkfifo mktemp mv realpath rm seq setsid sha256sum sleep sort stat timeout wc)
fd_fuentes_cerrados=0
for utilidad in "${utilidades[@]}"; do
  ruta_utilidad=$(type -P "$utilidad")
  [[ $ruta_utilidad == /* ]] || { printf 'NO-GO alias utilidad %s\n' "$utilidad" >&2; exit 1; }
  alias_huella=$(hijo_fd '' "$stat_bootstrap" -c '%u:%F' -- "$ruta_utilidad") || exit 1
  [[ ${alias_huella%%:*} == 0 && ( ${alias_huella#*:} == 'regular file' || ${alias_huella#*:} == 'symbolic link' ) ]] || exit 1
  acreditar_cadena_bootstrap "$stat_bootstrap" "${ruta_utilidad%/*}" || exit 1
  ruta_canonica=$(hijo_fd '' "$realpath_bootstrap" -e "$ruta_utilidad") || exit 1
  [[ $ruta_canonica == /* && -f $ruta_canonica && ! -L $ruta_canonica && -x $ruta_canonica ]] || { printf 'NO-GO utilidad %s\n' "$utilidad" >&2; exit 1; }
  acreditar_cadena_bootstrap "$stat_bootstrap" "${ruta_canonica%/*}" || { printf 'NO-GO directorio utilidad %s\n' "$utilidad" >&2; exit 1; }
  exec {fd_pin}< "$ruta_canonica"
  huella_fd=$(hijo_fd '' "$stat_bootstrap" -L -c '%d:%i:%u:%a:%F' -- "/proc/$$/fd/$fd_pin") || exit 1
  archivo_huella=$(hijo_fd '' "$stat_bootstrap" -c '%d:%i:%u:%a:%F' -- "$ruta_canonica") || exit 1
  archivo_resto=${archivo_huella#*:*:*:}
  archivo_modo=${archivo_resto%%:*}
  archivo_uid_resto=${archivo_huella#*:*:}
  archivo_uid=${archivo_uid_resto%%:*}
  [[ $huella_fd == "$archivo_huella" && ${archivo_huella##*:} == 'regular file' ]] || exit 1
  (( archivo_uid == 0 && (8#$archivo_modo & 022) == 0 && (8#$archivo_modo & 0111) != 0 )) || { printf 'NO-GO modo utilidad %s\n' "$utilidad" >&2; exit 1; }
  read -r sha_origen _ < <(hijo_fd '' "$sha256sum_bootstrap" "/proc/$$/fd/$fd_pin")
  destino_utilidad="$tool_runtime/$utilidad"
  hijo_fd '' "$install_bootstrap" -m 0500 "/proc/$$/fd/$fd_pin" "$destino_utilidad"
  read -r sha_copia _ < <(hijo_fd '' "$sha256sum_bootstrap" "$destino_utilidad")
  [[ ${destino_utilidad##*/} == "$utilidad" && -f $destino_utilidad && ! -L $destino_utilidad &&
    $(hijo_fd '' "$stat_bootstrap" -c '%F:%u:%a' -- "$destino_utilidad") == "regular file:$EUID:500" &&
    $sha_copia == "$sha_origen" ]] || exit 1
  exec {fd_pin}<&-; [[ ! -e /proc/$$/fd/$fd_pin ]] || exit 1; ((++fd_fuentes_cerrados))
  utilidad_ruta[$utilidad]="$destino_utilidad"
  utilidad_sha[$utilidad]=$sha_copia
  utilidad_canon[$utilidad]=$ruta_canonica
  utilidad_resto=${archivo_huella#*:*:}
  utilidad_uid[$utilidad]=${utilidad_resto%%:*}
  utilidad_resto=${utilidad_resto#*:}
  utilidad_modo[$utilidad]=${utilidad_resto%%:*}
  hash -p "${utilidad_ruta[$utilidad]}" "$utilidad"
done
[[ $fd_fuentes_cerrados -eq 29 ]] || exit 1
for utilidad in "${utilidades[@]}"; do [[ $(hash -t "$utilidad") == "${utilidad_ruta[$utilidad]}" ]] || exit 1; done
cardinalidad_runtime=$(hijo_fd '' "${utilidad_ruta[find]}" "$tool_runtime" -mindepth 1 -maxdepth 1 -type f -printf '.\n' | hijo_fd '' "${utilidad_ruta[wc]}" -l)
[[ $cardinalidad_runtime -eq 29 ]] || { printf 'NO-GO cardinalidad runtime %s\n' "$cardinalidad_runtime" >&2; exit 1; }
utilidad_privada() ( local nombre=$1; shift; cerrar_fd_excepto ''; exec "${utilidad_ruta[$nombre]}" "$@" )
for utilidad in "${utilidades[@]}"; do eval "$utilidad() { utilidad_privada '$utilidad' \"\$@\"; }"; done
hijo_fd '' "${utilidad_ruta[mkdir]}" -m 0700 "$tool_runtime/home-git" "$tool_runtime/home-git/xdg"; git_home="$tool_runtime/home-git"
git_privado() {
  hijo_fd '' "${utilidad_ruta[env]}" -i PATH="$tool_runtime" HOME="$git_home" XDG_CONFIG_HOME="$git_home/xdg" \
    GIT_CONFIG_NOSYSTEM=1 GIT_ATTR_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/dev/null GIT_TERMINAL_PROMPT=0 GIT_OPTIONAL_LOCKS=0 GIT_NO_REPLACE_OBJECTS=1 GIT_NO_LAZY_FETCH=1 \
    "${utilidad_ruta[git]}" -c core.fileMode=true -c core.trustctime=true -c core.checkStat=default -c core.ignoreStat=false -c core.untrackedCache=false -c core.hooksPath=/dev/null -c core.fsmonitor=false -c diff.external= "$@"
}
script_canonico=$(hijo_fd '' "${utilidad_ruta[realpath]}" -e "${BASH_SOURCE[0]}") || exit 2; [[ -f $script_canonico && ! -L $script_canonico && $script_canonico == */tools/o3c_p6_conductor/conductor.sh ]] || exit 2
unidad=${script_canonico%/*}; raiz=${unidad%/tools/o3c_p6_conductor}; [[ $raiz != "$unidad" && -d $raiz && ! -L $raiz ]] || exit 2
fuentes="$unidad/fuentes.tsv" casos="$unidad/casos.tsv" publicador_fallo_ruta="$unidad/fallo_durable.sh" base=14c1f31079e466a82b8e1d390168078973cc6e05
target=$(hijo_fd '' "${utilidad_ruta[realpath]}" -e "$1") || exit 2
evidencia=$(hijo_fd '' "${utilidad_ruta[realpath]}" -m "$2") || exit 2
[[ $evidencia != "$target" && $evidencia != "$target/"* ]] || { printf 'NO-GO evidencia dentro de target\n' >&2; exit 2; }
destino_padre=${evidencia%/*}
[[ $target == /* && -d $target && ! -L $target && $destino_padre == /* && -d $destino_padre && ! -L $destino_padre && ! -e $evidencia && ! -L $evidencia ]] || { printf 'NO-GO target/evidencia\n' >&2; exit 2; }
huella_destino_padre=$(hijo_fd '' "${utilidad_ruta[stat]}" -c '%d:%i:%u:%a' -- "$destino_padre") || exit 2
[[ $huella_destino_padre == *:"$EUID":700 ]] || { printf 'NO-GO padre destino\n' >&2; exit 2; }
exec {destino_padre_fd}<"$destino_padre"
[[ $(hijo_fd '' "${utilidad_ruta[stat]}" -L -c '%d:%i:%u:%a' -- "/proc/$$/fd/$destino_padre_fd") == "$huella_destino_padre" ]] || exit 2
hijo_fd "$destino_padre_fd" "${utilidad_ruta[flock]}" -n "$destino_padre_fd" || { printf 'NO-GO conductor concurrente\n' >&2; exit 2; }
target_huella=$(hijo_fd '' "${utilidad_ruta[stat]}" -c '%u:%a' -- "$target") || { printf 'NO-GO checkout Git no acreditable\n' >&2; exit 2; }
[[ $target_huella == "$EUID:700" ]] || { printf 'NO-GO checkout Git no acreditable: uid/mode=%s\n' "$target_huella" >&2; exit 2; }
git_dir="$target/.git"
[[ -d $git_dir && ! -L $git_dir && $(hijo_fd '' "${utilidad_ruta[realpath]}" -e "$git_dir") == "$git_dir" ]] || { printf 'NO-GO directorio Git no acreditable\n' >&2; exit 2; }
git_dir_huella=$(hijo_fd '' "${utilidad_ruta[stat]}" -c '%u:%a:%F' -- "$git_dir") || exit 2; git_dir_uid=${git_dir_huella%%:*}; git_dir_resto=${git_dir_huella#*:}; git_dir_modo=${git_dir_resto%%:*}
if ! [[ ${git_dir_huella##*:} == directory && $git_dir_uid == "$EUID" ]] || ! (( (8#$git_dir_modo & 022) == 0 )); then printf 'NO-GO uid/modo directorio Git\n' >&2; exit 2; fi
if ! git_top=$(git_privado -C "$target" rev-parse --show-toplevel 2>&1); then printf 'NO-GO toplevel Git: %s\n' "$git_top" >&2; exit 2; fi
if ! git_absoluto=$(git_privado -C "$target" rev-parse --absolute-git-dir 2>&1); then printf 'NO-GO git-dir absoluto: %s\n' "$git_absoluto" >&2; exit 2; fi
if ! git_comun=$(git_privado -C "$target" rev-parse --path-format=absolute --git-common-dir 2>&1); then printf 'NO-GO common-dir Git: %s\n' "$git_comun" >&2; exit 2; fi
[[ $git_top == "$target" && $git_absoluto == "$git_dir" && $git_comun == "$git_dir" && ! -e $git_dir/commondir && ! -L $git_dir/commondir ]] || { printf 'NO-GO identidad/common-dir Git\n' >&2; exit 2; }
git_config_local="$tool_runtime/git-config-local.keys"
git_privado -C "$target" config --local --no-includes --name-only --null --list > "$git_config_local" || { printf 'NO-GO lectura config local\n' >&2; exit 2; }
while IFS= read -r -d '' git_clave; do git_clave=${git_clave,,}; case "$git_clave" in include.*|includeif.*|filter.*|core.worktree|core.attributesfile|core.trustctime|core.checkstat|core.ignorestat|core.untrackedcache|extensions.worktreeconfig|extensions.partialclone|remote.*.promisor|remote.*.partialclonefilter) printf 'NO-GO config local prohibida: %s\n' "$git_clave" >&2; exit 2;; esac; done < "$git_config_local"
git_info="$git_dir/info"; [[ -d $git_info && ! -L $git_info && $(hijo_fd '' "${utilidad_ruta[realpath]}" -e "$git_info") == "$git_info" ]] || { printf 'NO-GO info Git no acreditable\n' >&2; exit 2; }
for git_info_nombre in attributes grafts; do git_info_archivo="$git_info/$git_info_nombre"; if [[ -e $git_info_archivo || -L $git_info_archivo ]]; then [[ -f $git_info_archivo && ! -L $git_info_archivo && ! -s $git_info_archivo ]] || { printf 'NO-GO info/%s no vacío\n' "$git_info_nombre" >&2; exit 2; }; fi; done
[[ ! -e $git_dir/shallow && ! -L $git_dir/shallow ]] || { printf 'NO-GO repositorio shallow\n' >&2; exit 2; }
git_indice_base=''
auditar_indice_git() {
  local bruto="$tool_runtime/git-index-$1-tagged.z" lista="$tool_runtime/git-index-$1.z" registro ruta total=0
  git_privado -C "$target" ls-files -t -v -z > "$bruto" || return 2; : > "$lista" || return 2
  while IFS= read -r -d '' registro; do [[ ${registro:0:2} == 'H ' && ${#registro} -gt 2 ]] || return 2; ruta=${registro:2}; printf '%s\0' "$ruta" >> "$lista" || return 2; ((++total)); done < "$bruto"
  (( total > 0 )) || return 2; if [[ -z $git_indice_base ]]; then git_indice_base=$lista; else cmp -- "$git_indice_base" "$lista" || return 2; fi
}
auditar_indice_git inicial || { printf 'NO-GO índice Git inicial\n' >&2; exit 2; }; git_trackeados=$git_indice_base git_atributos="$tool_runtime/git-attributes.z"
git_privado -C "$target" check-attr -z --stdin filter < "$git_trackeados" > "$git_atributos" || { printf 'NO-GO check-attr\n' >&2; exit 2; }
exec {git_atributos_fd}< "$git_atributos"; git_trackeados_total=0
while IFS= read -r -d '' git_ruta; do if ! IFS= read -r -d '' git_ruta_attr <&"$git_atributos_fd" || ! IFS= read -r -d '' git_nombre_attr <&"$git_atributos_fd" || ! IFS= read -r -d '' git_valor_attr <&"$git_atributos_fd"; then printf 'NO-GO salida check-attr truncada\n' >&2; exit 2; fi; [[ $git_ruta_attr == "$git_ruta" && $git_nombre_attr == filter && $git_valor_attr == unspecified ]] || { printf 'NO-GO filter efectivo en tracked\n' >&2; exit 2; }; ((++git_trackeados_total)); done < "$git_trackeados"
git_attr_extra=''; if IFS= read -r -d '' git_attr_extra <&"$git_atributos_fd" || [[ -n $git_attr_extra ]]; then printf 'NO-GO salida check-attr excedente\n' >&2; exit 2; fi; exec {git_atributos_fd}<&-
git_privado -C "$target" cat-file -e "$base^{commit}" 2>/dev/null || { printf 'NO-GO base ausente\n' >&2; exit 2; }
git_privado -C "$target" merge-base --is-ancestor "$base" HEAD || { printf 'NO-GO ascendencia\n' >&2; exit 2; }
head_inicial=$(git_privado -C "$target" rev-parse HEAD)
tree_inicial=$(git_privado -C "$target" rev-parse 'HEAD^{tree}')
if ! git_status_salida=$(git_privado -C "$target" status --porcelain=v1 --untracked-files=all); then printf 'NO-GO status Git inicial\n' >&2; exit 2; fi; if [[ -n $git_status_salida ]] || ! git_privado -C "$target" diff --no-ext-diff --no-textconv --quiet || ! git_privado -C "$target" diff --no-ext-diff --no-textconv --cached --quiet; then printf 'NO-GO checkout tracked sucio\n' >&2; exit 2; fi
staging=$(hijo_fd '' "${utilidad_ruta[mktemp]}" -d /var/tmp/o3c-p6.XXXXXX)
runtime_tmp="$staging/runtime-tmp" build_tmp="$staging/build-tmp" snapshot="$staging/snapshot"
hijo_fd '' "${utilidad_ruta[mkdir]}" "$staging/cache" "$build_tmp" "$runtime_tmp"
hijo_fd '' "${utilidad_ruta[mkdir]}" -m 700 "$snapshot"
destino_evidencia=$evidencia evidencia="$staging/evidencia"; hijo_fd '' "${utilidad_ruta[mkdir]}" -m 700 "$evidencia"
printf 'nombre\truta_canonica\tuid_origen\tmodo_origen\tsha256\tmodo_runtime\n' > "$evidencia/utilidades.tsv"
for utilidad in "${utilidades[@]}"; do
  printf '%s\t%s\t%s\t%s\t%s\t0500\n' "$utilidad" "${utilidad_canon[$utilidad]}" \
    "${utilidad_uid[$utilidad]}" "${utilidad_modo[$utilidad]}" "${utilidad_sha[$utilidad]}" >> "$evidencia/utilidades.tsv"
done
go_ruta=/srv/fabrica/orquesta/home/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.5.linux-amd64/bin/go
go_sha_esperado=8da5fd321795754b994c64e3eb8a5a14ff47bd285559a7e876f3c79abafc67f9 goroot=${go_ruta%/bin/go} gotooldir="${go_ruta%/bin/go}/pkg/tool/linux_amd64"
sha_goroot_esperado=b53ebeab1542ea933c6f995a2bcf862d505cb8343ad2b0d1f7a7de3238157ae6 sha_gotooldir_esperado=1061bd99d16310f8f549e375a5c0cb18a79d66441ca0ed4dee60f70fde633f9b
archivos_goroot_esperados=11536 herramientas_gotooldir_esperadas=8
[[ $(hijo_fd '' "${utilidad_ruta[realpath]}" -e "$go_ruta") == "$go_ruta" && -f $go_ruta && ! -L $go_ruta &&
  $(hijo_fd '' "${utilidad_ruta[realpath]}" -e "$goroot") == "$goroot" && -d $goroot && ! -L $goroot &&
  $(hijo_fd '' "${utilidad_ruta[realpath]}" -e "$gotooldir") == "$gotooldir" && -d $gotooldir && ! -L $gotooldir ]] || { printf 'NO-GO rutas toolchain\n' >&2; exit 2; }
acreditar_toolchain_transitiva() {
  local ruta sha huella cantidad
  [[ -z $(hijo_fd '' "${utilidad_ruta[find]}" "$goroot" \( \( ! -type d -a ! -type f \) -o ! -uid "$EUID" -o -perm /022 \) -print -quit) ]] || return 2
  huella=$(while IFS= read -r ruta; do read -r sha _ < <(hijo_fd '' "${utilidad_ruta[sha256sum]}" -- "$goroot/$ruta"); printf '%s\t%s\n' "$sha" "$ruta"; done < <(hijo_fd '' "${utilidad_ruta[find]}" "$goroot" -type f -printf '%P\n' | hijo_fd '' "${utilidad_ruta[env]}" -i LC_ALL=C "${utilidad_ruta[sort]}") | hijo_fd '' "${utilidad_ruta[sha256sum]}")
  cantidad=$(hijo_fd '' "${utilidad_ruta[find]}" "$goroot" -type f -printf '.\n' | hijo_fd '' "${utilidad_ruta[wc]}" -l)
  [[ ${huella%% *} == "$sha_goroot_esperado" && $cantidad -eq $archivos_goroot_esperados ]] || return 2
  huella=$(while IFS= read -r ruta; do read -r sha _ < <(hijo_fd '' "${utilidad_ruta[sha256sum]}" -- "$gotooldir/$ruta"); printf '%s\t%s\n' "$sha" "$ruta"; done < <(hijo_fd '' "${utilidad_ruta[find]}" "$gotooldir" -maxdepth 1 -type f -printf '%f\n' | hijo_fd '' "${utilidad_ruta[env]}" -i LC_ALL=C "${utilidad_ruta[sort]}") | hijo_fd '' "${utilidad_ruta[sha256sum]}")
  cantidad=$(hijo_fd '' "${utilidad_ruta[find]}" "$gotooldir" -maxdepth 1 -type f -printf '.\n' | hijo_fd '' "${utilidad_ruta[wc]}" -l)
  [[ ${huella%% *} == "$sha_gotooldir_esperado" && $cantidad -eq $herramientas_gotooldir_esperadas ]]
}
acreditar_toolchain_transitiva || { printf 'NO-GO frontera transitiva Go\n' >&2; exit 2; }
exec {go_fd}< "$go_ruta"
go_fd_huella=$(hijo_fd '' "${utilidad_ruta[stat]}" -L -c '%d:%i:%u:%a:%F' -- "/proc/$$/fd/$go_fd") || exit 1
go_ruta_huella=$(hijo_fd '' "${utilidad_ruta[stat]}" -c '%d:%i:%u:%a:%F' -- "$go_ruta") || exit 1
go_fd_resto=${go_fd_huella#*:*:*:}; go_fd_modo=${go_fd_resto%%:*}
go_uid_resto=${go_ruta_huella#*:*:}; go_uid=${go_uid_resto%%:*}
read -r go_sha_real _ < <(hijo_fd '' "${utilidad_ruta[sha256sum]}" "/proc/$$/fd/$go_fd")
[[ $go_fd_huella == "$go_ruta_huella" && ${go_fd_huella##*:} == 'regular file' &&
  $go_uid == "$EUID" && $go_sha_real == "$go_sha_esperado" ]] || exit 1
(( (8#$go_fd_modo & 022) == 0 && (8#$go_fd_modo & 0111) != 0 )) || exit 1
go_bin="/proc/$pid_conductor/fd/$go_fd"
hijo_fd '' "${utilidad_ruta[mkdir]}" -m 0700 "$staging/home-go"
go_version=$(hijo_fd '' "${utilidad_ruta[env]}" -i PATH="$tool_runtime" HOME="$staging/home-go" \
  GOROOT="$goroot" GOENV=off GOTOOLCHAIN=local "$go_bin" version)
mapfile -t go_entorno < <(hijo_fd '' "${utilidad_ruta[env]}" -i PATH="$tool_runtime" HOME="$staging/home-go" \
  GOROOT="$goroot" GOENV=off GOTOOLCHAIN=local "$go_bin" env GOROOT GOTOOLDIR)
[[ $go_version == 'go version go1.26.5 linux/amd64' && ${#go_entorno[@]} -eq 2 &&
  ${go_entorno[0]} == "$goroot" && ${go_entorno[1]} == "$gotooldir" ]] || { printf 'NO-GO toolchain\n' >&2; exit 2; }
printf 'go\t%s\t%s\t%s\t%s\tFD\n' "$go_ruta" "$go_uid" "$go_fd_modo" "$go_sha_real" >> "$evidencia/utilidades.tsv"
cc_ruta=/usr/bin/x86_64-linux-gnu-gcc-15 cxx_ruta=/usr/bin/x86_64-linux-gnu-g++-15 c_compiler_path=/usr/libexec/gcc/x86_64-linux-gnu/15:/usr/bin
c_rutas=("$cc_ruta" "$cxx_ruta" /usr/libexec/gcc/x86_64-linux-gnu/15/cc1 /usr/libexec/gcc/x86_64-linux-gnu/15/collect2 /usr/bin/x86_64-linux-gnu-as /usr/bin/x86_64-linux-gnu-ld.bfd /usr/libexec/gcc/x86_64-linux-gnu/15/lto-wrapper)
c_shas=(b5f1b773a7c733738352000c92a077dc5852a1a2fc6d836b1e411be1e9ec5f88 e6718f7e0c7d057c3ff77b550c603da9bc4030e3ede3c053705acce1293dbe4d 30510b346885152c4fce41d2b501751c95d3690b1c34eaa04ebb970e70eae620 8ab5d946ba2c948e5ea5c91e25a104439302e0599a521785770d10690c565437 4e5fcaa3cdc160173cb5cfe7aa35dc39d7aa1f06ba8d75b8d923bbae5473991b 97f48d93b8b076a92d2809ec29dcb17f0f37c8827358f832255e2ed22fef6075 b40116223236c8ebf893097ccb1082b4b2ea1a783cc23847ae4fd6c85dba3bcc)
for indice in "${!c_rutas[@]}"; do c_ruta=${c_rutas[$indice]}; [[ $(hijo_fd '' "${utilidad_ruta[realpath]}" -e "$c_ruta") == "$c_ruta" && $(hijo_fd '' "${utilidad_ruta[stat]}" -c '%u:%a:%F' "$c_ruta") == '0:755:regular file' && -x $c_ruta ]] || exit 2; acreditar_cadena_bootstrap "${utilidad_ruta[stat]}" "${c_ruta%/*}" || exit 2; read -r c_sha _ < <(hijo_fd '' "${utilidad_ruta[sha256sum]}" "$c_ruta"); [[ $c_sha == "${c_shas[$indice]}" ]] || exit 2; printf 'race_c_%s\t%s\t0\t755\t%s\texterno_fijado\n' "${c_ruta##*/}" "$c_ruta" "$c_sha" >> "$evidencia/utilidades.tsv"; done
[[ $(hijo_fd '' "${utilidad_ruta[env]}" -i PATH=/usr/bin:/bin "$cc_ruta" -dumpfullversion -dumpversion) == 15.2.0 &&
  $(hijo_fd '' "${utilidad_ruta[realpath]}" -e "$(hijo_fd '' "${utilidad_ruta[env]}" -i PATH=/usr/bin:/bin "$cc_ruta" -print-prog-name=cc1)") == "${c_rutas[2]}" &&
  $(hijo_fd '' "${utilidad_ruta[realpath]}" -e "$(hijo_fd '' "${utilidad_ruta[env]}" -i PATH=/usr/bin:/bin "$cc_ruta" -print-prog-name=collect2)") == "${c_rutas[3]}" &&
  $(hijo_fd '' "${utilidad_ruta[realpath]}" -e "$(hijo_fd '' "${utilidad_ruta[env]}" -i PATH=/usr/bin:/bin "$cc_ruta" -print-prog-name=as)") == "${c_rutas[4]}" &&
  $(hijo_fd '' "${utilidad_ruta[realpath]}" -e "$(hijo_fd '' "${utilidad_ruta[env]}" -i PATH=/usr/bin:/bin "$cc_ruta" -print-prog-name=ld)") == "${c_rutas[5]}" ]] || { printf 'NO-GO frontera C race\n' >&2; exit 2; }
[[ -f $publicador_fallo_ruta && ! -L $publicador_fallo_ruta ]] || { printf 'NO-GO publicador_fallo\n' >&2; exit 2; }
acreditar_cadena_bootstrap "${utilidad_ruta[stat]}" "${publicador_fallo_ruta%/*}" "$raiz" || { printf 'NO-GO padres publicador_fallo\n' >&2; exit 2; }
publicador_huella=$(hijo_fd '' "${utilidad_ruta[stat]}" -c '%d:%i:%u:%a:%F' -- "$publicador_fallo_ruta") || exit 2
[[ ${publicador_huella##*:} == 'regular file' ]] || exit 2
publicador_resto=${publicador_huella#*:*:}; publicador_uid=${publicador_resto%%:*}; publicador_resto=${publicador_resto#*:}; publicador_modo=${publicador_resto%%:*}
[[ $publicador_uid == 0 && $publicador_modo == 755 ]] || { printf 'NO-GO uid/modo publicador_fallo\n' >&2; exit 2; }
exec {publicador_fallo_fd}< "$publicador_fallo_ruta"
[[ $(hijo_fd '' "${utilidad_ruta[stat]}" -L -c '%d:%i:%u:%a:%F' -- "/proc/$$/fd/$publicador_fallo_fd") == "$publicador_huella" ]] || exit 2
read -r sha_publicador_fallo _ < <(hijo_fd '' "${utilidad_ruta[sha256sum]}" "/proc/$$/fd/$publicador_fallo_fd")
[[ $sha_publicador_fallo == b8f91102a2e98ce1e2e79ed73bfa9bd48c5d1f8ca002271dc5fc2461512bf174 ]] || { printf 'NO-GO hash publicador_fallo\n' >&2; exit 2; }
publicador_fallo="/proc/$pid_conductor/fd/$publicador_fallo_fd"
printf 'fallo_durable\t%s\t%s\t%s\t%s\tFD\n' "$publicador_fallo_ruta" "$publicador_uid" \
  "$publicador_modo" "$sha_publicador_fallo" >> "$evidencia/utilidades.tsv"
publicar_fallo_durable() {
  hijo_fd '' "${utilidad_ruta[env]}" -i PATH="$tool_runtime" HOME="$git_home" \
    "${utilidad_ruta[bash]}" --noprofile --norc "$publicador_fallo" publicar "$@"
}
fixture_ruta=deploy/postgresql/autorizacion_atestada_v3/probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh fixture_sha=7ad65a66ece586710a4651e579385b7aba2ad5b84ef6baf02ba4c36659cd6487
archivos=()
declare -A rutas_snapshot=()
casos_privados="$staging/casos.tsv"
fuentes_privadas="$staging/fuentes.tsv"
cp -- "$casos" "$casos_privados"
cp -- "$fuentes" "$fuentes_privadas"
chmod 0400 "$casos_privados" "$fuentes_privadas"
[[ -f $casos_privados && ! -L $casos_privados && -f $fuentes_privadas && ! -L $fuentes_privadas ]] || { printf 'NO-GO ledger privado no acreditable\n' >&2; exit 1; }
sha_matriz_privada=$(sha256sum "$casos_privados" | cut -d' ' -f1)
sha_fuentes_privadas=$(sha256sum "$fuentes_privadas" | cut -d' ' -f1)
casos=$casos_privados
fuentes=$fuentes_privadas
copiar_snapshot() {
  local sha=$1 ruta=$2 origen="$target/$2" destino="$snapshot/$2"
  [[ $ruta != /* && $ruta != ../* && $ruta != */../* ]] || { printf 'NO-GO ruta snapshot %s\n' "$ruta" >&2; exit 1; }
  [[ -z ${rutas_snapshot[$ruta]+presente} ]] || { printf 'NO-GO ruta duplicada %s\n' "$ruta" >&2; exit 1; }
  rutas_snapshot[$ruta]=1
  [[ -f $origen && ! -L $origen && $(stat -c '%F' -- "$origen") == 'regular file' ]] || { printf 'NO-GO fuente %s\n' "$ruta" >&2; exit 1; }
  mkdir -p "${destino%/*}"; cp --reflink=never -- "$origen" "$destino"; chmod 0400 "$destino"
  [[ -f $destino && ! -L $destino && $(stat -c '%u:%a:%h:%F' -- "$destino") == "$EUID:400:1:regular file" &&
    $(sha256sum "$destino" | cut -d' ' -f1) == "$sha" ]] || { printf 'NO-GO snapshot %s\n' "$ruta" >&2; exit 1; }
  printf '%s\t%s\n' "$sha" "$ruta"
}
{
  printf 'sha256\truta\n'
  while IFS=$'\t' read -r sha ruta; do
    [[ $sha == sha256 ]] && continue
    copiar_snapshot "$sha" "$ruta"; archivos+=("$ruta")
  done < "$fuentes"
  copiar_snapshot "$fixture_sha" "$fixture_ruta"
} > "$evidencia/fuentes.tsv"
[[ ${#archivos[@]} -eq 32 && ${#rutas_snapshot[@]} -eq 33 ]] || { printf 'NO-GO cardinalidad snapshot\n' >&2; exit 1; }
auditar_indice_git post_snapshot || { printf 'NO-GO índice Git tras snapshot\n' >&2; exit 1; }
head_final=$(git_privado -C "$target" rev-parse HEAD); tree_final=$(git_privado -C "$target" rev-parse 'HEAD^{tree}')
if ! git_status_salida=$(git_privado -C "$target" status --porcelain=v1 --untracked-files=all); then printf 'NO-GO status Git tras snapshot\n' >&2; exit 1; fi; [[ $head_final == "$head_inicial" && $tree_final == "$tree_inicial" && -z $git_status_salida ]] || { printf 'NO-GO target mutado durante snapshot\n' >&2; exit 1; }
if ! git_privado -C "$target" diff --no-ext-diff --no-textconv --quiet || ! git_privado -C "$target" diff --no-ext-diff --no-textconv --cached --quiet; then printf 'NO-GO target mutado durante snapshot\n' >&2; exit 1; fi
cd "$snapshot"
entorno=("${utilidad_ruta[env]}" -i PATH="$tool_runtime" HOME="$staging/home-go" TMPDIR="$build_tmp" GOTMPDIR="$build_tmp" GOCACHE="$staging/cache" GOROOT="$goroot" GOENV=off GOTOOLCHAIN=local CGO_ENABLED=0)
hijo_fd '' "${entorno[@]}" "$go_bin" test -c -buildvcs=false -o "$staging/o3c-normal" "${archivos[@]}"
entorno_race=("${utilidad_ruta[env]}" -i PATH="$tool_runtime" HOME="$staging/home-go" TMPDIR="$build_tmp" GOTMPDIR="$build_tmp" GOCACHE="$staging/cache" GOROOT="$goroot" GOENV=off GOTOOLCHAIN=local CGO_ENABLED=1 CC="$cc_ruta" CXX="$cxx_ruta" COMPILER_PATH="$c_compiler_path")
hijo_fd '' "${entorno_race[@]}" "$go_bin" test -race -c -buildvcs=false -o "$staging/o3c-race" "${archivos[@]}"
revalidar_snapshot() {
  local fase=$1 sha ruta total=0
  while IFS=$'\t' read -r sha ruta; do
    [[ $sha == sha256 ]] && continue
    [[ -f $snapshot/$ruta && ! -L $snapshot/$ruta && $(stat -c '%u:%a:%h:%F' -- "$snapshot/$ruta") == "$EUID:400:1:regular file" &&
      $(sha256sum "$snapshot/$ruta" | cut -d' ' -f1) == "$sha" ]] || { printf 'NO-GO snapshot %s %s\n' "$fase" "$ruta" >&2; return 2; }
    ((++total)); done < "$evidencia/fuentes.tsv"
  [[ $total -eq 33 ]] || { printf 'NO-GO cardinalidad snapshot %s\n' "$fase" >&2; return 2; }
}
revalidar_snapshot tras_builds || exit 1
sha_target=$(sha256sum "$evidencia/fuentes.tsv" | cut -d' ' -f1)
printf 'modo\tsha_binario\nnormal\t%s\nrace\t%s\n' "$(sha256sum "$staging/o3c-normal" | cut -d' ' -f1)" "$(sha256sum "$staging/o3c-race" | cut -d' ' -f1)" > "$evidencia/binarios.tsv"
sha_conductor_ejecucion=$(sha256sum "$script_canonico" | cut -d' ' -f1)
registrar_contexto() {
  local head_publicacion=$1 tree_publicacion=$2 status_vacio=$3 unstaged_vacio=$4 staged_vacio=$5 salida=${6:-$evidencia/contexto.tsv}
  [[ -f $salida && ! -L $salida ]] || return 2
  printf 'head_inicial\thead_snapshot\thead_publicacion\ttree_inicial\ttree_snapshot\ttree_publicacion\tstatus_vacio\tunstaged_vacio\tstaged_vacio\tgo_version\tgoroot\tgotooldir\teuid\tsha_conductor\tsha_publicador_fallo\tsha_matriz\tsha_fuentes\tsha_target\tutilidades_runtime\tsha_goroot\tarchivos_goroot\tsha_gotooldir\therramientas_gotooldir\tfd_fuentes_utilidades_cerrados\trace_c_tcb\n%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
    "$head_inicial" "$head_final" "$head_publicacion" "$tree_inicial" "$tree_final" "$tree_publicacion" \
    "$status_vacio" "$unstaged_vacio" "$staged_vacio" "$go_version" "$goroot" "$gotooldir" "$EUID" \
    "$sha_conductor_ejecucion" "$sha_publicador_fallo" "$sha_matriz_privada" "$sha_fuentes_privadas" \
    "$sha_target" "$cardinalidad_runtime" "$sha_goroot_esperado" "$archivos_goroot_esperados" "$sha_gotooldir_esperado" \
    "$herramientas_gotooldir_esperadas" "$fd_fuentes_cerrados" ejecutables_7_fijados_sysroot_host_no_atestado > "$salida"
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
  for fd in "/proc/$$/fd"/*; do [[ -L $fd ]] && ((nfd+=1)); done
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
  local id=$1 modo=$2 out=$3 err=$4 selectores_esperados=$5 env_assignment=; shift 5
  if [[ ${1:-} == *=* ]]; then env_assignment=$1; shift; fi
  local lider etiqueta=${id,,} runtime_aislado huella_tmpdir_aislado atestacion_selectores atestacion_selectores_huella atestacion_fd fd_marker fd_marker_fd fd_marker_value fd_marker_huella
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
  mkfifo -m 0600 "$fd_marker"
  exec {fd_marker_bootstrap}<>"$fd_marker"
  exec {fd_marker_fd}<"$fd_marker"
  exec {fd_marker_writer_fd}>"$fd_marker"
  exec {fd_marker_bootstrap}>&-
  fd_marker_huella=$(stat -L -c '%d:%i:%u:%a' -- "/proc/$$/fd/$fd_marker_fd")
  rm -f -- "$fd_marker"
  # El proceso intermedio cierra todo descriptor ambiental >=3 antes de
  # convertirse en el líder de sesión. stdout/stderr se fijan externamente.
  (
    local -a ambiente=(PATH="$tool_runtime" HOME="$runtime_aislado/home" TMPDIR="$runtime_aislado/tmp" GOTMPDIR="$runtime_aislado/tmp" GOROOT="$goroot" GOENV=off GOTOOLCHAIN=local O3C_FD_MARKER_FD="$fd_marker_writer_fd")
    [[ -z $env_assignment ]] || ambiente+=("$env_assignment")
    # SC2016: las expansiones se difieren deliberadamente al wrapper hijo.
    # shellcheck disable=SC2016
    local script='marker_fd=$O3C_FD_MARKER_FD; unset O3C_FD_MARKER_FD; [[ $marker_fd =~ ^[0-9]+$ && $marker_fd -ge 3 ]] || exit 125; for descriptor in /proc/self/fd/*; do numero_fd=${descriptor##*/}; if [[ -L $descriptor && $numero_fd =~ ^[0-9]+$ && $numero_fd -ge 3 && $numero_fd -ne $marker_fd ]]; then eval "exec ${numero_fd}>&-"; fi; done; fd_ambiental_residual=no; for descriptor in /proc/self/fd/*; do numero_fd=${descriptor##*/}; if [[ -L $descriptor && $numero_fd =~ ^[0-9]+$ && $numero_fd -ge 3 && $numero_fd -ne $marker_fd ]]; then fd_ambiental_residual=si; fi; done; if [[ $fd_ambiental_residual == no ]]; then printf "si\\n" >&"$marker_fd" || exit 125; else printf "no\\n" >&"$marker_fd" || exit 125; fi; exec {marker_fd}>&- || exit 125; fd_ambiental_residual=no; for descriptor in /proc/self/fd/*; do numero_fd=${descriptor##*/}; if [[ -L $descriptor && $numero_fd =~ ^[0-9]+$ && $numero_fd -ge 3 ]]; then fd_ambiental_residual=si; fi; done; [[ $fd_ambiental_residual == no ]] || exit 125; exec "$@"'
    [[ $selectores_esperados -gt 0 ]] && ambiente+=("O3C_P5_ATESTACION=$atestacion_selectores")
    cerrar_fd_excepto "$fd_marker_writer_fd" || exit 125
    exec "${utilidad_ruta[setsid]}" "${utilidad_ruta[timeout]}" --signal=KILL 180 "${utilidad_ruta[env]}" -i "${ambiente[@]}" "${utilidad_ruta[bash]}" --noprofile --norc -c "$script" _ "$@"
  ) >"$out" 2>"$err" &
  lider=$!
  exec {fd_marker_writer_fd}>&-
  wait "$lider"
  estado_aislado=$?
  set -e
  fd_marker_extra=
  if [[ $(stat -L -c '%d:%i:%u:%a' -- "/proc/$$/fd/$fd_marker_fd") == "$fd_marker_huella" ]] &&
    IFS= read -r fd_marker_value <&"$fd_marker_fd" && [[ $fd_marker_value == si ]] &&
    ! IFS= read -r fd_marker_extra <&"$fd_marker_fd" && [[ -z ${fd_marker_extra:-} ]]; then
    fd_ambiente_cerrado_aislado=si
  fi
  exec {fd_marker_fd}<&-
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
  local fdi fdf hi hf zi zf gi gf ti tf inicio fin estado so se resultado=GO selectores_esperados=0 salida_exacta=no
  case "$id" in
    C17_OWNERS|C18_O4A_OPACO|C19_RETIRADA|C20_POST_CONT|C22_RESIDUOS|CAP_*) selectores_esperados=7 ;;
  esac
  inventario fdi hi zi gi ti; inicio=${EPOCHREALTIME/./}
  ejecutar_aislado "$id" "$modo" "$out" "$err" "$selectores_esperados" \
    "$bin" "-test.run=^(${patron})$" -test.count=1; estado=$estado_aislado
  fin=${EPOCHREALTIME/./}; inventario fdf hf zf gf tf; so=$(wc -c <"$out"); se=$(wc -c <"$err")
  if printf '%s  %s\n' c26de83abdc9496cd1301470918ec39ecca1cf389ef0ae1c6504da1800d1c431 "$out" | sha256sum --status -c -; then salida_exacta=si; fi
  [[ $estado -eq 0 && $so -eq 5 && $salida_exacta == si && $se -eq 0 && $grupo_cero_aislado == si && $tmp_acreditable_aislado == si &&
    $contenedor_retirado_aislado == si && $fd_ambiente_cerrado_aislado == si &&
    $selectores_acreditados_aislado == si && $selectores_obtenidos_aislado -eq $selectores_esperados &&
    $fdi -eq $fdf && $hi -eq $hf && $zi -eq $zf && $gi -eq $gf && $ti -eq $tf ]] || resultado=NO-GO
  escribir_fila "$evidencia/casos.tsv" "$id" "$modo" "testbin -test.run=^(${patron})$ -test.count=1" \
    "$sha_target" "$estado" "$so" "$se" "$(((fin-inicio)/1000))" \
    "$fdi" "$fdf" "$hi" "$hf" "$zi" "$zf" "$gi" "$gf" "$ti" "$tf" \
    "$tmp_inicio_aislado" "$residuos_pre_limpieza_aislado" "$tmp_fin_aislado" "$tmp_acreditable_aislado" \
    "$contenedor_retirado_aislado" "$fd_ambiente_cerrado_aislado" "$selectores_esperados" \
    "$selectores_obtenidos_aislado" "$selectores_acreditados_aislado" "$grupo_cero_aislado" "$oraculo" "$resultado"
  if [[ $resultado != GO ]]; then
    publicar_fallo_durable "$evidencia" "$destino_evidencia" "$out" "$err" "$id" "$modo" "$estado" "$grupo_cero_aislado" \
      "$fdi,$hi,$zi,$gi,$ti" "$fdf,$hf,$zf,$gf,$tf" "$head_inicial" "$go_version"
    printf 'NO-GO caso=%s modo=%s estado=%d stdout=%d stderr=%d grupo=%s inventario=%d/%d,%d/%d,%d/%d,%d/%d,%d/%d\n' "$id" "$modo" "$estado" "$so" "$se" "$grupo_cero_aislado" "$fdi" "$fdf" "$hi" "$hf" "$zi" "$zf" "$gi" "$gf" "$ti" "$tf" >&2
    return 1
  fi
}
ejecutar_bf() {
  local id=$1 modo=$2 variable=$3 valor=$4 prueba=$5 oraculo=$6 bin=$7 out="$staging/out" err="$staging/err"
  local fdi fdf hi hf zi zf gi gf ti tf inicio fin estado so se resultado=GO selectores_esperados=0
  [[ $id == C17_BF_PARTICION ]] && selectores_esperados=1
  inventario fdi hi zi gi ti; inicio=${EPOCHREALTIME/./}
  ejecutar_aislado "$id" "$modo" "$out" "$err" "$selectores_esperados" "$variable=$valor" \
    "$bin" "-test.run=^${prueba}$" -test.count=1; estado=$estado_aislado
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
    publicar_fallo_durable "$evidencia" "$destino_evidencia" "$out" "$err" "$id" "$modo" "$estado" "$grupo_cero_aislado" \
      "$fdi,$hi,$zi,$gi,$ti" "$fdf,$hf,$zf,$gf,$tf" "$head_inicial" "$go_version"
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
  huella_padre_publicacion=$(stat -c '%d:%i:%u:%a' -- "$padre") || return 2
  destino_publicacion_go=$destino
  temporal_publicacion_go=$(mktemp -d "$padre/.${nombre}.go.XXXXXX")
  chmod 0700 "$temporal_publicacion_go"
  cp -- "$origen"/bf_directos.tsv "$origen"/binarios.tsv "$origen"/casos.tsv \
    "$origen"/contexto.tsv "$origen"/fuentes.tsv "$origen"/residuos.txt \
    "$origen"/resumen.txt "$origen"/tmpdir_selectores.tsv "$origen"/utilidades.tsv \
    "$temporal_publicacion_go/"
  huella_origen=$(stat -c '%d:%i:%u:%a' -- "$temporal_publicacion_go") || return 2
  propietario=$(stat -c '%u' -- "$temporal_publicacion_go") || return 2
  modo=$(stat -c '%a' -- "$temporal_publicacion_go") || return 2
  [[ $propietario == "$EUID" && $modo == 700 ]] || return 2
  [[ $(stat -L -c '%d:%i:%u:%a' -- "/proc/$$/fd/$destino_padre_fd") == "$huella_padre_publicacion" ]] || return 2
}
preparar_helper_noreplace() {
  local helper_src helper_cache propietario modo huella_origen
  [[ -d $temporal_publicacion_go && ! -L $temporal_publicacion_go ]] || return 2
  huella_origen=$(stat -c '%d:%i:%u:%a' -- "$temporal_publicacion_go") || return 2
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
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)
func main() {
	if len(os.Args) != 8 {
		os.Exit(2)
	}
	var fd int
	if _, err := fmt.Sscan(os.Args[1], &fd); err != nil || fd < 0 { os.Exit(2) }
	if os.Args[2] == "" || os.Args[3] == "" || os.Args[2] == "." || os.Args[2] == ".." || os.Args[3] == "." || os.Args[3] == ".." || filepath.Base(os.Args[2]) != os.Args[2] || filepath.Base(os.Args[3]) != os.Args[3] { os.Exit(2) }
	parent := fd
	var st syscall.Stat_t
	if syscall.Fstat(parent, &st) != nil || st.Uid != uint32(os.Geteuid()) || st.Mode&syscall.S_IFMT != syscall.S_IFDIR || st.Mode&0777 != 0700 { os.Exit(2) }
	origin, err := syscall.Openat(parent, os.Args[2], syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if err != nil { os.Exit(2) }
	var ost syscall.Stat_t
	if syscall.Fstat(origin, &ost) != nil || ost.Uid != uint32(os.Geteuid()) || ost.Mode&syscall.S_IFMT != syscall.S_IFDIR || ost.Mode&0777 != 0700 { os.Exit(2) }
	if fmt.Sprintf("%d:%d:%d:%o", st.Dev, st.Ino, st.Uid, st.Mode&0777) != os.Args[4] || fmt.Sprintf("%d:%d:%d:%o", ost.Dev, ost.Ino, ost.Uid, ost.Mode&0777) != os.Args[5] { os.Exit(2) }
	dirDup, err := syscall.Dup(origin)
	if err != nil { os.Exit(2) }
	dirFile := os.NewFile(uintptr(dirDup), "origin-dir")
	names, err := dirFile.Readdirnames(-1)
	if closeErr := dirFile.Close(); err != nil || closeErr != nil { os.Exit(2) }
	allowed := map[string]bool{"bf_directos.tsv": true, "binarios.tsv": true, "casos.tsv": true, "contexto.tsv": true, "fuentes.tsv": true, "publicacion.tsv": true, "residuos.txt": true, "resumen.txt": true, "rename_noreplace": true, "tmpdir_selectores.tsv": true, "utilidades.tsv": true, "SHA256SUMS": true}
	seen := map[string]bool{}
	for _, name := range names { if !allowed[name] || seen[name] { os.Exit(2) }; seen[name] = true }
	if len(seen) != 12 { os.Exit(2) }
	pathParent, err := syscall.Open(os.Args[7], syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if err != nil { os.Exit(2) }
	var pst syscall.Stat_t
	if syscall.Fstat(pathParent, &pst) != nil || fmt.Sprintf("%d:%d:%d:%o", pst.Dev, pst.Ino, pst.Uid, pst.Mode&0777) != os.Args[4] { os.Exit(2) }
	shaFD, err := syscall.Openat(origin, "SHA256SUMS", syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if err != nil { os.Exit(2) }
	shaFile := os.NewFile(uintptr(shaFD), "SHA256SUMS")
	shaBytes, err := io.ReadAll(shaFile)
	if err != nil { os.Exit(2) }
	if _, err = shaFile.Seek(0, 0); err != nil { os.Exit(2) }
	h := sha256.Sum256(shaBytes)
	if hex.EncodeToString(h[:]) != os.Args[6] { os.Exit(2) }
	if shaFile.Close() != nil { os.Exit(2) }
	checks := map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(string(shaBytes)))
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) != 2 || len(parts[0]) != 64 || checks[parts[1]] != "" { os.Exit(2) }
		checks[parts[1]] = parts[0]
	}
	if scanner.Err() != nil || len(checks) != 11 { os.Exit(2) }
	for _, name := range []string{"bf_directos.tsv", "binarios.tsv", "casos.tsv", "contexto.tsv", "fuentes.tsv", "publicacion.tsv", "residuos.txt", "resumen.txt", "rename_noreplace", "tmpdir_selectores.tsv", "utilidades.tsv", "SHA256SUMS"} {
		fd, err := syscall.Openat(origin, name, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
		if err != nil { os.Exit(2) }
		var entry syscall.Stat_t
		ok := syscall.Fstat(fd, &entry) == nil && entry.Uid == uint32(os.Geteuid()) && entry.Mode&syscall.S_IFMT == syscall.S_IFREG
		if ok && name != "SHA256SUMS" { file := os.NewFile(uintptr(fd), name); digest := sha256.New(); _, copyErr := io.Copy(digest, file); closeErr := file.Close(); ok = copyErr == nil && closeErr == nil && hex.EncodeToString(digest.Sum(nil)) == checks[name] }
		if name == "SHA256SUMS" { ok = syscall.Close(fd) == nil && ok }
		if !ok { os.Exit(2) }
	}
	const (
		sysRenameat2    = 316
		renameNoreplace = 1
	)
	oldPath, _ := syscall.BytePtrFromString(os.Args[2])
	newPath, _ := syscall.BytePtrFromString(os.Args[3])
	if syscall.Close(pathParent) != nil || syscall.Close(origin) != nil { os.Exit(2) }
	_, _, errno := syscall.Syscall6(sysRenameat2, uintptr(parent), uintptr(unsafe.Pointer(oldPath)), uintptr(parent), uintptr(unsafe.Pointer(newPath)), renameNoreplace, 0)
	if errno != 0 { os.Exit(int(errno)) }
	os.Exit(0)
}
EOF
  (
    cd "$staging"
    hijo_fd '' "${utilidad_ruta[env]}" -i PATH="$tool_runtime" HOME="$staging/home-go" TMPDIR="$staging" \
      GOTMPDIR="$staging" GOCACHE="$helper_cache" GOROOT="$goroot" \
      GOENV=off GOTOOLCHAIN=local CGO_ENABLED=0 \
      "$go_bin" build -buildvcs=false -trimpath -o "$helper_bin_noreplace" "$helper_src"
  ) || return 2
  acreditar_toolchain_transitiva || return 2
  chmod 0700 "$helper_bin_noreplace"
  sha_helper_noreplace_global=$(sha256sum "$helper_bin_noreplace" | cut -d' ' -f1)
  rm -f -- "$helper_src"
  rm -rf -- "$helper_cache"
}
sellar_paquete_go() {
  local huella_origen propietario modo
  huella_origen=$(stat -c '%d:%i:%u:%a' -- "$temporal_publicacion_go") || return 2
  propietario=$(stat -c '%u' -- "$temporal_publicacion_go") || return 2
  modo=$(stat -c '%a' -- "$temporal_publicacion_go") || return 2
  [[ $propietario == "$EUID" && $modo == 700 ]] || return 2
  printf 'metodo\thuella_origen\tsha_helper_noreplace\tpostcondiciones\n' > "$temporal_publicacion_go/publicacion.tsv"
  printf 'renameat2_RENAME_NOREPLACE\t%s\t%s\torigen_ausente+destino_real+identidad_conservada\n' \
    "$huella_origen" "$sha_helper_noreplace_global" >> "$temporal_publicacion_go/publicacion.tsv"
  (
    cd "$temporal_publicacion_go"
    sha256sum bf_directos.tsv binarios.tsv casos.tsv contexto.tsv fuentes.tsv publicacion.tsv \
      residuos.txt resumen.txt rename_noreplace tmpdir_selectores.tsv utilidades.tsv | sort -k2 > SHA256SUMS
    sha256sum -c SHA256SUMS >/dev/null
  )
  sha_sha256sums_global=$(sha256sum "$temporal_publicacion_go/SHA256SUMS" | cut -d' ' -f1)
}
publicar_go_sin_reemplazo() {
  local destino=$1 padre nombre origen_nombre huella_origen helper_bin helper_fd descriptor numero_fd
  padre=${destino%/*}
  nombre=${destino##*/}
  [[ $destino == "$destino_publicacion_go" && $destino == /* && $nombre != "$destino" &&
    -d $padre && ! -L $padre && ! -e $destino && ! -L $destino &&
    -d $temporal_publicacion_go && ! -L $temporal_publicacion_go ]] || return 2
  huella_origen=$(stat -c '%d:%i:%u:%a' -- "$temporal_publicacion_go") || return 2
  origen_nombre=${temporal_publicacion_go##*/}
  helper_bin="$temporal_publicacion_go/rename_noreplace"
  [[ -f $helper_bin && ! -L $helper_bin && $(stat -c '%F:%u:%a' -- "$helper_bin") == "regular file:$EUID:700" &&
    $(sha256sum "$helper_bin" | cut -d' ' -f1) == "$sha_helper_noreplace_global" ]] || return 2
  [[ -d $padre && ! -L $padre && $(stat -L -c '%d:%i:%u:%a' -- "/proc/$$/fd/$destino_padre_fd") == "$huella_padre_publicacion" ]] || return 2
  [[ $sha_sha256sums_global =~ ^[0-9a-f]{64}$ ]] || return 2
  exec {helper_fd}< "$helper_bin"
  [[ $(stat -L -c '%F:%u:%a' -- "/proc/$$/fd/$helper_fd") == "regular file:$EUID:700" &&
    $(sha256sum "/proc/$$/fd/$helper_fd" | cut -d' ' -f1) == "$sha_helper_noreplace_global" ]] || { exec {helper_fd}<&-; return 2; }
  exec {go_fd}<&-
  go_fd=
  exec {publicador_fallo_fd}<&-
  publicador_fallo_fd=
  hash -r
  hijo_fd '' "$rm_bootstrap" -rf -- "$tool_runtime"
  [[ ! -e $tool_runtime && ! -L $tool_runtime ]] || { exec {helper_fd}<&-; return 2; }
  tool_runtime=
  for descriptor in "/proc/$$/fd"/*; do
    [[ -L $descriptor ]] || continue
    numero_fd=${descriptor##*/}
    case "$numero_fd" in 0|1|2|"$destino_padre_fd"|"$helper_fd") ;; *) eval "exec ${numero_fd}>&-" || return 2 ;; esac
  done
  for descriptor in "/proc/$$/fd"/*; do
    [[ -L $descriptor ]] || continue
    numero_fd=${descriptor##*/}
    case "$numero_fd" in 0|1|2|"$destino_padre_fd"|"$helper_fd") ;; *) return 2 ;; esac
  done
  hijo_fd "$destino_padre_fd" "/proc/$pid_conductor/fd/$helper_fd" "$destino_padre_fd" "$origen_nombre" "$nombre" "$huella_padre_publicacion" "$huella_origen" "$sha_sha256sums_global" "$padre" || { exec {helper_fd}<&-; return 2; }
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
revalidar_snapshot pre_publicacion || { printf 'NO-GO snapshot antes de publicación\n' >&2; exit 1; }
auditar_indice_git pre_publicacion || { printf 'NO-GO índice Git antes de publicación\n' >&2; exit 1; }
head_publicacion=$(git_privado -C "$target" rev-parse HEAD); tree_publicacion=$(git_privado -C "$target" rev-parse 'HEAD^{tree}')
status_vacio=no unstaged_vacio=no staged_vacio=no; if ! git_status_salida=$(git_privado -C "$target" status --porcelain=v1 --untracked-files=all); then printf 'NO-GO status Git antes de publicación\n' >&2; exit 1; fi; [[ -z $git_status_salida ]] && status_vacio=si
git_privado -C "$target" diff --no-ext-diff --no-textconv --quiet && unstaged_vacio=si; git_privado -C "$target" diff --no-ext-diff --no-textconv --cached --quiet && staged_vacio=si
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
go_version=$go_version
sha_conductor=$sha_conductor_ejecucion
sha_publicador_fallo=$sha_publicador_fallo
sha_matriz=$sha_matriz_privada
sha_fuentes=$sha_fuentes_privadas
sha_target=$sha_target
fuentes=32
snapshot_rutas=33
oraculos=22_por_modo
capturas_normal=100
capturas_race=100
casos_totales=$filas
bf_directos=$filas_bf
bf_estado_eof_no_retorno_0_0=si
snapshot_rutas_regulares_euid_0400_nlink1_sin_symlink=si
snapshot_revalidado_post_ejecucion=33
target_head_tree_limpieza_revalidados=si
tmpdir_exacto_inicio_preconteo_fin_cero=si
tmpdir_selectores=1472
contenedor_runtime_retirado=si
fd_ambiente_cerrado=si
utilidades_runtime=29
bootstrap_utilidades=6
fd_fuentes_utilidades_cerrados=$fd_fuentes_cerrados
goroot=$goroot
gotooldir=$gotooldir
sha_goroot=$sha_goroot_esperado
archivos_goroot=$archivos_goroot_esperados
sha_gotooldir=$sha_gotooldir_esperado
herramientas_gotooldir=$herramientas_gotooldir_esperadas
cc=$cc_ruta
cxx=$cxx_ruta
race_c_tcb=ejecutables_7_fijados_sysroot_host_no_atestado
publicacion_go_no_replace=si
residuos=cero
EOF
preparar_paquete_go "$evidencia" "$destino_evidencia" || { printf 'NO-GO preparacion paquete GO no acreditada\n' >&2; exit 1; }
preparar_helper_noreplace || { printf 'NO-GO helper no acreditado\n' >&2; exit 1; }
staging_preparado=$staging
cd "$raiz"
rm -rf -- "$staging_preparado"
[[ ! -e $staging_preparado && ! -L $staging_preparado ]] || {
  printf 'NO-GO staging no retirado\n' >&2
  exit 1
}
staging=
auditar_indice_git post_staging || { printf 'NO-GO índice Git tras staging\n' >&2; exit 1; }
head_post_staging=$(git_privado -C "$target" rev-parse HEAD); tree_post_staging=$(git_privado -C "$target" rev-parse 'HEAD^{tree}')
status_post_staging=no unstaged_post_staging=no staged_post_staging=no; if ! git_status_salida=$(git_privado -C "$target" status --porcelain=v1 --untracked-files=all); then printf 'NO-GO status Git tras staging\n' >&2; exit 1; fi; [[ -z $git_status_salida ]] && status_post_staging=si
git_privado -C "$target" diff --no-ext-diff --no-textconv --quiet && unstaged_post_staging=si; git_privado -C "$target" diff --no-ext-diff --no-textconv --cached --quiet && staged_post_staging=si
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
