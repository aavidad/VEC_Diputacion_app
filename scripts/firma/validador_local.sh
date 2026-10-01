#!/usr/bin/env bash
# Ensayo E3 desechable: todos los secretos y datos se generan fuera de Git.
set -euo pipefail

af_source="${AUTOFIRMAV2_SOURCE:-$HOME/Trabajo/AutofirmaV2-vec-verificacion}"
script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
for command in bwrap rsync go timeout prlimit; do
  command -v "$command" >/dev/null || { echo "Falta $command" >&2; exit 2; }
done
[[ -f "$af_source/go.mod" && -f "$af_source/cmd/autofirma/main.go" ]] || { echo 'Falta la fuente local de AutofirmaV2' >&2; exit 2; }
cache="$(go env GOMODCACHE)"
[[ -d "$cache" ]] || { echo 'Falta la caché local de módulos Go; no se descargarán dependencias' >&2; exit 2; }
toolchain="$(go env GOROOT)"
if [[ "$toolchain" == "$cache/"* ]]; then
  sandbox_toolchain="/modcache/${toolchain#"$cache/"}"
elif [[ "$toolchain" == /usr/* ]]; then
  sandbox_toolchain="$toolchain"
else
  echo 'Toolchain Go fuera de los directorios permitidos' >&2; exit 2
fi

scratch_parent="${VEC_E3_SCRATCH_PARENT:-${XDG_CACHE_HOME:-$HOME/.cache}}"
mkdir -p -- "$scratch_parent"
scratch="$(mktemp -d "$scratch_parent/vec-e3-validador.XXXXXXXX")"
chmod 700 "$scratch"
cleanup() { rm -rf -- "$scratch"; }
trap cleanup EXIT HUP INT TERM

mkdir -p "$scratch/src/cmd/vecfixture" "$scratch/home" "$scratch/tmp" "$scratch/cache" "$scratch/bin"
rsync -a --exclude=.git --exclude=.worktrees --exclude='*.p12' --exclude='*.key' "$af_source/" "$scratch/src/"
cp -- "$script_dir/fixture/main.go.tmpl" "$scratch/src/cmd/vecfixture/main.go"
cp -- "$script_dir/validador_runtime.sh" "$scratch/runtime.sh"

# El proceso ensayado solo ve el árbol temporal, herramientas y caché Go de lectura.
# Namespace de red aislado: únicamente loopback. Entorno reconstruido por bwrap.
prlimit --cpu=170 --as=4294967296 --nproc=4096 --nofile=128 --fsize=67108864 -- \
  timeout --kill-after=5s 180s bwrap --unshare-all --new-session --cap-drop ALL --clearenv \
    --ro-bind /usr /usr --ro-bind /bin /bin --ro-bind /lib /lib --ro-bind /lib64 /lib64 \
    --ro-bind "$cache" /modcache --bind "$scratch" /work \
    --proc /proc --dev /dev --tmpfs /tmp --dir /home --chdir /work/src \
    --setenv PATH "$sandbox_toolchain/bin:/usr/bin:/bin" --setenv GOROOT "$sandbox_toolchain" --setenv HOME /work/home \
    --setenv TMPDIR /work/tmp --setenv GOCACHE /work/cache --setenv GOMODCACHE /modcache \
    --setenv GOPROXY off --setenv GOSUMDB off --setenv GOTOOLCHAIN local --setenv GOMAXPROCS 2 \
    -- bash /work/runtime.sh
