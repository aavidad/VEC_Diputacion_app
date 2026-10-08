#!/usr/bin/env bash
set -Eeuo pipefail
ejecutar_ensayo_ct193() {
umask 077
ulimit -f 262144  # 256 MiB por fichero del runner y sus salidas privadas.

# CT193: preimagen real construida desde migraciones, PostgreSQL 18 aislado y bundles
# VEC-AD-3 emitidos por el código Go real. No toca la principal ni ejecuta DOWN.
readonly etiqueta='vec.prueba=ct193-e3-pg18'
readonly imagen='postgres@sha256:3a82e1f56c8f0f5616a11103ac3d47e632c3938698946a7ad26da0df1334744a'
readonly prefijo="vec-ct193-e3-${UID}-$$"
readonly contenedor="${prefijo}-db"
readonly volumen="${prefijo}-datos"

directorio="$(cd -- "$(dirname -- "$0")" && pwd -P)"
raiz="$(git -C "$directorio" rev-parse --show-toplevel)"
if [[ ${VEC_CT_E3_BD_DESECHABLE:-} != SI ]]; then
    printf 'CT193 E3 exige VEC_CT_E3_BD_DESECHABLE=SI\n' >&2
    exit 64
fi
for herramienta in docker flock go stat df awk bwrap timeout prlimit sha256sum rg tar; do
    command -v "$herramienta" >/dev/null 2>&1 || {
        printf 'CT193 E3: falta %s\n' "$herramienta" >&2
        exit 69
    }
done
# Un contexto heredado puede enviar incluso `docker run --network none` a un
# daemon remoto. Se exige el socket Unix local antes de cualquier Docker CLI.
for variable in DOCKER_HOST DOCKER_CONTEXT DOCKER_CONFIG DOCKER_TLS DOCKER_TLS_VERIFY DOCKER_CERT_PATH; do
    if [[ -n ${!variable:-} ]]; then
        printf 'CT193 E3: configuración Docker externa no admitida (%s)\n' "$variable" >&2
        exit 65
    fi
done
if [[ ! -S /var/run/docker.sock || -L /var/run/docker.sock ]]; then
    printf 'CT193 E3: se exige /var/run/docker.sock local\n' >&2
    exit 65
fi
docker() { command docker --host unix:///var/run/docker.sock "$@"; }
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off
modcache="$(go env GOMODCACHE)"
if [[ ! -d $modcache || $modcache != /* ]]; then
    printf 'CT193 E3: falta modulecache local\n' >&2
    exit 69
fi
go_bin="$modcache/golang.org/toolchain@v0.0.1-go1.26.6.linux-amd64/bin/go"
if [[ ! -x $go_bin || $($go_bin version) != 'go version go1.26.6 linux/amd64' ]]; then
    printf 'CT193 E3 exige toolchain Go 1.26.6 ya instalado en modulecache local\n' >&2
    exit 69
fi

shopt -s nullglob
migraciones=("$directorio"/migraciones/000193_*.up.sql)
if (( ${#migraciones[@]} != 1 )); then
    printf 'CT193 E3: se exige un único UP 000193 en este worktree\n' >&2
    exit 66
fi
readonly migracion=${migraciones[0]}
readonly sha_ct48='f33cf450fb6189ab0b21712a90ff80f4df95f20fb3a142b13a54bc65c9b6c6cc'
readonly sha_ct165='7a7ac82c0137d77339996022e234c416843a2525cf426f306430c0c66a05bf6e'
readonly migracion_sha='cbef35ad78d8b7551d639b8769f020ac9fd4a56c92a15292d0356d89ec127a2a'
if [[ -n $(git -C "$raiz" status --porcelain=v1) ]]; then
    printf 'CT193 E3: el worktree debe estar limpio para atribuir el ensayo\n' >&2
    exit 65
fi
for especificacion in \
    "$sha_ct48 $directorio/migraciones/000048_replay_alta_tras_reinicio_o2_07.up.sql" \
    "$sha_ct165 $directorio/migraciones/000165_periodo_fin_segun_modalidad.up.sql" \
    "$migracion_sha $migracion"
do
    IFS=' ' read -r esperado ruta_sql <<<"$especificacion"
    if [[ ! -r $ruta_sql || $(sha256sum "$ruta_sql" | cut -d ' ' -f 1) != "$esperado" ]]; then
        printf 'CT193 E3: hash de preimagen/UP incompatible (%s)\n' "${ruta_sql##*/}" >&2
        exit 65
    fi
done
readonly fixture_o205="$raiz/deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/preparar_entorno_o2_05.sql"
readonly helpers_o205="$raiz/deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/ayudantes_o2_05.sql"
readonly fixture_e3="$directorio/pruebas_sql/ct193_e3_preparar.sql"
for ruta in "$fixture_o205" "$helpers_o205" "$fixture_e3"; do
    [[ -r $ruta ]] || { printf 'CT193 E3: falta fixture requerida\n' >&2; exit 66; }
done

# El contenedor solo acepta socket Unix; ninguna variable heredada elige otra BD.
unset PGHOST PGHOSTADDR PGPORT PGDATABASE PGUSER PGPASSWORD PGPASSFILE PGSERVICE
unset PGSERVICEFILE PGOPTIONS PGAPPNAME PGSSLMODE PGCONNECT_TIMEOUT PGCLIENTENCODING
unset PGTARGETSESSIONATTRS PGLOADBALANCEHOSTS PGCHANNELBINDING PGREQUIREAUTH
unset PGSSLCERT PGSSLKEY PGSSLROOTCERT PGSSLCRL PGSSLCRLDIR PGREQUIREPEER
unset PGGSSENCMODE PGKRBSRVNAME PGGSSLIB PGSYSCONFDIR PGLOCALEDIR
unset DATABASE_URL DB_DSN DSN VEC_DATABASE_URL

bloqueo_dir="/run/user/$(id -u)"
readonly bloqueo_dir
if [[ ! -d $bloqueo_dir || -L $bloqueo_dir ||
      $(stat -c '%u:%a' "$bloqueo_dir") != "$(id -u):700" ]]; then
    printf 'CT193 E3: falta directorio de bloqueo privado del UID\n' >&2
    exit 65
fi
# `<>` crea el fichero con umask 077 y no trunca otro bloqueo existente.
exec 9<>"$bloqueo_dir/vec-postgres-dynamic-ct193-e3.lock"
if ! flock -w 900 9; then
    printf 'CT193 E3: no se adquirió el bloqueo PostgreSQL dinámico\n' >&2
    exit 75
fi
if docker container inspect "$contenedor" >/dev/null 2>&1 ||
   docker volume inspect "$volumen" >/dev/null 2>&1; then
    printf 'CT193 E3: colisión de recursos efímeros\n' >&2
    exit 65
fi
imagen_id="$(docker image inspect --format '{{.Id}}' "$imagen" 2>/dev/null || true)"
if [[ $imagen_id != 'sha256:1bf3d6960db467e87a506daef30feb41fecc23b7c5f96b157e873059f2ffb50a' ]]; then
    printf 'CT193 E3: falta la imagen PostgreSQL 18 fijada; no se descargará\n' >&2
    exit 69
fi

# El ensayo ordinario usa datos y temporales en disco, además del GOCACHE
# habitual del operador. Se compara el espacio libre real con la reserva de
# cada uso y se suman reservas cuando comparten sistema de ficheros.
if [[ ! ${HOME:-} == /* || ! -d $HOME ]]; then
    printf 'CT193 E3: HOME local inválido para GOCACHE\n' >&2
    exit 65
fi
export GOCACHE="$HOME/.cache/go-build"
mkdir -p -- "$GOCACHE"
docker_root="$(timeout 5s docker info --format '{{.DockerRootDir}}')" || {
    printf 'CT193 E3: no se pudo comprobar el almacén Docker local\n' >&2
    exit 69
}
if [[ $docker_root != /* || ! -d $docker_root ]]; then
    printf 'CT193 E3: almacén Docker local inválido\n' >&2
    exit 65
fi
declare -A requerido_kib=() libre_kib=() rutas_disco=()
reservar_disco() {
    local ruta=$1 minimo=$2 dispositivo libre
    dispositivo="$(stat -c '%d' -- "$ruta")" || return 1
    libre="$(df -Pk -- "$ruta" | awk 'NR==2 {print $4}')" || return 1
    [[ $libre =~ ^[0-9]+$ ]] || return 1
    requerido_kib[$dispositivo]=$(( ${requerido_kib[$dispositivo]:-0} + minimo ))
    libre_kib[$dispositivo]=$libre
    rutas_disco[$dispositivo]="${rutas_disco[$dispositivo]:-} $ruta"
}
reservar_disco /var/tmp 2097152 &&
    reservar_disco "$docker_root" 3145728 &&
    reservar_disco "$GOCACHE" 1048576 || {
    printf 'CT193 E3: no se pudo medir el espacio libre local\n' >&2
    exit 69
}
for dispositivo in "${!requerido_kib[@]}"; do
    printf '[CT193:E3] preflight_disco dispositivo=%s rutas=%s disponible_kib=%s minimo_kib=%s\n' \
        "$dispositivo" "${rutas_disco[$dispositivo]}" \
        "${libre_kib[$dispositivo]}" "${requerido_kib[$dispositivo]}"
    if (( libre_kib[$dispositivo] < requerido_kib[$dispositivo] )); then
        printf 'CT193 E3: espacio libre insuficiente para scratch, Docker y GOCACHE (%s KiB disponibles; %s KiB requeridos)\n' \
            "${libre_kib[$dispositivo]}" "${requerido_kib[$dispositivo]}" >&2
        exit 78
    fi
done

base="$(mktemp -d "/var/tmp/${prefijo}.XXXXXX")"
temporal="$base/datos"
sandbox_root="$base/raiz"
raiz_export="$base/repo"
mkdir -m 0700 -- "$temporal" "$sandbox_root" "$raiz_export"
socket="$temporal/socket"
vectores="$temporal/vectores"
mkdir -m 0777 -- "$socket"
mkdir -m 0700 -- "$vectores"
creado_volumen=0
creado_contenedor=0
limpiar() {
    local estado=$?
    local etiqueta_real
    trap - EXIT INT TERM
    if (( creado_contenedor )) && timeout 5s docker container inspect "$contenedor" >/dev/null 2>&1; then
        etiqueta_real="$(docker container inspect --format '{{index .Config.Labels "vec.prueba"}}' "$contenedor" 2>/dev/null)" || estado=1
        if [[ $etiqueta_real == ct193-e3-pg18 ]]; then
            # postgres cambia la propiedad del bind de sockets y activa sticky bit.
            timeout 10s docker exec -u 0 "$contenedor" chown -R "$(id -u):$(id -g)" \
                /var/run/postgresql >/dev/null 2>&1 || estado=1
            timeout 15s docker rm -f -v "$contenedor" >/dev/null 2>&1 || estado=1
        else
            printf 'CT193 E3: etiqueta de contenedor ajena; no se elimina\n' >&2
            estado=1
        fi
    elif (( creado_contenedor )) && ! timeout 5s docker info >/dev/null 2>&1; then
        printf 'CT193 E3: daemon inaccesible; limpieza de contenedor no verificable\n' >&2
        estado=1
    fi
    if (( creado_volumen )) && timeout 5s docker volume inspect "$volumen" >/dev/null 2>&1; then
        etiqueta_real="$(docker volume inspect --format '{{index .Labels "vec.prueba"}}' "$volumen" 2>/dev/null)" || estado=1
        if [[ $etiqueta_real == ct193-e3-pg18 ]]; then
            timeout 15s docker volume rm -- "$volumen" >/dev/null 2>&1 || estado=1
        else
            printf 'CT193 E3: etiqueta de volumen ajena; no se elimina\n' >&2
            estado=1
        fi
    elif (( creado_volumen )) && ! timeout 5s docker info >/dev/null 2>&1; then
        printf 'CT193 E3: daemon inaccesible; limpieza de volumen no verificable\n' >&2
        estado=1
    fi
    rm -rf -- "$base" || estado=1
    if timeout 5s docker container inspect "$contenedor" >/dev/null 2>&1 ||
       timeout 5s docker volume inspect "$volumen" >/dev/null 2>&1; then
        printf 'CT193 E3: limpieza incompleta\n' >&2
        estado=1
    elif ! timeout 5s docker info >/dev/null 2>&1; then
        printf 'CT193 E3: daemon inaccesible; ausencia de recursos no verificable\n' >&2
        estado=1
    fi
    exit "$estado"
}
trap limpiar EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

# `git status` no muestra ignorados: solo el archivo HEAD versionado entra al
# contenedor y al proceso Go. Ningún secreto local ignorado queda montado.
if ! git -C "$raiz" archive --format=tar HEAD | tar -xf - -C "$raiz_export"; then
    printf 'CT193 E3: no se pudo extraer la fuente versionada aislada\n' >&2
    exit 1
fi

creado_volumen=1
docker volume create --label "$etiqueta" "$volumen" >/dev/null
creado_contenedor=1
iniciar_contenedor() {
docker run -d --pull=never --rm --restart=no --name "$contenedor" --label "$etiqueta" \
    --network none --memory=2g --cpus=2 --pids-limit=256 --log-driver=none \
    --mount "type=volume,src=$volumen,dst=/var/lib/postgresql" \
    --mount "type=bind,src=$socket,dst=/var/run/postgresql" \
    --mount "type=bind,src=$raiz_export,dst=/repo,readonly" \
    --env POSTGRES_HOST_AUTH_METHOD=trust --env POSTGRES_INITDB_ARGS='--encoding=UTF8' \
    "$imagen" -c listen_addresses= -c unix_socket_directories=/var/run/postgresql >/dev/null
}
iniciar_contenedor

listo=0
for _ in {1..80}; do
    if docker exec "$contenedor" pg_isready -q -h /var/run/postgresql -U postgres 2>/dev/null; then
        listo=1
        break
    fi
    sleep 0.25
done
if (( ! listo )); then
    printf 'CT193 E3: PostgreSQL 18 no quedó listo\n' >&2
    exit 1
fi
psql_admin() {
    docker exec -i "$contenedor" psql -X --no-psqlrc -q -v ON_ERROR_STOP=1 \
        -h /var/run/postgresql -U postgres -d postgres "$@"
}
psql_como() {
    local usuario=$1
    shift
    docker exec -i "$contenedor" psql -X --no-psqlrc -q -v ON_ERROR_STOP=1 \
        -h /var/run/postgresql -U "$usuario" -d postgres "$@"
}
consultar() {
    psql_admin -At --command "$1" 2>/dev/null
}
cargar() {
    local nombre=$1 archivo=$2
    if [[ $archivo != "$raiz/"* ]]; then
        printf 'CT193 E3: ruta SQL fuera del worktree (%s)\n' "$nombre" >&2
        exit 1
    fi
    if ! psql_admin --file "/repo/${archivo#"$raiz/"}" \
        >"$temporal/psql.log" 2>&1; then
        printf 'CT193 E3: falló %s; salida SQL privada omitida\n' "$nombre" >&2
        exit 1
    fi
}

if [[ $(consultar "SELECT current_database()||'|'||(current_setting('server_version_num')::integer/10000)") != 'postgres|18' ]]; then
    printf 'CT193 E3: destino PostgreSQL inesperado\n' >&2
    exit 1
fi
printf '[CT193:E3] construyendo preimagen real en PostgreSQL 18 sin red\n'
if ! psql_admin >"$temporal/psql.log" 2>&1 <<'SQL'
DO $cierre$
BEGIN
 EXECUTE pg_catalog.format('REVOKE ALL PRIVILEGES ON DATABASE %I FROM PUBLIC',
                           pg_catalog.current_database());
END $cierre$;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SQL
then
    printf 'CT193 E3: no se cerró PUBLIC en la base efímera\n' >&2
    exit 1
fi
for ruta in \
    contexto_actor_v1/roles_up.sql \
    contexto_actor_v1/migraciones/000001_contexto_actor_v1.up.sql \
    contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql \
    contexto_actor_v1/pruebas_sql/fixtures_sinteticos.sql \
    autorizacion/pruebas_sql/fixture_contexto_actor_v3.sql \
    contratacion_temporal/pruebas_sql/fixture_contexto_actor_b_o3.sql
do
    cargar "$ruta" "$raiz/deploy/postgresql/$ruta"
done
if ! psql_admin --command \
    'CREATE ROLE vec_contexto_ct193_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS; GRANT vec_contexto_actor_v1_runtime TO vec_contexto_ct193_runtime WITH ADMIN FALSE, INHERIT TRUE, SET FALSE' \
    >"$temporal/psql.log" 2>&1; then
    printf 'CT193 E3: no se creó runtime sintético de contexto\n' >&2
    exit 1
fi
if ! psql_como vec_contexto_ct193_runtime >"$temporal/psql.log" 2>&1 <<'SQL'
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SELECT count(*) FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(
 'oca_registro_v3_000000000000000000000000',
 'rca_registro_v3_000000000000000000000000',
 'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa',
 'prf_sintetico_cccccccccccccccccccccccc',
 'certificado','alto',clock_timestamp());
COMMIT;
SQL
then
    printf 'CT193 E3: no se resolvió el actor sintético R3B\n' >&2
    exit 1
fi
if ! psql_admin --command \
    'CREATE EXTENSION pgcrypto WITH SCHEMA public; REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA public FROM PUBLIC' \
    >"$temporal/psql.log" 2>&1; then
    printf 'CT193 E3: pgcrypto no disponible\n' >&2
    exit 1
fi
for ruta in \
    autorizacion/roles_up.sql \
    autorizacion/roles_v2_up.sql \
    autorizacion/migraciones/000001_autorizacion.up.sql \
    ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql \
    autorizacion/migraciones/000003_proyeccion_motivos_autorizacion_v2.up.sql \
    autorizacion/migraciones/000004_registro_decisiones_solicitud_ligada_v2.up.sql \
    autorizacion/migraciones/000005_registro_decisiones_contexto_actor_v3.up.sql \
    autorizacion/migraciones/000006_funcion_registro_decisiones_contexto_actor_v3.up.sql \
    autorizacion/pruebas_sql/fixture_autorizacion_contexto_actor_v3.sql \
    autorizacion/pruebas_sql/integracion_contexto_actor_v3.sql \
    autorizacion/migraciones/000007_revalidacion_viva_decision_contexto_actor_v3.up.sql \
    contratacion_temporal/roles_up.sql \
    contratacion_temporal/migraciones_autorizacion/000001_revalidacion_analisis_v3.up.sql \
    autorizacion_atestada_v3/roles_up.sql \
    autorizacion_atestada_v3/migraciones/000001_gobierno_y_registro_v3.up.sql \
    autorizacion_atestada_v3/migraciones/000002_consumidor_capacidad_v3.up.sql
do
    cargar "$ruta" "$raiz/deploy/postgresql/$ruta"
done
for n in 1 2 3 4 5 6 7 8 9 10 11 13 46 47 48; do
    printf -v numero '%06d' "$n"
    archivos=("$directorio/migraciones/${numero}_"*.up.sql)
    if (( ${#archivos[@]} != 1 )); then
        printf 'CT193 E3: falta UP CT%s único en la cola causal\n' "$numero" >&2
        exit 1
    fi
    cargar "CT$numero" "${archivos[0]}"
done
# CT165 conserva marcas exactas de análisis, cobertura y detalle RRHH.
# Esta cola instala sus dueños reales y barreras en orden, sin funciones falsas.
for n in 12 14 15; do
    printf -v numero '%06d' "$n"
    archivos=("$directorio/migraciones/${numero}_"*.up.sql)
    [[ ${#archivos[@]} == 1 ]] || { printf 'CT193 E3: falta CT%s\n' "$numero" >&2; exit 1; }
    cargar "CT$numero" "${archivos[0]}"
done
for n in 2 3 4 5 6; do
    printf -v numero '%06d' "$n"
    archivos=("$directorio/migraciones_autorizacion/${numero}_"*.up.sql)
    [[ ${#archivos[@]} == 1 ]] || { printf 'CT193 E3: falta AUT-CT%s\n' "$numero" >&2; exit 1; }
    cargar "AUT-CT$numero" "${archivos[0]}"
done
for n in $(seq 17 27); do
    printf -v numero '%06d' "$n"
    archivos=("$directorio/migraciones/${numero}_"*.up.sql)
    [[ ${#archivos[@]} == 1 ]] || { printf 'CT193 E3: falta CT%s\n' "$numero" >&2; exit 1; }
    cargar "CT$numero" "${archivos[0]}"
done
cargar 'roles confirmador cobertura' "$directorio/roles_confirmador_cobertura_up.sql"
for n in $(seq 28 34); do
    printf -v numero '%06d' "$n"
    archivos=("$directorio/migraciones/${numero}_"*.up.sql)
    [[ ${#archivos[@]} == 1 ]] || { printf 'CT193 E3: falta CT%s\n' "$numero" >&2; exit 1; }
    cargar "CT$numero" "${archivos[0]}"
done
cargar 'roles lector resultado cobertura' "$directorio/roles_lector_resultado_cobertura_up.sql"
cargar 'CT000035' "$directorio/migraciones/000035_recuperacion_propia_cobertura_o4_05.up.sql"
cargar 'CT000035a' "$directorio/migraciones/000035a_compatibilidad_barreras_decision_cobertura.up.sql"
cargar 'roles consultor RRHH' "$directorio/roles_consultor_rrhh_up.sql"
for n in 36 37; do
    printf -v numero '%06d' "$n"
    archivos=("$directorio/migraciones/${numero}_"*.up.sql)
    cargar "CT$numero" "${archivos[0]}"
done
for ruta in \
    identidad_sesiones_v1/roles_up.sql \
    identidad_sesiones_v1/migraciones_autorizacion/000001_capacidad_tablas_v1.up.sql \
    identidad_sesiones_v1/migraciones/000001_registro_base_v1.up.sql \
    identidad_sesiones_v1/migraciones/000002_operaciones_v1.up.sql \
    identidad_sesiones_v1/migraciones/000003_revalidacion_autenticacion_actor_v1.up.sql \
    contratacion_temporal/roles_cursor_rrhh_up.sql
do
    cargar "$ruta" "$raiz/deploy/postgresql/$ruta"
done
cargar 'CT000038' "$directorio/migraciones/000038_cursores_cuadro_rrhh.up.sql"
cargar 'CT identidad consulta RRHH' \
    "$directorio/migraciones_identidad/000001_revalidacion_consulta_rrhh_v1.up.sql"
for n in 39 40 41 42; do
    printf -v numero '%06d' "$n"
    archivos=("$directorio/migraciones/${numero}_"*.up.sql)
    cargar "CT$numero" "${archivos[0]}"
done
for n in 3 4 5 6; do
    printf -v numero '%06d' "$n"
    archivos=("$raiz/deploy/postgresql/autorizacion_atestada_v3/migraciones/${numero}_"*.up.sql)
    cargar "AD3-$numero" "${archivos[0]}"
done
cargar 'CT000043' "$directorio/migraciones/000043_prueba_resultado_recibo_rrhh.up.sql"
cargar 'CT000043a' "$directorio/migraciones/000043a_detalle_version_actual.up.sql"
cargar 'CT000044' "$directorio/migraciones/000044_motor_consultas_rrhh.up.sql"
for n in 67 102 106 159 165; do
    printf -v numero '%06d' "$n"
    archivos=("$directorio/migraciones/${numero}_"*.up.sql)
    [[ ${#archivos[@]} == 1 ]] || { printf 'CT193 E3: falta CT%s\n' "$numero" >&2; exit 1; }
    cargar "CT$numero" "${archivos[0]}"
done
preflight="$(consultar "SELECT concat_ws('|',
 (to_regprocedure('vec_contratacion_temporal.confirmar_alta_atestada_v3(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)') IS NOT NULL),
 (to_regprocedure('vec_contratacion_temporal.periodo_previsto_estructural_v1(jsonb)') IS NOT NULL),
 (to_regprocedure('vec_contratacion_temporal.necesidad_alta_valida_v3(jsonb)') IS NULL),
 (to_regprocedure('vec_contratacion_temporal.reconstruir_efecto_alta_v3(jsonb)') IS NULL),
 (to_regclass('public.vectores_o2_05') IS NULL),
 (to_regclass('vec_contratacion_temporal.expediente_alta_version') IS NOT NULL))")"
if [[ $preflight != 't|t|t|t|t|t' ]]; then
    printf 'CT193 E3: preimagen incompatible; se exige CT48+CT165 y CT193 ausente\n' >&2
    exit 1
fi
if [[ ${VEC_CT_E3_SOLO_PREIMAGEN:-} == SI ]]; then
    printf '[CT193:E3] SOLO PREIMAGEN CT48+CT165: sin CT193 ni prueba firmada; NO-GO\n' >&2
    exit 77
fi
if [[ $(consultar "SELECT EXISTS(SELECT 1 FROM vec_autorizacion.decision_concedida_contexto_actor_v3 WHERE decision_ref='decision:registro-v3:positiva')") != t ]]; then
    printf 'CT193 E3: la preimagen carece del contexto/decisión sintéticos O2-05\n' >&2
    exit 1
fi

psql_admin --command \
    'GRANT EXECUTE ON FUNCTION public.gen_random_bytes(integer) TO vec_autorizacion_atestada_v3_propietario' \
    >"$temporal/psql.log" 2>&1 || { printf 'CT193 E3: falta pgcrypto/ACL O2-05\n' >&2; exit 1; }
cargar 'fixture O2-05' "$fixture_o205"
cargar 'helpers O2-05' "$helpers_o205"
if ! psql_admin >"$temporal/psql.log" 2>&1 <<'SQL'
CREATE ROLE vec_ct_e3_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
 INHERIT NOREPLICATION NOBYPASSRLS;
GRANT CONNECT ON DATABASE postgres TO vec_ct_e3_runtime;
GRANT vec_contratacion_temporal_ejecutor TO vec_ct_e3_runtime
 WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
CREATE TABLE public.ct193_e3_desechable (
 marca text PRIMARY KEY CHECK (marca='SI')
);
INSERT INTO public.ct193_e3_desechable(marca) VALUES ('SI');
REVOKE ALL ON TABLE public.ct193_e3_desechable FROM PUBLIC;
SQL
then
    printf 'CT193 E3: no se pudo crear runtime/marker desechables\n' >&2
    exit 1
fi

export GOMODCACHE="$modcache"
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off
export TMPDIR="$temporal"
# La raíz del sandbox es hermana del scratch montado, por lo que el proceso
# de prueba no puede modificarla a través de su único bind de escritura.
mkdir -p -- "$sandbox_root/usr" "$sandbox_root/etc" \
    "$sandbox_root/dev" "$sandbox_root/proc" \
    "$sandbox_root$raiz" "$sandbox_root$modcache" \
    "$sandbox_root$temporal" "$sandbox_root$GOCACHE"
ln -s usr/bin "$sandbox_root/bin"
ln -s usr/lib "$sandbox_root/lib"
ln -s usr/lib64 "$sandbox_root/lib64"
# El código Go de la prueba solo ve toolchain, módulo y repo de lectura,
# el GOCACHE ordinario y el scratch de socket, vectores y recibos. Sin red.
sandbox=(
    bwrap --die-with-parent --unshare-net --unshare-pid --unshare-ipc
    --unshare-uts --clearenv
    --ro-bind "$sandbox_root" /
    --ro-bind /usr /usr
    --dev /dev --proc /proc
    --ro-bind "$raiz_export" "$raiz"
    --ro-bind "$modcache" "$modcache"
    --bind "$GOCACHE" "$GOCACHE"
    --bind "$temporal" "$temporal"
    --setenv PATH "$modcache/golang.org/toolchain@v0.0.1-go1.26.6.linux-amd64/bin:/usr/bin:/bin"
    --setenv HOME "$HOME" --setenv GOCACHE "$GOCACHE"
    --setenv GOMODCACHE "$GOMODCACHE" --setenv GOTOOLCHAIN local
    --setenv GOPROXY off --setenv GOSUMDB off --setenv TMPDIR "$TMPDIR"
    --setenv LANG C.UTF-8 --chdir "$raiz"
)
emisor="$temporal/emisor-o205.test"
adaptador="$temporal/confirmacion-e3.test"
(
    timeout --signal=TERM --kill-after=10s 600s \
    prlimit --as=8589934592 --cpu=600 --nproc=8192 --fsize=134217728 -- \
    "${sandbox[@]}" -- "$go_bin" test -buildvcs=false -p 6 -c -o "$emisor" \
        ./internal/vec/adapters/seguridad/confianzaatestacion
    timeout --signal=TERM --kill-after=10s 600s \
    prlimit --as=8589934592 --cpu=600 --nproc=8192 --fsize=134217728 -- \
    "${sandbox[@]}" -- "$go_bin" test -buildvcs=false -p 6 -c -o "$adaptador" \
        ./internal/modules/contrataciontemporal/adapters/postgres
) >"$temporal/go-build.log" 2>&1 || {
    printf 'CT193 E3: falló compilación Go; primeras 20 líneas acotadas del compilador:\n' >&2
    sed -n '1,20p' "$temporal/go-build.log" | cut -c 1-300 >&2
    exit 1
}

probar_fase() {
    local fase=$1
    local caso estado_antes estado_despues
    case $fase in
        e2_pre|e2_post) caso=e2_pre ;;
        e3|e3_replay|colision) caso=e3_alta ;;
        *) caso=$fase ;;
    esac
    estado_antes="$(estado_caso "$caso")"
    if ! timeout --signal=TERM --kill-after=5s 45s \
        prlimit --as=4294967296 --cpu=40 --nproc=8192 --fsize=67108864 -- \
        "${sandbox[@]}" \
        --setenv VEC_CT_E3_PG18 SI --setenv VEC_CT_E3_FASE "$fase" \
        --setenv VEC_CT_E3_VECTORES_DIR "$vectores" \
        --setenv VEC_CT_E3_RECIBO_E2 "$temporal/recibo-e2.json" \
        --setenv VEC_CT_E3_RECIBO_E3 "$temporal/recibo-e3.json" \
        --setenv VEC_CT_E3_INICIO_PG "$temporal/inicio-pg.json" \
        --setenv VEC_CT_E3_RUNTIME_DSN "host=$socket port=5432 dbname=postgres user=vec_ct_e3_runtime sslmode=disable" \
        --setenv VEC_CT_E3_ADMIN_DSN "host=$socket port=5432 dbname=postgres user=postgres sslmode=disable" \
        -- "$adaptador" -test.run '^TestConfirmacionAltaV3PostgreSQL18$' -test.count=1 -test.v \
        >"$temporal/go-fase.log" 2>&1; then
        printf 'CT193 E3: fase %s falló; salida privada omitida\n' "$fase" >&2
        diagnostico="$(rg -m1 'confirmacion_alta_v3_postgresql18_test[.]go:[0-9]+:' \
            "$temporal/go-fase.log" | cut -c1-1000 || true)"
        if [[ -n $diagnostico ]]; then
            printf 'CT193 E3: diagnóstico focal sintético: %s\n' "$diagnostico" >&2
        fi
        exit 1
    fi
    if ! rg -q -- '^--- PASS: TestConfirmacionAltaV3PostgreSQL18 \(' "$temporal/go-fase.log"; then
        printf 'CT193 E3: fase %s sin marcador de prueba ejecutada; NO-GO\n' "$fase" >&2
        exit 1
    fi
    estado_despues="$(estado_caso "$caso")"
    printf '[CT193:E3] fase %s OK; versiones/actuaciones/auditoría/outbox/recibo válido %s → %s\n' \
        "$fase" "$estado_antes" "$estado_despues"
}

estado_caso() {
    local caso=$1
    consultar "WITH objetivo AS (
      SELECT convert_from(alta,'UTF8')::jsonb->>'expediente_ref' AS expediente_ref
        FROM public.vectores_o2_05 WHERE caso='$caso'
    ) SELECT concat_ws('/',
       (SELECT count(*) FROM vec_contratacion_temporal.expediente_alta_version v
         JOIN objetivo o USING (expediente_ref)),
       (SELECT count(*) FROM vec_contratacion_temporal.actuacion_alta a
         JOIN objetivo o USING (expediente_ref)),
       (SELECT count(*) FROM vec_contratacion_temporal.auditoria_alta a
         JOIN objetivo o USING (expediente_ref)),
       (SELECT count(*) FROM vec_contratacion_temporal.outbox_alta x
         JOIN objetivo o USING (expediente_ref)),
       (SELECT count(*) FROM vec_contratacion_temporal.confirmacion_agregado_alta c
         JOIN objetivo o USING (expediente_ref)
        WHERE c.recibo_ref<>'' AND c.recibo_huella_sha256~'^[0-9a-f]{64}$'))"
}

emitir_aplicar() {
    local caso=$1 nombre=$2
    local entrada="$temporal/entrada-${caso}.json"
    local bundle="$temporal/bundle-${caso}.json"
    local remoto="/tmp/bundle-${caso}.json"
    consultar "SELECT public.exportar_entrada_go_o2_05('$caso')" >"$entrada"
    chmod 600 "$entrada"
    if ! timeout --signal=TERM --kill-after=5s 30s \
        prlimit --as=4294967296 --cpu=25 --nproc=8192 --fsize=67108864 -- \
        "${sandbox[@]}" \
        --setenv VEC_O205_VECTOR_ENTRADA "$entrada" \
        --setenv VEC_O205_VECTOR_SALIDA "$bundle" \
        -- "$emisor" -test.run '^TestGenerarVectorO205ParaSQL$' -test.count=1 \
        >"$temporal/go-emisor.log" 2>&1; then
        printf 'CT193 E3: emisor O2-05 falló en %s\n' "$caso" >&2
        exit 1
    fi
    [[ -s $bundle ]] || { printf 'CT193 E3: bundle vacío en %s\n' "$caso" >&2; exit 1; }
    docker cp "$bundle" "$contenedor:$remoto"
    docker exec "$contenedor" chown postgres:postgres "$remoto"
    docker exec "$contenedor" chmod 600 "$remoto"
    if ! psql_admin --command \
        "SELECT public.aplicar_bundle_go_o2_05('$caso',pg_catalog.pg_read_file('$remoto')::jsonb)" \
        >"$temporal/psql.log" 2>&1; then
        printf 'CT193 E3: no se aplicó el bundle de %s\n' "$caso" >&2
        exit 1
    fi
    ligadura="$(consultar "WITH v AS (
      SELECT *, convert_from(capacidad,'UTF8')::jsonb AS c,
             convert_from(decision,'UTF8')::jsonb AS d,
             convert_from(contexto,'UTF8')::jsonb AS x
        FROM public.vectores_o2_05 WHERE caso='$caso'
    ) SELECT coalesce(string_agg(k.nombre,',' ORDER BY k.nombre)
             FILTER (WHERE k.valido IS NOT TRUE),'OK')
      FROM v CROSS JOIN LATERAL (VALUES
       ('persona_version',(v.x->>'persona_version') IS NOT DISTINCT FROM v.persona_version::text),
       ('perfil_version',(v.x->>'perfil_version') IS NOT DISTINCT FROM v.perfil_version::text),
       ('contexto',(v.c->>'huella_contexto_sha256') IS NOT DISTINCT FROM encode(sha256(v.contexto),'hex')),
       ('decision',(v.c->>'huella_decision_sha256') IS NOT DISTINCT FROM encode(sha256(v.decision),'hex')),
       ('motivo',(v.c->>'huella_motivo_sha256') IS NOT DISTINCT FROM encode(sha256(v.motivo),'hex')),
       ('payload',(v.c->>'huella_payload_vec_ad_3_sha256') IS NOT DISTINCT FROM encode(sha256(v.payload),'hex')),
       ('cose',(v.c->>'huella_sobre_cose_sign1_sha256') IS NOT DISTINCT FROM encode(sha256(v.cose),'hex')),
       ('evidencia',(v.c->>'huella_prueba_confianza_sha256') IS NOT DISTINCT FROM encode(sha256(v.evidencia),'hex')),
       ('spki',(v.c->>'huella_raiz_spki_sha256') IS NOT DISTINCT FROM encode(sha256(v.spki),'hex')),
       ('decision_ref',(v.d->>'decision_ref') IS NOT DISTINCT FROM v.c->>'decision_ref'),
       ('motivo_decision',(v.d->>'motivo_huella_sha256') IS NOT DISTINCT FROM v.c->>'huella_motivo_sha256'),
       ('accion',(v.d->>'accion') IS NOT DISTINCT FROM v.c->>'operacion'),
       ('recurso',(v.d->>'recurso_ref') IS NOT DISTINCT FROM v.c->>'efecto_ref'),
       ('digest_recurso',(v.d->>'contexto_recurso_huella_sha256') IS NOT DISTINCT FROM v.c->>'huella_efecto_sha256'),
       ('decision_vigencia',(v.d->>'valida_hasta') IS NOT DISTINCT FROM v.c->>'decision_valida_hasta'),
       ('contexto_ref',(v.d#>>'{vinculo_autenticacion_actor,registro_contexto_ref}') IS NOT DISTINCT FROM v.c->>'contexto_ref'),
       ('actor',(v.d->>'principal_id') IS NOT DISTINCT FROM v.x->>'principal_ref'),
       ('perfil',(v.d->>'perfil_activo_ref') IS NOT DISTINCT FROM v.x->>'perfil_activo_ref')
      ) AS k(nombre,valido)")"
    if [[ $ligadura != OK ]]; then
        printf 'CT193 E3: ligadura estructural divergente en %s: %s\n' \
            "$caso" "$ligadura" >&2
        exit 1
    fi
    consistencia="$(consultar "WITH v AS (
      SELECT *, convert_from(capacidad,'UTF8')::jsonb AS c,
             convert_from(decision,'UTF8')::jsonb AS d
        FROM public.vectores_o2_05 WHERE caso='$caso'
    ), b AS (SELECT pg_catalog.pg_read_file('$remoto')::jsonb AS j)
    SELECT concat_ws('|',
       ((v.capacidad=decode(b.j->>'capacidad_b64','base64')) AND
        (v.decision=decode(b.j->>'decision_b64','base64')) AND
        (v.contexto=decode(b.j->>'contexto_b64','base64')) AND
        (v.alta=decode(b.j->>'alta_b64','base64')) AND
        (v.sellos=decode(b.j->>'sellos_b64','base64'))),
       (SELECT p.configuracion_revision=v.c->>'revision_confianza'
          FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual p
          ORDER BY p.orden DESC LIMIT 1),
       (SELECT p.clave_id=v.c->>'clave_id' AND p.version=(v.c->>'clave_version')::numeric
          FROM vec_autorizacion_atestada_v3.puntero_clave_emision p
          ORDER BY p.orden DESC LIMIT 1),
       (SELECT a.asignacion_ref=v.d->>'asignacion_ref'
          FROM vec_autorizacion.asignacion_perfil_actual a
          WHERE a.perfil_activo_ref=v.d->>'perfil_activo_ref'),
       floor(extract(epoch FROM (clock_timestamp()-
           (v.c->>'emitida_en')::timestamptz))*1000)::integer)
      FROM v CROSS JOIN b")"
    IFS='|' read -r bytes_ok gobierno_ok clave_ok asignacion_ok edad_ms <<<"$consistencia"
    if [[ $bytes_ok != t || $gobierno_ok != t || $clave_ok != t ||
          $asignacion_ok != t || ! $edad_ms =~ ^[0-9]+$ ]] ||
       (( edad_ms >= 4000 )); then
        printf 'CT193 E3: bundle/gobierno no fresco en %s: bytes=%s gobierno=%s clave=%s asignación=%s edad_ms=%s\n' \
            "$caso" "$bytes_ok" "$gobierno_ok" "$clave_ok" "$asignacion_ok" "$edad_ms" >&2
        exit 1
    fi
    printf '[CT193:E3] bundle %s: bytes/gobierno/clave/asignación OK, edad %sms antes del test\n' \
        "$caso" "$edad_ms"
    docker exec "$contenedor" rm -f -- "$remoto"
    mv -- "$bundle" "$vectores/$nombre.json"
    chmod 600 "$vectores/$nombre.json"
    rm -f -- "$entrada"
}

if ! psql_admin >"$temporal/psql.log" 2>&1 <<'SQL'
SELECT public.preparar_vector_o2_05('e2_pre','valido',1);
WITH original AS (
 SELECT convert_from(alta,'UTF8')::jsonb AS a
 FROM public.vectores_o2_05 WHERE caso='e2_pre'
), normalizado AS (
 SELECT jsonb_set(jsonb_set(jsonb_set(jsonb_set(
     a,'{solicitud,periodo,inicio}',to_jsonb(left(a#>>'{solicitud,periodo,inicio}',10))),
     '{solicitud,periodo,fin}',to_jsonb(left(a#>>'{solicitud,periodo,fin}',10))),
     '{solicitud,rc,fecha}','""'::jsonb),
     '{solicitud,rc,importe,moneda}','"EUR"'::jsonb) AS a
 FROM original
)
UPDATE public.vectores_o2_05 AS v
 SET alta=vec_contratacion_temporal.reconstruir_efecto_alta_v2(n.a)
 FROM normalizado AS n WHERE v.caso='e2_pre';
SQL
then
    printf 'CT193 E3: no se preparó E2\n' >&2
    exit 1
fi
emitir_aplicar e2_pre e2
probar_fase e2_pre

printf '[CT193:E3] instalando CT193 una sola vez sobre la preimagen validada\n'
if [[ $(sha256sum "$migracion" | cut -d ' ' -f 1) != "$migracion_sha" ]]; then
    printf 'CT193 E3: cambió el UP durante el ensayo; se descarta la base efímera\n' >&2
    exit 1
fi
printf '[CT193:E3] UP SHA256 %s\n' "$migracion_sha"
cargar 'CT193 UP' "$migracion"
if [[ $(consultar "SELECT concat_ws('|',
 (to_regprocedure('vec_contratacion_temporal.necesidad_alta_valida_v3(jsonb)') IS NOT NULL),
 (to_regprocedure('vec_contratacion_temporal.reconstruir_efecto_alta_v3(jsonb)') IS NOT NULL))") != 't|t' ]]; then
    printf 'CT193 E3: postimagen incompleta\n' >&2
    exit 1
fi
probar_fase e2_post

for fixture in ct193_canon_necesidad_pg18.sql ct193_reglas_causa_pg18.sql ct193_lectura_estados_pg18.sql; do
    cargar "$fixture" "$directorio/pruebas_sql/$fixture"
done
cargar 'preparador E3' "$fixture_e3"

if ! psql_admin --command \
    "SELECT public.ct193_e3_preparar_vector('e3_alta',1125,false)" \
    >"$temporal/psql.log" 2>&1; then
    printf 'CT193 E3: no se preparó e3\n' >&2
    exit 1
fi
emitir_aplicar e3_alta e3
probar_fase e3
if [[ ! -s $temporal/recibo-e3.json || ! -s $temporal/inicio-pg.json ]]; then
    printf 'CT193 E3: faltan recibo o instante de arranque previos al reinicio\n' >&2
    exit 1
fi
printf '[CT193:E3] recuperando PostgreSQL desde el mismo volumen efímero\n'
docker stop -t 15 "$contenedor" >/dev/null
for _ in {1..40}; do
    if ! docker container inspect "$contenedor" >/dev/null 2>&1; then break; fi
    sleep 0.25
done
if docker container inspect "$contenedor" >/dev/null 2>&1; then
    printf 'CT193 E3: el contenedor anterior no se retiró tras stop\n' >&2
    exit 1
fi
iniciar_contenedor
listo=0
for _ in {1..80}; do
    if docker exec "$contenedor" pg_isready -q -h /var/run/postgresql -U postgres 2>/dev/null; then
        listo=1
        break
    fi
    sleep 0.25
done
if (( ! listo )); then
    printf 'CT193 E3: PostgreSQL no volvió tras reinicio\n' >&2
    exit 1
fi
probar_fase e3_replay

for fase in abierto colision concurrente; do
    jornada=1125
    if [[ $fase == colision ]]; then jornada=1124; fi
    colision=false
    if [[ $fase == colision ]]; then colision=true; fi
    if ! psql_admin --command \
        "SELECT public.ct193_e3_preparar_vector('$fase',$jornada,$colision)" \
        >"$temporal/psql.log" 2>&1; then
        printf 'CT193 E3: no se preparó %s\n' "$fase" >&2
        exit 1
    fi
    emitir_aplicar "$fase" "$fase"
    probar_fase "$fase"
done
printf '[CT193:E3] E2 pre/post, E3 recuperada tras reinicio, abierto, colisión y concurrencia completas\n'
}

export -f ejecutar_ensayo_ct193
script_ct193="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)/$(basename -- "${BASH_SOURCE[0]}")"
timeout --signal=TERM --kill-after=60s 1740s \
    bash -Eeuo pipefail -c 'ejecutar_ensayo_ct193' "$script_ct193"
