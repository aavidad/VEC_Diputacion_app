#!/usr/bin/env bash
# CT111 en PostgreSQL 18.4 desechable: preimagen con dobles AD3, cadena CT
# real 000054-000064, UP/DOWN/UP, doble aplicación rechazada, pruebas
# funcionales y negativas, reinicio y reversión denegada con historia.
# Datos en /dev/shm/vec-pg-ct111-<pid> montado con -v (sin volúmenes anónimos).
set -Eeuo pipefail

if [[ ${VEC_CT111_BD_DESECHABLE:-} != SI ]]; then
    printf 'CT111 exige VEC_CT111_BD_DESECHABLE=SI\n' >&2
    exit 64
fi
command -v docker >/dev/null 2>&1 || { printf 'docker no disponible\n' >&2; exit 69; }

readonly imagen="${IMAGEN_POSTGRES:-postgres:18.4}"
readonly contenedor="vec-pg-ct111-$$"
readonly datos="/dev/shm/vec-pg-ct111-$$"
directorio="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd -P)"
readonly fuente="$(cd -- "$directorio/.." && pwd -P)"

limpiar() {
    docker rm --force "$contenedor" >/dev/null 2>&1 || true
    if [[ -d $datos ]]; then
        docker run --rm -v "$datos:/d" --entrypoint /bin/sh "$imagen" -c 'rm -rf /d/* /d/.[!.]*' >/dev/null 2>&1 || true
        rmdir "$datos" 2>/dev/null || true
    fi
}
trap limpiar EXIT INT TERM

paso() { printf '[ct111:postgres] %s\n' "$1"; }

esperar() {
    local intento
    for intento in {1..60}; do
        if docker exec "$contenedor" pg_isready --quiet -U postgres -d postgres 2>/dev/null; then
            # El entrypoint reinicia el servidor tras la inicialización.
            sleep 1
            docker exec "$contenedor" pg_isready --quiet -U postgres -d postgres && return 0
        fi
        sleep 1
    done
    docker logs "$contenedor" >&2 || true
    return 1
}

psql_fichero() {
    docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres -f "/pg/$1"
}

# Muestra los casos comprobados; ante un fallo, la salida completa.
probar() {
    local salida
    if ! salida="$(psql_fichero "$1" 2>&1)"; then
        printf '%s\n' "$salida" >&2
        return 1
    fi
    grep -E 'NOTICE' <<<"$salida" | sed -E 's/^.*NOTICE: +/  /'
}

esperar_fallo() {
    local descripcion="$1" patron="$2" salida
    shift 2
    if salida="$("$@" 2>&1)"; then
        printf 'se esperaba rechazo: %s\n%s\n' "$descripcion" "$salida" >&2
        return 1
    fi
    grep -Fq "$patron" <<<"$salida" || { printf 'rechazo distinto: %s\n%s\n' "$descripcion" "$salida" >&2; return 1; }
    paso "rechazo verificado: $descripcion"
}

mkdir "$datos"
docker run -d --name "$contenedor" -e POSTGRES_PASSWORD=solo-prueba-efimera-no-reutilizar \
    -v "$datos:/var/lib/postgresql" -v "$fuente:/pg:ro" "$imagen" >/dev/null
esperar
if [[ -n $(docker inspect --format '{{range .Mounts}}{{if eq .Type "volume"}}{{.Name}}{{end}}{{end}}' "$contenedor") ]]; then
    printf 'volumen anónimo inesperado\n' >&2
    exit 65
fi

paso 'preimagen y cadena CT real 000054-000064'
psql_fichero contratacion_temporal/pruebas_sql/000111_preimagen_pg18.sql >/dev/null
for migracion in 000054 000056 000057 000058 000059 000060 000062 000063 000064; do
    fichero="$(cd "$fuente/contratacion_temporal/migraciones" && ls "${migracion}"_*.up.sql)"
    psql_fichero "contratacion_temporal/migraciones/$fichero" >/dev/null
done

readonly up=contratacion_temporal/migraciones/000111_plazo_respuesta_llamamiento.up.sql
readonly down=contratacion_temporal/migraciones/000111_plazo_respuesta_llamamiento.down.sql
paso 'UP, DOWN y UP'
psql_fichero "$up" >/dev/null
psql_fichero "$down" >/dev/null
psql_fichero "$up" >/dev/null
esperar_fallo 'doble aplicación de CT111' 'CT111 ya instalada' psql_fichero "$up"

paso 'pruebas funcionales, negativas y ACL'
probar contratacion_temporal/pruebas_sql/000111_plazo_respuesta_llamamiento_pg18.sql

paso 'reinicio de PostgreSQL'
docker restart "$contenedor" >/dev/null
esperar
probar contratacion_temporal/pruebas_sql/000111_plazo_respuesta_llamamiento_reinicio_pg18.sql
esperar_fallo 'reversión con historia de plazo' 'reversión denegada' psql_fichero "$down"
paso 'CT111 verificada'
