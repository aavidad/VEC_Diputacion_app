#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-bolsa-accesos-pg18-$$"
base=postgres

limpiar() {
    docker rm -f "$contenedor" >/dev/null 2>&1 || true
    rm -f "/tmp/${contenedor}-a.out" "/tmp/${contenedor}-b.out"
}
trap limpiar EXIT INT TERM

docker run --rm -d --name "$contenedor" \
    -e POSTGRES_PASSWORD=solo-prueba-local-no-real \
    -p 127.0.0.1::5432 \
    -v "$raiz:/repo:ro" "$imagen" >/dev/null
for _ in $(seq 1 30); do
    if docker exec "$contenedor" pg_isready -U postgres -d "$base" \
        >/dev/null 2>&1; then
        break
    fi
    sleep 1
done
docker exec "$contenedor" pg_isready -U postgres -d "$base" >/dev/null
puerto=$(docker port "$contenedor" 5432/tcp | sed -n 's/.*://p' | head -n 1)
if [[ ! "$puerto" =~ ^[0-9]+$ ]]; then
    echo "no se pudo resolver el puerto efímero PG18 T13" >&2
    exit 1
fi

psql_archivo() {
    docker exec "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
        --username postgres --dbname "$base" --file "$1"
}

for archivo in \
    /repo/deploy/postgresql/autorizacion/roles_up.sql \
    /repo/deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql \
    /repo/deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql \
    /repo/deploy/postgresql/autorizacion/roles_v2_up.sql \
    /repo/deploy/postgresql/autorizacion/migraciones/000003_proyeccion_motivos_autorizacion_v2.up.sql \
    /repo/deploy/postgresql/autorizacion/migraciones/000004_registro_decisiones_solicitud_ligada_v2.up.sql \
    /repo/deploy/postgresql/bolsa_registro_accesos/roles_up.sql \
    /repo/deploy/postgresql/bolsa_registro_accesos/migraciones_autorizacion/000001_revalidacion_registro_accesos_v2.up.sql \
    /repo/deploy/postgresql/bolsa_registro_accesos/migraciones/000001_registro_accesos_t13.up.sql \
    /repo/deploy/postgresql/bolsa_registro_accesos/cerrar_acl_dba.sql
do
    psql_archivo "$archivo"
done

# El down de Autorización no puede retirar la frontera mientras T13 la usa.
if psql_archivo \
    /repo/deploy/postgresql/bolsa_registro_accesos/migraciones_autorizacion/000001_revalidacion_registro_accesos_v2.down.sql \
    >/dev/null 2>&1; then
    echo "down inverso de Autorización dejó T13 roto" >&2
    exit 1
fi
docker exec "$contenedor" psql -X --quiet --tuples-only \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    --command "SELECT to_regprocedure('vec_autorizacion.revalidar_decision_registro_accesos_bolsa_v2(jsonb,bytea,bytea,text,text,text,jsonb,text,text,text)') IS NOT NULL AND to_regprocedure('vec_bolsa_registro_accesos.consultar_accesos_administrativos_v1(jsonb)') IS NOT NULL" \
    | grep -q t

# Down limpio completo y reinstalación: nunca usa CASCADE y CREATE se cierra.
psql_archivo \
    /repo/deploy/postgresql/bolsa_registro_accesos/migraciones/000001_registro_accesos_t13.down.sql
psql_archivo \
    /repo/deploy/postgresql/bolsa_registro_accesos/migraciones_autorizacion/000001_revalidacion_registro_accesos_v2.down.sql
psql_archivo \
    /repo/deploy/postgresql/bolsa_registro_accesos/migraciones_autorizacion/000001_revalidacion_registro_accesos_v2.up.sql
docker exec --interactive "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" --command \
    "GRANT CREATE ON DATABASE $base TO vec_bolsa_accesos_propietario"
psql_archivo \
    /repo/deploy/postgresql/bolsa_registro_accesos/migraciones/000001_registro_accesos_t13.up.sql
psql_archivo /repo/deploy/postgresql/bolsa_registro_accesos/cerrar_acl_dba.sql

psql_archivo \
    /repo/deploy/postgresql/bolsa_registro_accesos/pruebas_sql/acl_y_falsificacion.sql
psql_archivo \
    /repo/deploy/postgresql/bolsa_registro_accesos/pruebas_sql/mecanica_registro.sql
