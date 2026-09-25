#!/usr/bin/env bash
# Bolsa 000039 en PostgreSQL 18.4 desechable: cadena real 000001-000006 de
# llamamientos con dobles AD3, UP/DOWN/UP con definición restaurada exacta,
# doble aplicación rechazada, guardado real de orden, apertura, terminal
# «sin respuesta» y siguiente (registros generados por el servicio Go),
# negativas, ACL, reinicio y reversión denegada con historia.
# Datos en /dev/shm/vec-pg-b39-<pid> montado con -v (sin volúmenes anónimos).
set -Eeuo pipefail

if [[ ${VEC_BOLSA39_BD_DESECHABLE:-} != SI ]]; then
    printf 'Bolsa 000039 exige VEC_BOLSA39_BD_DESECHABLE=SI\n' >&2
    exit 64
fi
command -v docker >/dev/null 2>&1 || { printf 'docker no disponible\n' >&2; exit 69; }
command -v go >/dev/null 2>&1 || { printf 'go no disponible\n' >&2; exit 69; }

readonly imagen="${IMAGEN_POSTGRES:-postgres:18.4}"
readonly contenedor="vec-pg-b39-$$"
readonly datos="/dev/shm/vec-pg-b39-$$"
readonly fixtures="/dev/shm/vec-pg-b39-fixtures-$$"
directorio="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd -P)"
readonly fuente="$(cd -- "$directorio/.." && pwd -P)"
readonly raiz="$(cd -- "$directorio/../../.." && pwd -P)"

limpiar() {
    docker rm --force "$contenedor" >/dev/null 2>&1 || true
    if [[ -d $datos ]]; then
        docker run --rm -v "$datos:/d" --entrypoint /bin/sh "$imagen" -c 'rm -rf /d/* /d/.[!.]*' >/dev/null 2>&1 || true
        rmdir "$datos" 2>/dev/null || true
    fi
    rm -rf "$fixtures"
}
trap limpiar EXIT INT TERM

paso() { printf '[bolsa39:postgres] %s\n' "$1"; }
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
psql_orden() {
    docker exec -i "$contenedor" psql -X -q -At -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"
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
readonly definicion="SELECT md5(pg_get_functiondef('vec_bolsa_llamamientos.guardar_integracion_desarrollo_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure))||md5(string_agg(conname||pg_get_constraintdef(oid),',' ORDER BY conname)) FROM pg_constraint WHERE conrelid='vec_bolsa_llamamientos.integracion_desarrollo'::regclass"

mkdir "$datos" "$fixtures"
docker run -d --name "$contenedor" -e POSTGRES_PASSWORD=solo-prueba-efimera-no-reutilizar \
    -v "$datos:/var/lib/postgresql" -v "$fuente:/pg:ro" -v "$fixtures:/fixtures:ro" "$imagen" >/dev/null
esperar
if [[ -n $(docker inspect --format '{{range .Mounts}}{{if eq .Type "volume"}}{{.Name}}{{end}}{{end}}' "$contenedor") ]]; then
    printf 'volumen anónimo inesperado\n' >&2
    exit 65
fi

paso 'cadena real de autorización y llamamientos 000001-000006 con dobles AD3'
for fichero in autorizacion/roles_up.sql autorizacion/migraciones/000001_autorizacion.up.sql \
    ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql \
    bolsa_llamamientos/roles_up.sql bolsa_llamamientos/migraciones_autorizacion/000001_revalidacion_llamamientos.up.sql \
    bolsa_llamamientos/migraciones/000001_almacen_llamamientos.up.sql bolsa_llamamientos/migraciones/000002_guardado_cerrado.up.sql \
    bolsa_llamamientos/pruebas_sql/expiracion_rrhh/dobles_ad3.sql \
    bolsa_llamamientos/migraciones/000003_integracion_desarrollo.up.sql \
    bolsa_llamamientos/migraciones/000004_aceptacion_rrhh_integracion_desarrollo.up.sql \
    bolsa_llamamientos/migraciones/000005_renuncia_rrhh_integracion_desarrollo.up.sql \
    bolsa_llamamientos/migraciones/000006_continuacion_llamamiento_desarrollo.up.sql; do
    psql_fichero "$fichero" >/dev/null 2>&1 || { psql_fichero "$fichero"; exit 1; }
done
psql_orden "CREATE ROLE vec_b39_runtime LOGIN; GRANT vec_bolsa_llamamientos_ejecutor TO vec_b39_runtime; CREATE ROLE vec_b39_ajeno LOGIN; GRANT TEMPORARY ON DATABASE postgres TO vec_b39_runtime;" >/dev/null

readonly up=bolsa_llamamientos/migraciones/000039_expiracion_rrhh_integracion_desarrollo.up.sql
readonly down=bolsa_llamamientos/migraciones/000039_expiracion_rrhh_integracion_desarrollo.down.sql
paso 'UP, DOWN (definición y restricciones exactas) y UP'
antes="$(psql_orden "$definicion")"
psql_fichero "$up" >/dev/null
instalada="$(psql_orden "$definicion")"
psql_fichero "$down" >/dev/null
[[ "$(psql_orden "$definicion")" == "$antes" ]] || { printf 'DOWN no restauró la definición exacta\n' >&2; exit 1; }
psql_fichero "$up" >/dev/null
[[ "$(psql_orden "$definicion")" == "$instalada" ]] || { printf 'segunda subida distinta\n' >&2; exit 1; }
esperar_fallo 'doble aplicación de Bolsa 000039' 'ya instalada' psql_fichero "$up"

paso 'registros canónicos generados por el servicio Go de integración'
(cd "$raiz" && VEC_BOLSA39_FIXTURES="$fixtures" go test -count=1 -run '^TestGenerarFixturesBolsa39$' ./internal/modules/bolsa/application/ >/dev/null)
[[ -s "$fixtures/registros.json" ]] || { printf 'no se generaron registros\n' >&2; exit 1; }

paso 'guardado real, negativas y ACL (espera ~40 s a los instantes de cada efecto)'
probar bolsa_llamamientos/pruebas_sql/expiracion_rrhh/pruebas.sql

paso 'reinicio de PostgreSQL'
docker restart "$contenedor" >/dev/null
esperar
probar bolsa_llamamientos/pruebas_sql/expiracion_rrhh/reinicio.sql
esperar_fallo 'reversión con historia de expiración' 'reversión denegada' psql_fichero "$down"
paso 'Bolsa 000039 verificada'
