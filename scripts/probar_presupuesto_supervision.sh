#!/usr/bin/env bash
# M0 — Encaje y presupuesto de recursos de la capa de supervisión (Zabbix 7.0 LTS).
#
# Modos:
#   recursos     Recursos disponibles del equipo (CPU, memoria, disco, carga).
#   coherencia   Las unidades Quadlet usan las imágenes de imagenes.lock y los
#                límites/ajustes de recursos.env.
#   rootless     Comprueba que los límites cgroup se aplican de verdad a un
#                usuario sin privilegios (delegación de systemd y, si hay
#                podman rootless, contenedores reales).
#   piloto       Piloto aislado: PostgreSQL propio + servidor + frontend +
#                agente 2; mide reposo y carga; verifica límites y visibilidad
#                del agente SIN socket del motor; limpia todo al terminar.
#
# Variables: MOTOR=podman|docker (por defecto podman si existe), REPOSO_MIN=5,
# CARGA_MIN=15, MUESTREO_S=15, ESTABILIZAR_MIN=3, PUERTO_WEB=18080, PUERTO_SERVIDOR=20051,
# DIR_TRABAJO (por defecto ~/.local/state/vec-supervision-m0-piloto),
# SUBRED_DATOS y SUBRED_FRONTAL (opcionales, p. ej. 10.253.10.0/24).
#
# El propio script corre con nice 19 e ionice «idle»; el piloto tiene topes de
# CPU/memoria/pids por contenedor. Sin datos personales: host sintético
# «vec-piloto-m0», contenedores por identificador o nombre de unidad.
set -Eeuo pipefail

if [[ -z "${VECSUP_PRIORIDAD_BAJA:-}" ]]; then
  export VECSUP_PRIORIDAD_BAJA=1
  exec nice -n 19 ionice -c 3 "$0" "$@"
fi

RAIZ="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DZ="$RAIZ/deploy/observabilidad/zabbix"
# shellcheck source=/dev/null
source "$DZ/recursos.env"
# shellcheck source=/dev/null
source "$DZ/imagenes.lock"

MODO="${1:-ayuda}"
REPOSO_MIN="${REPOSO_MIN:-5}"
CARGA_MIN="${CARGA_MIN:-15}"
MUESTREO_S="${MUESTREO_S:-15}"
PUERTO_WEB="${PUERTO_WEB:-18080}"
PUERTO_SERVIDOR="${PUERTO_SERVIDOR:-20051}"
DIR_TRABAJO="${DIR_TRABAJO:-$HOME/.local/state/vec-supervision-m0-piloto}"
P=vecsup-m0
HOST_PILOTO=vec-piloto-m0
API="http://127.0.0.1:${PUERTO_WEB}/api_jsonrpc.php"
PY="$DZ/piloto/zabbix_api.py"

log() { printf '[%(%H:%M:%S)T] %s\n' -1 "$*" >&2; }
fallo() { log "ERROR: $*"; exit 1; }

detectar_motor() {
  if [[ -n "${MOTOR:-}" ]]; then :;
  elif command -v podman >/dev/null; then MOTOR=podman;
  elif command -v docker >/dev/null; then MOTOR=docker;
  else fallo "no hay podman ni docker"; fi
  ROOTLESS=no
  if [[ "$MOTOR" == podman ]]; then
    [[ "$(podman info --format '{{.Host.Security.Rootless}}')" == true ]] && ROOTLESS=si
  elif [[ "$(docker info --format '{{.SecurityOptions}}')" == *rootless* ]]; then
    ROOTLESS=si
  fi
  log "motor=$MOTOR rootless=$ROOTLESS"
}