psql_archivo \
    /repo/deploy/postgresql/bolsa_registro_accesos/pruebas_sql/doble_autorizacion_mecanica.sql
psql_archivo \
    /repo/deploy/postgresql/bolsa_registro_accesos/pruebas_sql/consulta_mecanica.sql

docker exec "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" --command \
    "ALTER ROLE vec_bolsa_accesos_consultor_prueba PASSWORD 'vec-prueba-t13-no-real-2026'"
(
    cd "$raiz"
    VEC_BOLSA_ACCESOS_PG18_DSN="postgres://vec_bolsa_accesos_consultor_prueba:vec-prueba-t13-no-real-2026@127.0.0.1:${puerto}/${base}?sslmode=disable" \
    VEC_BOLSA_ACCESOS_PG18_ADMIN_DSN="postgres://postgres:solo-prueba-local-no-real@127.0.0.1:${puerto}/${base}?sslmode=disable" \
    go test -race \
        ./internal/modules/bolsa/adapters/postgres/registroaccesos \
        -run '^TestIntegracionConsultarAccesosAdministrativosPG18$' \
        -count=1
)

instante=$(date -u '+%Y-%m-%dT%H:%M:%S.000000Z')
set +e
docker exec "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --set marca=6 --set instante="$instante" \
    --username postgres --dbname "$base" \
    --file /repo/deploy/postgresql/bolsa_registro_accesos/pruebas_sql/carrera_registro.sql \
    >"/tmp/${contenedor}-a.out" 2>&1 &
pid_a=$!
docker exec "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --set marca=7 --set instante="$instante" \
    --username postgres --dbname "$base" \
    --file /repo/deploy/postgresql/bolsa_registro_accesos/pruebas_sql/carrera_registro.sql \
    >"/tmp/${contenedor}-b.out" 2>&1 &
pid_b=$!
wait "$pid_a"; estado_a=$?
wait "$pid_b"; estado_b=$?
set -e
if [[ "$estado_a" -ne 0 && "$estado_b" -ne 0 ]]; then
    cat "/tmp/${contenedor}-a.out" "/tmp/${contenedor}-b.out" >&2
    echo "ambos contendientes T13 fallaron" >&2
    exit 1
fi

# Recuperación determinista: el ganador hace replay exacto y el abortado se
# incorpora; ambos payloads conservan su instante/correlación.
for marca in 6 7; do
    docker exec "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
        --set marca="$marca" --set instante="$instante" \
        --username postgres --dbname "$base" \
        --file /repo/deploy/postgresql/bolsa_registro_accesos/pruebas_sql/carrera_registro.sql \
        >/dev/null
done

docker exec --interactive "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
DO $verificar$
BEGIN
    IF (SELECT count(*)
          FROM vec_bolsa_registro_accesos.registro_acceso
         WHERE correlation_ref LIKE 'carrera-%') <> 2
       OR EXISTS (
           SELECT 1
             FROM (
                 SELECT secuencia, firma_anterior,
                        lag(firma) OVER (ORDER BY secuencia) AS anterior
                   FROM vec_bolsa_registro_accesos.registro_acceso
             ) AS cadena
            WHERE secuencia > 1 AND firma_anterior <> anterior
       ) THEN
        RAISE EXCEPTION 'carrera/continuidad T13 incorrecta';
    END IF;
END
$verificar$;
SQL

docker restart "$contenedor" >/dev/null
for _ in $(seq 1 30); do
    if docker exec "$contenedor" pg_isready -U postgres -d "$base" \
        >/dev/null 2>&1; then
        break
    fi
    sleep 1
done
docker exec "$contenedor" psql -X --quiet --tuples-only \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    --command "SELECT count(*) > 0 FROM vec_bolsa_registro_accesos.registro_acceso" \
    | grep -q t

# La historia durable bloquea una retirada destructiva.
if psql_archivo \
    /repo/deploy/postgresql/bolsa_registro_accesos/migraciones/000001_registro_accesos_t13.down.sql \
    >/dev/null 2>&1; then
    echo "down destructivo T13 aceptado con historia" >&2
    exit 1
fi

echo "PG18 T13: instalación, down limpio, ACL, falsificación, cadena, Go-pgx, vínculo-filtro, tipos, auditoría exacta, respuesta cerrada, commit, rollback, cancelación, consumo, carrera, recuperación y reinicio OK"
