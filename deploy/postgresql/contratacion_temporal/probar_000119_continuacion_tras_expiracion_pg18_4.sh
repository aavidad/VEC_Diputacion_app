#!/usr/bin/env bash
# CT119 en PostgreSQL 18.4 desechable: preimagen con dobles AD3, cadena CT
# real 000054-000064 y 000111 con sus pruebas (dejan una expiración
# confirmada), y después CT119: UP/DOWN/UP, doble aplicación rechazada,
# pruebas funcionales, negativas y ACL, reinicio y reversión denegada.
# Datos en /dev/shm/vec-pg-ct119-<pid> montado con -v (sin volúmenes anónimos).
set -Eeuo pipefail

if [[ ${VEC_CT119_BD_DESECHABLE:-} != SI ]]; then
    printf 'CT119 exige VEC_CT119_BD_DESECHABLE=SI\n' >&2
    exit 64
fi
command -v docker >/dev/null 2>&1 || { printf 'docker no disponible\n' >&2; exit 69; }

readonly imagen="${IMAGEN_POSTGRES:-postgres:18.4}"
readonly contenedor="vec-pg-ct119-$$"
readonly datos="/dev/shm/vec-pg-ct119-$$"
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

paso() { printf '[ct119:postgres] %s\n' "$1"; }

esperar() {
    local intento
    for intento in {1..60}; do
        if docker exec "$contenedor" pg_isready --quiet -U postgres -d postgres 2>/dev/null; then
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

paso 'preimagen, cadena CT real 000054-000064 y CT111 con sus datos'
psql_fichero contratacion_temporal/pruebas_sql/000111_preimagen_pg18.sql >/dev/null
for migracion in 000054 000056 000057 000058 000059 000060 000062 000063 000064 000111; do
    fichero="$(cd "$fuente/contratacion_temporal/migraciones" && ls "${migracion}"_*.up.sql)"
    psql_fichero "contratacion_temporal/migraciones/$fichero" >/dev/null
done
psql_fichero contratacion_temporal/pruebas_sql/000111_plazo_respuesta_llamamiento_pg18.sql >/dev/null

readonly up=contratacion_temporal/migraciones/000119_continuacion_tras_expiracion.up.sql
readonly down=contratacion_temporal/migraciones/000119_continuacion_tras_expiracion.down.sql
paso 'UP, DOWN y UP'
psql_fichero "$up" >/dev/null
psql_fichero "$down" >/dev/null
psql_fichero "$up" >/dev/null
esperar_fallo 'doble aplicación de CT119' 'CT119 ya instalada' psql_fichero "$up"

paso 'pruebas funcionales, negativas y ACL'
probar contratacion_temporal/pruebas_sql/000119_continuacion_tras_expiracion_pg18.sql

paso 'reinicio de PostgreSQL'
docker restart "$contenedor" >/dev/null
esperar
probar contratacion_temporal/pruebas_sql/000119_continuacion_tras_expiracion_reinicio_pg18.sql
esperar_fallo 'reversión con continuación tras expiración' 'reversión denegada' psql_fichero "$down"
paso 'CT119 verificada'