# ---------------------------------------------------------------- recursos
modo_recursos() {
  echo "cpus_logicas=$(nproc)"
  awk '/MemTotal|MemAvailable|SwapTotal/ {printf "%s=%.1fGiB\n", $1, $2/1048576}' /proc/meminfo | tr -d ':'
  echo "carga_1_5_15=$(cut -d' ' -f1-3 /proc/loadavg)"
  df -h --output=target,size,avail,pcent / "$HOME" | sed 's/^/disco /'
  local uid; uid="$(id -u)"
  echo "cgroup_delegado_usuario=$(cat "/sys/fs/cgroup/user.slice/user-$uid.slice/user@$uid.service/cgroup.controllers" 2>/dev/null || echo desconocido)"
}

# -------------------------------------------------------------- coherencia
modo_coherencia() {
  local q="$DZ/quadlet" err=0
  comprobar() { # fichero texto
    if grep -qF -- "$2" "$q/$1"; then echo "OK   $1: $2"; else echo "FALLO $1: falta «$2»"; err=1; fi
  }
  comprobar vec-supervision-pg.container "Image=$IMAGEN_PG"
  comprobar vec-supervision-servidor.container "Image=$IMAGEN_SERVIDOR_SIN_NMAP"
  if grep -qF "FROM $IMAGEN_SERVIDOR" "$DZ/servidor-sin-nmap/Containerfile"; then echo "OK   Containerfile parte de $IMAGEN_SERVIDOR"; else echo "FALLO Containerfile no parte del digest fijado"; err=1; fi
  comprobar vec-supervision-web.container "Image=$IMAGEN_WEB"
  comprobar vec-supervision-agente.container "Image=$IMAGEN_AGENTE"
  comprobar vec-supervision-pg.container "--cpus=$PG_CPUS --memory=$PG_MEMORIA --memory-swap=$PG_MEMORIA --pids-limit=$PG_PIDS --cpu-shares=$CUOTA_CPU_RELATIVA"
  comprobar vec-supervision-servidor.container "--cpus=$SERVIDOR_CPUS --memory=$SERVIDOR_MEMORIA --memory-swap=$SERVIDOR_MEMORIA --pids-limit=$SERVIDOR_PIDS --cpu-shares=$CUOTA_CPU_RELATIVA"
  comprobar vec-supervision-web.container "--cpus=$WEB_CPUS --memory=$WEB_MEMORIA --memory-swap=$WEB_MEMORIA --pids-limit=$WEB_PIDS --cpu-shares=$CUOTA_CPU_RELATIVA"
  comprobar vec-supervision-agente.container "--pid=host --cpus=$AGENTE_CPUS --memory=$AGENTE_MEMORIA --memory-swap=$AGENTE_MEMORIA --pids-limit=$AGENTE_PIDS --cpu-shares=$CUOTA_CPU_RELATIVA"
  comprobar vec-supervision-pg.container "Exec=postgres $PG_AJUSTES"
  comprobar vec-supervision-servidor.container "Environment=$SERVIDOR_AJUSTES"
  comprobar vec-supervision-web.container "Environment=$WEB_AJUSTES"
  if grep -rqiE '\.sock|/run/podman|docker\.sock' "$q"; then echo "FALLO socket del motor montado"; err=1; else echo "OK   ninguna unidad monta el socket del motor"; fi
  if grep -E '^PublishPort=' "$q"/*.container | grep -vqE '=127\.0\.0\.1:'; then echo "FALLO puerto publicado fuera de loopback"; err=1; else echo "OK   puertos publicados solo en 127.0.0.1"; fi
  return "$err"
}

# ---------------------------------------------------------------- rootless
ruta_unidad() { systemctl --user show -p ControlGroup --value "$1"; }

modo_rootless() {
  local uid; uid="$(id -u)"
  local base="/sys/fs/cgroup/user.slice/user-$uid.slice/user@$uid.service"
  local deleg; deleg="$(cat "$base/cgroup.controllers")"
  echo "controladores_delegados=«$deleg»"
  local c
  for c in cpu memory pids io; do
    if [[ " $deleg " == *" $c "* ]]; then echo "delegado_$c=si"; else echo "delegado_$c=NO"; fi
  done

  # CPU: CPUQuota=20 % sobre un bucle activo durante 6 s.
  systemd-run --user --quiet --scope --unit="$P-cpu" -p CPUQuota=20% -- timeout 8 sh -c 'while :; do :; done' &
  sleep 1
  local cg; cg="/sys/fs/cgroup$(ruta_unidad "$P-cpu.scope")"
  local u0; u0="$(awk '/^usage_usec/{print $2}' "$cg/cpu.stat")"
  sleep 5
  local u1; u1="$(awk '/^usage_usec/{print $2}' "$cg/cpu.stat")"
  echo "cpu.max=$(cat "$cg/cpu.max") uso_medido_pct=$(( (u1 - u0) / 50000 )) nr_throttled=$(awk '/^nr_throttled/{print $2}' "$cg/cpu.stat")"
  wait || true

  # Memoria: MemoryMax=64M frente a 200 MiB retenidos -> debe matar por OOM.
  if systemd-run --user --quiet --scope --unit="$P-mem" -p MemoryMax=64M -p MemorySwapMax=0 -- \
      sh -c 'head -c 200m /dev/zero | tail >/dev/null' 2>/dev/null; then
    echo "memoria_limite_aplicado=NO (el proceso terminó bien)"
  else
    echo "memoria_limite_aplicado=si resultado=$(systemctl --user show -p Result --value "$P-mem.scope" 2>/dev/null)"
  fi
  systemctl --user reset-failed "$P-mem.scope" 2>/dev/null || true

  # Pids: TasksMax=8 frente a 20 procesos; pids.events cuenta los rechazos.
  systemd-run --user --quiet --scope --unit="$P-pids" -p TasksMax=8 -- \
      timeout 6 sh -c 'for i in $(seq 20); do sleep 4 & done 2>/dev/null; wait' >/dev/null 2>&1 &
  sleep 2
  cg="/sys/fs/cgroup$(ruta_unidad "$P-pids.scope")"
  echo "pids.max=$(cat "$cg/pids.max") pids.current=$(cat "$cg/pids.current") rechazos=$(awk '/^max/{print $2}' "$cg/pids.events")"
  wait || true
  systemctl --user reset-failed "$P-pids.scope" 2>/dev/null || true

  # E/S: io.max solo existe si el controlador io está delegado.
  systemd-run --user --quiet --scope --unit="$P-io" -p "IOWriteBandwidthMax=/dev/null 1M" -- sleep 3 &
  sleep 1
  cg="/sys/fs/cgroup$(ruta_unidad "$P-io.scope")"
  if [[ -e "$cg/io.max" ]]; then echo "io_limite_disponible=si"; else echo "io_limite_disponible=NO (sin io.max: controlador io no delegado)"; fi
  wait || true

  if command -v podman >/dev/null && [[ "$(podman info --format '{{.Host.Security.Rootless}}')" == true ]]; then
    echo "--- podman rootless real"
    podman run --rm --cpus=0.2 --memory=64m --memory-swap=64m --pids-limit=16 "$IMAGEN_PG" \
      sh -c 'echo cpu.max=$(cat /sys/fs/cgroup/cpu.max) memory.max=$(cat /sys/fs/cgroup/memory.max) pids.max=$(cat /sys/fs/cgroup/pids.max)
             timeout 5 sh -c "while :; do :; done"; grep -E "usage_usec|nr_throttled" /sys/fs/cgroup/cpu.stat | tr "\n" " "; echo'
    if podman run --rm --memory=64m --memory-swap=64m "$IMAGEN_PG" sh -c 'head -c 200m /dev/zero | tail >/dev/null'; then
      echo "podman_memoria_limite_aplicado=NO"
    else
      echo "podman_memoria_limite_aplicado=si (código $?)"
    fi
  else
    echo "--- podman rootless no disponible en este equipo: comprobado el mecanismo de delegación que usa podman"
  fi
}

# ------------------------------------------------------------------ piloto
contenedores() { echo "$P-pg $P-servidor $P-web $P-agente"; }

limpiar() {
  set +e
  log "limpieza: contenedores, redes, volúmenes y ficheros del piloto"
  local c
  for c in $(contenedores); do $MOTOR rm -f "$c" >/dev/null 2>&1; done
  $MOTOR network rm "$P-datos" "$P-frontal" >/dev/null 2>&1
  if [[ "$MOTOR" == podman ]]; then
    podman volume rm -f "$P-pg" >/dev/null 2>&1
    podman secret rm "$P-pg-clave" >/dev/null 2>&1
  fi
  if [[ -d "$DIR_TRABAJO/pg" ]]; then
    $MOTOR run --rm -v "$DIR_TRABAJO:/w" --entrypoint rm "$IMAGEN_PG" -rf /w/pg >/dev/null 2>&1
  fi
  rm -rf "$DIR_TRABAJO/sonda" "$DIR_TRABAJO/secretos" 2>/dev/null
  rmdir /var/tmp/vecsup-m0-sonda 2>/dev/null
  log "limpieza terminada; resultados conservados en $DIR_TRABAJO/resultados"
}

cgrupo_con_limite() { # contenedor -> ruta cgroup que contiene los límites
  local pid rel ruta
  pid="$($MOTOR inspect -f '{{.State.Pid}}' "$1")"
  rel="$(awk -F'::' '/^0::/{print $2}' "/proc/$pid/cgroup")"
  ruta="/sys/fs/cgroup$rel"
  while [[ "$ruta" != /sys/fs/cgroup && "$(cat "$ruta/memory.max" 2>/dev/null)" == max ]]; do ruta="$(dirname "$ruta")"; done
  echo "$ruta"
}

sumar_io() { awk '{for(i=2;i<=NF;i++){split($i,a,"="); if(a[1]==k) t+=a[2]}} END{print t+0}' k="$2" "$1/io.stat" 2>/dev/null || echo 0; }

muestrear() { # fase segundos
  local fin=$(( $(date +%s) + $2 )) c ruta ts linea
  while (( $(date +%s) < fin )); do
    ts="$(date +%s.%N)"
    for c in $(contenedores); do
      ruta="${RUTAS[$c]}"
      linea="$ts,$1,${c#"$P-"}"
      linea+=",$(awk '/^usage_usec/{print $2}' "$ruta/cpu.stat")"
      linea+=",$(awk '/^throttled_usec/{print $2}' "$ruta/cpu.stat")"
      linea+=",$(cat "$ruta/memory.current")"
      linea+=",$(awk '/^anon /{print $2}' "$ruta/memory.stat")"
      linea+=",$(awk '/^file /{print $2}' "$ruta/memory.stat")"
      linea+=",$(sumar_io "$ruta" rbytes),$(sumar_io "$ruta" wbytes)"
      linea+=",$(cat "$ruta/pids.current")"
      echo "$linea" >>"$RES/contenedores.csv"
    done
    echo "$ts,$1,$($MOTOR exec "$P-pg" psql -U zabbix -d zabbix -Atc \
      "select pg_database_size('zabbix'), coalesce(sum(n_tup_ins) filter (where relname like 'history%'),0), coalesce(sum(pg_total_relation_size(relid)) filter (where relname like 'history%' or relname like 'trends%'),0) from pg_stat_user_tables" | tr '|' ',')" >>"$RES/base.csv"
    if [[ "$1" == carga && $(( ${ts%.*} - ULTIMO_PANEL )) -ge 60 ]]; then
      echo "$ts,$(python3 "$PY" panel "$API" "$CLAVE_ADMIN" "$HOST_PILOTO")" >>"$RES/panel_ms.csv"
      ULTIMO_PANEL="${ts%.*}"
    fi
    sleep "$MUESTREO_S"
  done
}

arrancar() {
  # Servidor sin nmap (NPSL, no OSI): se construye desde el digest fijado.
  if ! $MOTOR image inspect "$IMAGEN_SERVIDOR_SIN_NMAP" >/dev/null 2>&1; then
    log "construyendo $IMAGEN_SERVIDOR_SIN_NMAP"
    $MOTOR build -q -f "$DZ/servidor-sin-nmap/Containerfile" -t "$IMAGEN_SERVIDOR_SIN_NMAP" "$DZ/servidor-sin-nmap" >/dev/null
  fi
  mkdir -p "$DIR_TRABAJO/sonda" "$DIR_TRABAJO/secretos" "$RES" /var/tmp/vecsup-m0-sonda
  chmod 700 "$DIR_TRABAJO/secretos"
  CLAVE_ADMIN="$DIR_TRABAJO/secretos/admin"
  local fichero_clave_pg="$DIR_TRABAJO/secretos/pg"
  head -c 32 /dev/urandom | base64 | tr -dc 'A-Za-z0-9' >"$fichero_clave_pg"
  chmod 644 "$fichero_clave_pg" # legible por el UID del contenedor; el directorio es 0700

  # Subredes explícitas (SUBRED_DATOS/SUBRED_FRONTAL) por si el motor agotó sus
  # rangos predefinidos; vacías = rango automático del motor.
  $MOTOR network create --internal ${SUBRED_DATOS:+--subnet "$SUBRED_DATOS"} "$P-datos" >/dev/null
  $MOTOR network create ${SUBRED_FRONTAL:+--subnet "$SUBRED_FRONTAL"} "$P-frontal" >/dev/null

  local secreto_pg vol_pg io_args=()
  if [[ "$MOTOR" == podman ]]; then
    podman secret create "$P-pg-clave" "$fichero_clave_pg" >/dev/null
    secreto_pg=(--secret "$P-pg-clave,type=mount,target=/run/secrets/pg_clave")
    vol_pg=(-v "$P-pg:/var/lib/postgresql")
  else
    secreto_pg=(-v "$fichero_clave_pg:/run/secrets/pg_clave:ro")
    mkdir -p "$DIR_TRABAJO/pg"
    vol_pg=(-v "$DIR_TRABAJO/pg:/var/lib/postgresql")
  fi
  # Límite de E/S del PostgreSQL de supervisión sobre el disco de sus datos.
  local fuente disco
  fuente="$(findmnt -no SOURCE --target "$DIR_TRABAJO")"
  disco="/dev/$(lsblk -no PKNAME "$fuente" 2>/dev/null | head -1)"
  if [[ -b "$disco" && ( "$ROOTLESS" == no || " $(cat "/sys/fs/cgroup/user.slice/user-$(id -u).slice/user@$(id -u).service/cgroup.controllers") " == *" io "* ) ]]; then
    io_args=(--device-write-bps "$disco:$PG_ESCRITURA_BPS" --device-read-bps "$disco:$PG_LECTURA_BPS")
  else
    log "AVISO: sin límite de E/S (controlador io no delegado o disco no detectado)"
  fi
  lim() { echo "--cpus=$1 --memory=$2 --memory-swap=$2 --pids-limit=$3 --cpu-shares=$CUOTA_CPU_RELATIVA"; }
  local comunes=(--security-opt no-new-privileges)

  log "arrancando PostgreSQL de supervisión"
  # shellcheck disable=SC2046
  $MOTOR run -d --name "$P-pg" --network "$P-datos" "${comunes[@]}" $(lim "$PG_CPUS" "$PG_MEMORIA" "$PG_PIDS") "${io_args[@]}" \
    "${secreto_pg[@]}" "${vol_pg[@]}" -e POSTGRES_USER=zabbix -e POSTGRES_DB=zabbix -e POSTGRES_PASSWORD_FILE=/run/secrets/pg_clave \
    "$IMAGEN_PG" postgres $PG_AJUSTES >/dev/null

  for _ in $(seq 60); do $MOTOR exec "$P-pg" pg_isready -U zabbix -q 2>/dev/null && break; sleep 2; done

  local env_srv=() env_web=() kv
  for kv in $SERVIDOR_AJUSTES; do env_srv+=(-e "$kv"); done
  for kv in $WEB_AJUSTES; do env_web+=(-e "$kv"); done
  local db=(-e DB_SERVER_HOST="$P-pg" -e POSTGRES_USER=zabbix -e POSTGRES_DB=zabbix -e POSTGRES_PASSWORD_FILE=/run/secrets/pg_clave)

  log "arrancando servidor Zabbix (crea el esquema la primera vez)"
  # shellcheck disable=SC2046
  $MOTOR run -d --name "$P-servidor" --network "$P-datos" "${comunes[@]}" --cap-drop ALL $(lim "$SERVIDOR_CPUS" "$SERVIDOR_MEMORIA" "$SERVIDOR_PIDS") \
    -p "127.0.0.1:$PUERTO_SERVIDOR:10051" "${secreto_pg[@]}" "${db[@]}" "${env_srv[@]}" "$IMAGEN_SERVIDOR_SIN_NMAP" >/dev/null
  $MOTOR network connect "$P-frontal" "$P-servidor"

  log "arrancando frontend (solo 127.0.0.1:$PUERTO_WEB)"
  # shellcheck disable=SC2046
  $MOTOR run -d --name "$P-web" --network "$P-datos" "${comunes[@]}" $(lim "$WEB_CPUS" "$WEB_MEMORIA" "$WEB_PIDS") \
    -p "127.0.0.1:$PUERTO_WEB:8080" "${secreto_pg[@]}" "${db[@]}" -e ZBX_SERVER_HOST="$P-servidor" -e ZBX_SERVER_NAME=VEC-supervision-piloto \
    "${env_web[@]}" "$IMAGEN_WEB" >/dev/null
  $MOTOR network connect "$P-frontal" "$P-web"

  log "arrancando agente 2 (red y pid del host, cgroup ro, sin socket del motor)"
  # shellcheck disable=SC2046
  $MOTOR run -d --name "$P-agente" --network host --pid host "${comunes[@]}" --cap-drop ALL $(lim "$AGENTE_CPUS" "$AGENTE_MEMORIA" "$AGENTE_PIDS") \
    -v /sys/fs/cgroup:/host/cgroup:ro -v "$DZ/agente/vec_supervision.conf:/etc/zabbix/zabbix_agentd.d/vec_supervision.conf:ro" \
    -v "$DIR_TRABAJO/sonda:/sondas/home:ro" -v /var/tmp/vecsup-m0-sonda:/sondas/raiz:ro \
    -e ZBX_HOSTNAME="$HOST_PILOTO" -e ZBX_SERVER_HOST=127.0.0.1 -e ZBX_SERVER_PORT="$PUERTO_SERVIDOR" \
    -e ZBX_PASSIVE_ALLOW=false -e ZBX_ACTIVE_ALLOW=true "$IMAGEN_AGENTE" >/dev/null

  python3 "$PY" esperar "$API" >"$RES/api_version.txt"
  log "API disponible: $(cat "$RES/api_version.txt")"
}

verificar_limites() {
  local c ruta
  for c in $(contenedores); do RUTAS[$c]="$(cgrupo_con_limite "$c")"; done
  for c in $(contenedores); do
    ruta="${RUTAS[$c]}"
    printf '%s cpu.max=«%s» cpu.weight=%s memory.max=%s memory.swap.max=%s pids.max=%s io.max=«%s»\n' "${c#"$P-"}" \
      "$(cat "$ruta/cpu.max")" "$(cat "$ruta/cpu.weight" 2>/dev/null)" "$(cat "$ruta/memory.max")" \
      "$(cat "$ruta/memory.swap.max" 2>/dev/null)" "$(cat "$ruta/pids.max")" "$(tr '\n' ' ' <"$ruta/io.max" 2>/dev/null)"
  done | tee "$RES/limites.txt"
}

verificar_agente() {
  {
    echo "montajes del agente:"
    $MOTOR inspect -f '{{range .Mounts}}{{.Source}} -> {{.Destination}} ({{if .RW}}rw{{else}}ro{{end}}){{"\n"}}{{end}}' "$P-agente"
    if $MOTOR inspect -f '{{range .Mounts}}{{.Source}}{{"\n"}}{{end}}' "$P-agente" | grep -qE '\.sock$|/run/podman|/var/run/docker'; then
      echo "SOCKET DEL MOTOR MONTADO: FALLO"
    else
      echo "socket del motor: no montado"
    fi
    echo "pruebas de claves en el agente (zabbix_agent2 -t):"
    local k
    for k in 'system.uptime' 'proc.num' 'vm.memory.size[total]' 'vfs.fs.size[/sondas/home,pused]' 'net.if.discovery' \
             'vfs.file.contents[/host/cgroup/../../etc/passwd]' 'system.run[id]' 'vfs.file.contents[/etc/hostname]'; do
      $MOTOR exec "$P-agente" zabbix_agent2 -c /etc/zabbix/zabbix_agent2.conf -t "$k" 2>&1 | cut -c1-160
    done
  } | tee "$RES/agente.txt"
}

resumir() {
  python3 - "$RES" "$REPOSO_MIN" "$CARGA_MIN" <<'PYEOF'
import csv, math, sys, statistics as st
res = sys.argv[1]
filas = list(csv.reader(open(f"{res}/contenedores.csv")))
out = []
def mib(b): return f"{b/1048576:.0f}"
for fase in ("reposo", "carga"):
    out.append(f"== {fase} ==")
    out.append(f"{'componente':10} {'CPU media %':>11} {'CPU p95 %':>9} {'CPU máx %':>9} {'estrang. %':>10} {'mem máx MiB':>11} {'anon máx MiB':>12} {'caché máx MiB':>13} {'lect. KiB/s':>11} {'escr. KiB/s':>11} {'pids máx':>8}")
    tot_cpu = tot_mem = tot_anon = 0.0
    for comp in ("pg", "servidor", "web", "agente"):
        fs = [r for r in filas if r[1] == fase and r[2] == comp]
        if len(fs) < 2: continue
        t = [float(r[0]) for r in fs]; u = [int(r[3]) for r in fs]; th = [int(r[4]) for r in fs]
        cpu = [(u[i]-u[i-1])/((t[i]-t[i-1])*1e4) for i in range(1, len(fs))]
        dt = t[-1]-t[0]
        media = (u[-1]-u[0])/(dt*1e4); estr = (th[-1]-th[0])/(dt*1e4)
        p95 = sorted(cpu)[max(0, math.ceil(len(cpu)*0.95)-1)]
        mem = max(int(r[5]) for r in fs); anon = max(int(r[6]) for r in fs); cache = max(int(r[7]) for r in fs)
        rd = (int(fs[-1][8])-int(fs[0][8]))/dt/1024; wr = (int(fs[-1][9])-int(fs[0][9]))/dt/1024
        pids = max(int(r[10]) for r in fs)
        tot_cpu += media; tot_mem += mem; tot_anon += anon
        out.append(f"{comp:10} {media:11.1f} {p95:9.1f} {max(cpu):9.1f} {estr:10.2f} {mib(mem):>11} {mib(anon):>12} {mib(cache):>13} {rd:11.1f} {wr:11.1f} {pids:8}")
    out.append(f"{'TOTAL':10} {tot_cpu:11.1f} {'':9} {'':9} {'':10} {mib(tot_mem):>11} {mib(tot_anon):>12}")
b = list(csv.reader(open(f"{res}/base.csv")))
for fase in ("reposo", "carga"):
    fs = [r for r in b if r[1] == fase]
    if len(fs) < 2: continue
    dt = float(fs[-1][0]) - float(fs[0][0])
    crec = int(fs[-1][2]) - int(fs[0][2]); ins = int(fs[-1][3]) - int(fs[0][3])
    hist = int(fs[-1][4]) - int(fs[0][4])
    out.append(f"base {fase}: tamaño final {mib(int(fs[-1][2]))} MiB; crecimiento total {crec/1024:.0f} KiB en {dt/60:.1f} min; "
               f"historia+tendencias {hist/1024:.0f} KiB (= {hist/dt*86400/1048576:.1f} MiB/día); "
               f"valores insertados {ins} (= {ins/dt:.1f} valores/s)")
    if ins:
        out.append(f"  bytes por valor en historia ~ {hist/ins:.0f}")
try:
    p = [float(r[1]) for r in csv.reader(open(f"{res}/panel_ms.csv"))]
    out.append(f"consulta tipo panel (API): {len(p)} veces, mediana {st.median(p):.0f} ms, máx {max(p):.0f} ms")
except FileNotFoundError:
    pass
print("\n".join(out))
PYEOF
}

modo_piloto() {
  detectar_motor
  RES="$DIR_TRABAJO/resultados/$(date +%Y%m%dT%H%M%S)"
  mkdir -p "$RES"
  declare -gA RUTAS=()
  ULTIMO_PANEL=0
  trap limpiar EXIT
  echo "ts,fase,componente,usage_usec,throttled_usec,mem_current,anon,file,rbytes,wbytes,pids" >"$RES/contenedores.csv"
  echo "ts,fase,db_bytes,history_ins,historia_bytes" >"$RES/base.csv"
  modo_recursos >"$RES/equipo.txt"
  arrancar
  python3 "$PY" preparar "$API" "$CLAVE_ADMIN"
  verificar_limites
  # La creación del esquema deja escrituras de arranque (checkpoint, autovacuum);
  # se descartan antes de medir el reposo.
  log "estabilización: ${ESTABILIZAR_MIN:-3} min sin medir"
  sleep $(( ${ESTABILIZAR_MIN:-3} * 60 ))
  log "reposo: $REPOSO_MIN min sin elementos supervisados"
  muestrear reposo $(( REPOSO_MIN * 60 ))
  # Directorio que agrupa los contenedores: el padre del cgroup del PostgreSQL.
  local base regex='^docker-\w+\.scope$'
  base="/host/cgroup$(dirname "${RUTAS[$P-pg]#/sys/fs/cgroup}")"
  [[ "$MOTOR" == podman ]] && regex='^(libpod-\w+\.scope|.*\.service)$'
  python3 "$PY" cargar "$API" "$CLAVE_ADMIN" "$DZ/plantillas/vec_cgroup_contenedores.yaml" "$HOST_PILOTO" "$base" "$regex"
  log "carga: $CARGA_MIN min (host + contenedores por cgroup, intervalo 1 min, consulta tipo panel cada 60 s)"
  muestrear carga $(( CARGA_MIN * 60 ))
  python3 "$PY" verificar "$API" "$CLAVE_ADMIN" "$HOST_PILOTO" | tee "$RES/verificacion.json"
  verificar_agente
  $MOTOR exec "$P-pg" du -sb /var/lib/postgresql | tee "$RES/volumen_pg.txt"
  $MOTOR exec "$P-pg" psql -U zabbix -d zabbix -Atc \
    "select relname, pg_total_relation_size(relid) from pg_stat_user_tables order by 2 desc limit 8" | tee "$RES/tablas.txt"
  $MOTOR logs "$P-servidor" 2>&1 | grep -iE 'error|cannot|fail' | cut -c1-200 | tee "$RES/errores_servidor.txt" | wc -l | sed 's/^/lineas_error_servidor=/'
  $MOTOR logs "$P-agente" 2>&1 | grep -iE 'error|cannot|fail' | cut -c1-200 >"$RES/errores_agente.txt" || true
  resumir | tee "$RES/resumen.txt"
}

case "$MODO" in
  recursos) modo_recursos ;;
  coherencia) modo_coherencia ;;
  rootless) modo_rootless ;;
  piloto) modo_piloto ;;
  *) sed -n '2,24p' "$0"; exit 2 ;;
esac
