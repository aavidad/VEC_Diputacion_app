#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-bolsa-publica-pg-prueba-${USER:-usuario}-$$"
base=vec_bolsa_publica_prueba
ancla_inicial=33929a8b6abbab2c1b57aac36a3070f475610f59fdd064a0b0a3e7f760cf8807
clave_admin="VecAdminPrueba${BASHPID}${RANDOM}"
clave_lector="VecLectorPrueba${BASHPID}${RANDOM}"
clave_publicador="VecPublicadorPrueba${BASHPID}${RANDOM}"
directorio_tls=$(mktemp -d /tmp/vec-bolsa-publica-tls-XXXXXX)
cache_go=$(mktemp -d /tmp/vec-bolsa-publica-go-XXXXXX)

limpiar() {
    docker rm -f "$contenedor" >/dev/null 2>&1 || true
    rm -rf "$directorio_tls" "$cache_go"
}
trap limpiar EXIT INT TERM

openssl req -x509 -newkey rsa:3072 -sha256 -nodes -days 1 \
    -subj '/CN=CA integracion VEC Bolsa publica' \
    -keyout "$directorio_tls/ca.key" -out "$directorio_tls/ca.crt" >/dev/null 2>&1
cat >"$directorio_tls/servidor.ext" <<'EXT'
subjectAltName=DNS:localhost,IP:127.0.0.1
extendedKeyUsage=serverAuth
keyUsage=digitalSignature,keyEncipherment
EXT
openssl req -newkey rsa:3072 -nodes -sha256 -subj '/CN=localhost' \
    -keyout "$directorio_tls/servidor.key" -out "$directorio_tls/servidor.csr" >/dev/null 2>&1
openssl x509 -req -sha256 -days 1 -in "$directorio_tls/servidor.csr" \
    -CA "$directorio_tls/ca.crt" -CAkey "$directorio_tls/ca.key" -CAcreateserial \
    -extfile "$directorio_tls/servidor.ext" -out "$directorio_tls/servidor.crt" >/dev/null 2>&1

docker run --detach --rm \
    --name "$contenedor" \
    --publish 127.0.0.1::5432 \
    --env POSTGRES_DB="$base" \
    --env POSTGRES_PASSWORD="$clave_admin" \
    "$imagen" >/dev/null

for _ in $(seq 1 60); do
    if docker exec "$contenedor" pg_isready --username postgres --dbname "$base" >/dev/null 2>&1; then
        break
    fi
    sleep 1
done
docker exec "$contenedor" pg_isready --username postgres --dbname "$base" >/dev/null

fichero_configuracion=$(docker exec "$contenedor" psql -X --tuples-only --no-align \
    --username postgres --dbname "$base" --command 'SHOW config_file')
if [[ -z "$fichero_configuracion" ]]; then
    echo "no se pudo localizar postgresql.conf" >&2
    exit 1
fi
docker cp "$directorio_tls/servidor.crt" "$contenedor:/tmp/vec-bolsa-publica-server.crt"
docker cp "$directorio_tls/servidor.key" "$contenedor:/tmp/vec-bolsa-publica-server.key"
docker cp "$directorio_tls/ca.crt" "$contenedor:/tmp/vec-bolsa-publica-ca.crt"
docker exec --user root --env VEC_CONFIGURACION_POSTGRESQL="$fichero_configuracion" "$contenedor" sh -ceu '
    chown postgres:postgres /tmp/vec-bolsa-publica-server.crt /tmp/vec-bolsa-publica-server.key /tmp/vec-bolsa-publica-ca.crt
    chmod 600 /tmp/vec-bolsa-publica-server.key
    chmod 644 /tmp/vec-bolsa-publica-server.crt /tmp/vec-bolsa-publica-ca.crt
    printf "%s\n" \
      "ssl = on" \
      "ssl_cert_file = '\''/tmp/vec-bolsa-publica-server.crt'\''" \
      "ssl_key_file = '\''/tmp/vec-bolsa-publica-server.key'\''" \
      "ssl_ca_file = '\''/tmp/vec-bolsa-publica-ca.crt'\''" \
      "ssl_min_protocol_version = '\''TLSv1.2'\''" \
      >> "$VEC_CONFIGURACION_POSTGRESQL"
'
docker restart "$contenedor" >/dev/null
for _ in $(seq 1 60); do
    if docker exec "$contenedor" pg_isready --username postgres --dbname "$base" >/dev/null 2>&1; then
        break
    fi
    sleep 1
done
docker exec "$contenedor" pg_isready --username postgres --dbname "$base" >/dev/null

# La base por defecto conserva CONNECT/TEMPORARY para PUBLIC. El bootstrap no
# debe corregir esa ACL global de forma oportunista ni dejar roles parciales.
if docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    <"$raiz/deploy/postgresql/bolsa_publica/roles_up.sql" >/dev/null 2>&1; then
    echo "roles_up acepto una base sin endurecer" >&2
    exit 1
fi
roles_parciales=$(docker exec "$contenedor" psql -X --tuples-only --no-align \
    --username postgres --dbname "$base" --command "
SELECT count(*) FROM pg_catalog.pg_roles
 WHERE rolname LIKE 'vec_bolsa_publica_%'")
if [[ "$roles_parciales" != "0" ]]; then
    echo "roles_up dejo roles tras rechazar la ACL de base" >&2
    exit 1
fi
esquemas_parciales=$(docker exec "$contenedor" psql -X --tuples-only --no-align \
    --username postgres --dbname "$base" --command "
SELECT count(*) FROM pg_catalog.pg_namespace
 WHERE nspname IN ('vec_bolsa_publica_datos','vec_bolsa_publica_lectura')")
if [[ "$esquemas_parciales" != "0" ]]; then
    echo "roles_up dejo esquemas tras rechazar la ACL de base" >&2
    exit 1
fi

docker exec "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" --command \
    "REVOKE CONNECT, TEMPORARY ON DATABASE ${base} FROM PUBLIC"

# CREATE en la base o en el schema public tampoco puede pasar desapercibido.
docker exec "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" --command \
    "GRANT CREATE ON DATABASE ${base} TO PUBLIC"
if docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    <"$raiz/deploy/postgresql/bolsa_publica/roles_up.sql" >/dev/null 2>&1; then
    echo "roles_up acepto CREATE de PUBLIC sobre la base" >&2
    exit 1
fi
docker exec "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" --command \
    "REVOKE CREATE ON DATABASE ${base} FROM PUBLIC; GRANT CREATE ON SCHEMA public TO PUBLIC"
if docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    <"$raiz/deploy/postgresql/bolsa_publica/roles_up.sql" >/dev/null 2>&1; then
    echo "roles_up acepto CREATE de PUBLIC sobre el schema public" >&2
    exit 1
fi
docker exec "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" --command \
    "REVOKE CREATE ON SCHEMA public FROM PUBLIC;
     REVOKE ALL PRIVILEGES ON DATABASE postgres, template1 FROM PUBLIC"

roles_parciales=$(docker exec "$contenedor" psql -X --tuples-only --no-align \
    --username postgres --dbname "$base" --command \
    "SELECT count(*) FROM pg_catalog.pg_roles WHERE rolname LIKE 'vec_bolsa_publica_%'")
if [[ "$roles_parciales" != "0" ]]; then
    echo "roles_up dejo roles tras rechazar CREATE de PUBLIC" >&2
    exit 1
fi

docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    <"$raiz/deploy/postgresql/bolsa_publica/roles_up.sql"

{
    printf '%s\n' 'SET ROLE vec_bolsa_publica_migrador;'
    cat "$raiz/deploy/postgresql/bolsa_publica/migraciones/000001_proyeccion_publica.up.sql"
} | docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base"

# Ningun rol tecnico recibe CREATE/TEMPORARY sobre la base. CONNECT es el
# unico privilegio de base y el LOGIN de aplicacion solo hereda el rol lector.
acl_base=$(docker exec "$contenedor" psql -X --tuples-only --no-align \
    --username postgres --dbname "$base" --command "
SELECT string_agg(
           rol || ':' || puede_conectar::text || ':' || puede_crear::text || ':' || puede_temporal::text,
           ',' ORDER BY rol
       )
  FROM (
      SELECT rol,
             has_database_privilege(rol, current_database(), 'CONNECT') AS puede_conectar,
             has_database_privilege(rol, current_database(), 'CREATE') AS puede_crear,
             has_database_privilege(rol, current_database(), 'TEMPORARY') AS puede_temporal
        FROM unnest(ARRAY[
	            'vec_bolsa_publica_consulta',
	            'vec_bolsa_publica_migrador',
	            'vec_bolsa_publica_propietario',
	            'vec_bolsa_publica_publicacion_propietario',
	            'vec_bolsa_publica_publicador'
	        ]) AS rol
	  ) AS permisos")
acl_base_esperada='vec_bolsa_publica_consulta:true:false:false,vec_bolsa_publica_migrador:true:false:false,vec_bolsa_publica_propietario:true:false:false,vec_bolsa_publica_publicacion_propietario:false:false:false,vec_bolsa_publica_publicador:true:false:false'
if [[ "$acl_base" != "$acl_base_esperada" ]]; then
    echo "ACL de base inesperada para los roles tecnicos: $acl_base" >&2
    exit 1
fi

docker exec --interactive \
    --env VEC_CLAVE_LECTOR="$clave_lector" \
    --env VEC_CLAVE_PUBLICADOR="$clave_publicador" \
    "$contenedor" \
	    psql -X --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
\getenv clave_lector VEC_CLAVE_LECTOR
\getenv clave_publicador VEC_CLAVE_PUBLICADOR
CREATE ROLE vec_bolsa_publica_integracion_login LOGIN NOSUPERUSER NOCREATEDB
	    NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS PASSWORD :'clave_lector';
GRANT vec_bolsa_publica_consulta TO vec_bolsa_publica_integracion_login
	    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
ALTER ROLE vec_bolsa_publica_publicador_login PASSWORD :'clave_publicador';
ALTER DATABASE vec_bolsa_publica_prueba SET log_parameter_max_length_on_error = 4096;
SET application_name = 'vec-bolsa-publicador';
SET search_path = 'pg_catalog,pg_temp';
SET statement_timeout = '60s';
SET lock_timeout = '5s';
SET idle_in_transaction_session_timeout = '5s';
SET transaction_timeout = '2min';
SET log_parameter_max_length_on_error = 0;

CREATE TABLE public.proyeccion_publica_prueba(payload jsonb NOT NULL);
REVOKE ALL ON public.proyeccion_publica_prueba FROM PUBLIC;
INSERT INTO public.proyeccion_publica_prueba(payload) VALUES (
$proyeccion$
{
  "fuente":{"revision":"revision-001","actualizada_en":"2026-07-22T10:00:00Z"},
  "catalogos":[
    {"referencia":"tipos_convocatoria","version":1,"entradas":[{"clave":"bolsa_temporal","etiqueta":"Bolsa temporal","descripcion":"Proceso temporal.","semantica":"informacion","orden":1}]},
    {"referencia":"estados_convocatoria","version":1,"entradas":[{"clave":"inscripcion","etiqueta":"Inscripción","descripcion":"Plazo de inscripción.","semantica":"exito","orden":1},{"clave":"cerrada","etiqueta":"Cerrada","descripcion":"Proceso cerrado.","semantica":"neutro","orden":2}]},
    {"referencia":"tipos_plazo","version":1,"entradas":[{"clave":"inscripcion","etiqueta":"Inscripción","descripcion":"Presentación de solicitudes.","semantica":"informacion","orden":1}]},
    {"referencia":"tipos_documento","version":1,"entradas":[{"clave":"bases","etiqueta":"Bases","descripcion":"Bases reguladoras.","semantica":"documento","orden":1}]},
    {"referencia":"categorias_ayuda","version":1,"entradas":[{"clave":"general","etiqueta":"General","descripcion":"Información general.","semantica":"informacion","orden":1}]}
  ],
  "categorias":{
    "actual":{"catalogo_id":"categorias-profesionales","catalogo_version":2,"catalogo_huella_sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","catalogo_huella_proyeccion_sha256":"b661b37ca7323fa168734899038f8fa99cb77ff07d114e4a7d787d62b5d36593"},
    "snapshots":[
      {"catalogo_id":"categorias-profesionales","version":1,
       "huella_gobernada_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
       "huella_proyeccion_publica_sha256":"d3118b2252d7dac326cf575a66937c6947e1439e0cfe74b23ff599ba80f2d594",
       "categorias":[{"clave":"auxiliar-administrativo","etiqueta":"Auxiliar administrativo (histórica)","descripcion":"Categoría profesional histórica de auxiliar administrativo.","semantica":"informacion","orden":1,"area":"administracion","area_etiqueta":"Administración","suscribible":false,"vigente_desde":"2020-01-01T00:00:00Z","vigente_hasta":"2025-01-01T00:00:00Z"}]},
      {"catalogo_id":"categorias-profesionales","version":2,
       "huella_gobernada_sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
       "huella_proyeccion_publica_sha256":"b661b37ca7323fa168734899038f8fa99cb77ff07d114e4a7d787d62b5d36593",
       "categorias":[{"clave":"auxiliar-administrativo","etiqueta":"Auxiliar administrativo","descripcion":"Categoría profesional de auxiliar administrativo.","semantica":"informacion","orden":1,"area":"administracion","area_etiqueta":"Administración","suscribible":true,"vigente_desde":"2025-01-01T00:00:00Z","vigente_hasta":null}]}
    ]
  },
  "convocatorias":[{
    "identificador_publico":"auxiliares-2024","version_publica":"v1","estado":"cerrada","tipo":"bolsa_temporal",
    "catalogo_categorias":{"catalogo_id":"categorias-profesionales","catalogo_version":1,"catalogo_huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","catalogo_huella_proyeccion_sha256":"d3118b2252d7dac326cf575a66937c6947e1439e0cfe74b23ff599ba80f2d594"},
    "huella_publica_sha256":"47eabd5ff97ffb18f416016a6c0482baa93abcac5ecf87ac9e547e3eb5ac7376",
    "huella_resumen_publico_sha256":"3ebdf22211e8ed768f5eb294836dd61a5abc4075d4bb0952d524ead7abe861f7",
    "titulo":"Bolsa histórica de auxiliares administrativos","resumen":"Convocatoria histórica de auxiliares administrativos.",
    "descripcion":"Proceso histórico cerrado con su categoría versionada.","publicada_en":"2024-01-10T09:00:00Z","actualizada_en":"2024-12-31T09:00:00Z",
    "categorias":["auxiliar-administrativo"],"plazos":[],"requisitos":[],"documentos":[],"ayuda":[]
  },{
    "identificador_publico":"auxiliares-2026","version_publica":"v1","estado":"inscripcion","tipo":"bolsa_temporal",
    "catalogo_categorias":{"catalogo_id":"categorias-profesionales","catalogo_version":2,"catalogo_huella_sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","catalogo_huella_proyeccion_sha256":"b661b37ca7323fa168734899038f8fa99cb77ff07d114e4a7d787d62b5d36593"},
    "huella_publica_sha256":"87d3955bdf4ad4b6488b33727c679aebab1980c0060c088ba23ec21d6de981dd",
    "huella_resumen_publico_sha256":"0fbd6e6d48a178c1b41c125feeb9bdb2bd847a7acf295b06fc8827ebbdda97fd",
    "titulo":"Bolsa temporal de auxiliares administrativos","resumen":"Convocatoria pública para auxiliares administrativos.",
    "descripcion":"Proceso selectivo sujeto a bases firmadas y publicadas.","publicada_en":"2026-07-01T09:00:00Z","actualizada_en":"2026-07-01T09:00:00Z",
    "categorias":["auxiliar-administrativo"],
    "plazos":[{"referencia":"plazo:inscripcion","tipo":"inscripcion","titulo":"Inscripción","descripcion":"Plazo de presentación de solicitudes.","abre_en":"2026-07-01T00:00:00Z","cierra_en":"2026-07-31T23:59:59Z"}],
    "requisitos":[{"referencia":"requisito:edad","orden":1,"titulo":"Edad","descripcion":"Cumplir la edad exigida.","obligatorio":true}],
    "documentos":[{"referencia":"documento:bases","tipo":"bases","orden":1,"titulo":"Bases reguladoras","descripcion":"Bases oficiales de la convocatoria.","formato":"pdf","url":"/bolsa/documentos/bases-auxiliares-2026.pdf","publicado_en":"2026-07-01T09:00:00Z"}],
    "ayuda":[{"referencia":"ayuda:inscripcion","categoria":"general","orden":1,"pregunta":"¿Cómo presento la solicitud?","respuesta":"Acceda al área personal durante el plazo."}]
  }]
}
$proyeccion$::jsonb
);
\o /dev/null
SELECT pg_catalog.set_config(
    'vec.prueba_proyeccion_publica',
    (SELECT payload::text FROM public.proyeccion_publica_prueba),
    false
);
\o

SET SESSION AUTHORIZATION vec_bolsa_publica_publicador_login;
SELECT vec_bolsa_publica_publicacion.publicar_proyeccion_v2(
    current_setting('vec.prueba_proyeccion_publica')::jsonb,
'33929a8b6abbab2c1b57aac36a3070f475610f59fdd064a0b0a3e7f760cf8807'
);
RESET SESSION AUTHORIZATION;

-- El historial es monotono incluso después de A→B o de invalidar A a cero.
-- Cada escenario se revierte para que los tests Go sigan arrancando sobre A.
BEGIN;
SET SESSION AUTHORIZATION vec_bolsa_publica_publicador_login;
SELECT vec_bolsa_publica_publicacion.publicar_proyeccion_v2(
    jsonb_set(
        current_setting('vec.prueba_proyeccion_publica')::jsonb,
        '{fuente,revision}', '"revision-002"'::jsonb
    ),
    repeat('b', 64)
);
DO $rechazar_reutilizacion$
BEGIN
    PERFORM vec_bolsa_publica_publicacion.publicar_proyeccion_v2(
        current_setting('vec.prueba_proyeccion_publica')::jsonb,
        '33929a8b6abbab2c1b57aac36a3070f475610f59fdd064a0b0a3e7f760cf8807'
    );
    RAISE EXCEPTION USING ERRCODE = 'P0001',
        MESSAGE = 'el historial permitio reutilizar A despues de B';
EXCEPTION WHEN invalid_parameter_value THEN
    NULL;
END
$rechazar_reutilizacion$;
RESET SESSION AUTHORIZATION;
DO $comprobar_b$
BEGIN
    IF (SELECT revision FROM vec_bolsa_publica_datos.fuente) <> 'revision-002'
       OR (SELECT count(*) FROM vec_bolsa_publica_datos.manifiesto_consumido) <> 2 THEN
        RAISE EXCEPTION 'el rechazo A despues de B altero la proyeccion vigente';
    END IF;
END
$comprobar_b$;
ROLLBACK;

BEGIN;
SET ROLE vec_bolsa_publica_propietario;
UPDATE vec_bolsa_publica_datos.categoria_publica
   SET etiqueta = 'Etiqueta alterada que no puede publicarse';
RESET ROLE;
SET SESSION AUTHORIZATION vec_bolsa_publica_publicador_login;
DO $rechazar_despues_de_cero$
BEGIN
    PERFORM vec_bolsa_publica_publicacion.publicar_proyeccion_v2(
        current_setting('vec.prueba_proyeccion_publica')::jsonb,
        '33929a8b6abbab2c1b57aac36a3070f475610f59fdd064a0b0a3e7f760cf8807'
    );
    RAISE EXCEPTION USING ERRCODE = 'P0001',
        MESSAGE = 'el historial permitio reutilizar A despues de invalidarla';
EXCEPTION WHEN invalid_parameter_value THEN
    NULL;
END
$rechazar_despues_de_cero$;
RESET SESSION AUTHORIZATION;
DO $comprobar_cero$
BEGIN
    IF (SELECT manifiesto_sha256 FROM vec_bolsa_publica_datos.fuente) <> repeat('0', 64)
       OR (SELECT count(*) FROM vec_bolsa_publica_datos.manifiesto_consumido) <> 1 THEN
        RAISE EXCEPTION 'el rechazo A despues de cero altero controles';
    END IF;
END
$comprobar_cero$;
ROLLBACK;

-- Todas las allowlists anidadas rechazan una clave personal desconocida. Las
-- excepciones se capturan y sus mensajes no incluyen el valor rechazado.
SET SESSION AUTHORIZATION vec_bolsa_publica_publicador_login;
DO $rechazar_campos_desconocidos$
DECLARE
    base jsonb;
    exceso_entradas jsonb;
    mutacion jsonb;
BEGIN
    base := current_setting('vec.prueba_proyeccion_publica')::jsonb;
    SELECT jsonb_set(
               base,
               '{catalogos}',
               jsonb_agg(jsonb_build_object(
                   'referencia', 'catalogo_' || numero_catalogo,
                   'version', 1,
                   'entradas', (
                       SELECT jsonb_agg(jsonb_build_object(
                           'clave', 'clave_' || numero_entrada,
                           'etiqueta', 'Entrada ' || numero_entrada,
                           'descripcion', '',
                           'semantica', 'informacion',
                           'orden', numero_entrada
                       ))
                         FROM generate_series(1, 205) AS numero_entrada
                   )
               ))
           )
      INTO exceso_entradas
      FROM generate_series(1, 5) AS numero_catalogo;
    FOREACH mutacion IN ARRAY ARRAY[
        base || '{"dni":"00000000T"}'::jsonb,
        jsonb_set(base, '{fuente}', base#>'{fuente}' || '{"dni":"00000000T"}'::jsonb),
        jsonb_set(base, '{catalogos,0}', base#>'{catalogos,0}' || '{"dni":"00000000T"}'::jsonb),
        jsonb_set(base, '{catalogos,0,entradas,0}', base#>'{catalogos,0,entradas,0}' || '{"dni":"00000000T"}'::jsonb),
        jsonb_set(base, '{categorias}', base#>'{categorias}' || '{"dni":"00000000T"}'::jsonb),
        jsonb_set(base, '{categorias,actual}', base#>'{categorias,actual}' || '{"dni":"00000000T"}'::jsonb),
        jsonb_set(base, '{categorias,snapshots,0}', base#>'{categorias,snapshots,0}' || '{"dni":"00000000T"}'::jsonb),
        jsonb_set(base, '{categorias,snapshots,0,categorias,0}', base#>'{categorias,snapshots,0,categorias,0}' || '{"dni":"00000000T"}'::jsonb),
        jsonb_set(base, '{convocatorias,0}', base#>'{convocatorias,0}' || '{"dni":"00000000T"}'::jsonb),
        jsonb_set(base, '{convocatorias,0,catalogo_categorias}', base#>'{convocatorias,0,catalogo_categorias}' || '{"dni":"00000000T"}'::jsonb),
        jsonb_set(base, '{convocatorias,0,categorias,0}', '{"dni":"00000000T"}'::jsonb),
        jsonb_set(base, '{convocatorias,1,plazos,0}', base#>'{convocatorias,1,plazos,0}' || '{"dni":"00000000T"}'::jsonb),
        jsonb_set(base, '{convocatorias,1,requisitos,0}', base#>'{convocatorias,1,requisitos,0}' || '{"dni":"00000000T"}'::jsonb),
        jsonb_set(base, '{convocatorias,1,documentos,0}', base#>'{convocatorias,1,documentos,0}' || '{"dni":"00000000T"}'::jsonb),
        jsonb_set(base, '{convocatorias,1,ayuda,0}', base#>'{convocatorias,1,ayuda,0}' || '{"dni":"00000000T"}'::jsonb),
        jsonb_set(base, '{catalogos,0,version}', '1.5'::jsonb),
        jsonb_set(base, '{categorias,snapshots,0,categorias,0,orden}', '2147483648'::jsonb),
        jsonb_set(base, '{convocatorias,1,requisitos,0,orden}', '1.5'::jsonb),
        jsonb_set(
            base, '{categorias,snapshots,0,huella_proyeccion_publica_sha256}',
            to_jsonb(repeat('e', 64))
        ),
        exceso_entradas
    ] LOOP
        BEGIN
            PERFORM vec_bolsa_publica_publicacion.publicar_proyeccion_v2(
                mutacion, repeat('c', 64)
            );
            RAISE EXCEPTION USING ERRCODE = 'P0001',
                MESSAGE = 'la allowlist recursiva acepto un campo desconocido';
        EXCEPTION WHEN invalid_parameter_value THEN
            NULL;
        END;
    END LOOP;
    BEGIN
        PERFORM vec_bolsa_publica_publicacion.publicar_proyeccion_v2(
            base, repeat('0', 64)
        );
        RAISE EXCEPTION USING ERRCODE = 'P0001',
            MESSAGE = 'el publicador acepto el sentinel cero';
    EXCEPTION WHEN invalid_parameter_value THEN
        NULL;
    END;
END
$rechazar_campos_desconocidos$;
RESET SESSION AUTHORIZATION;
DO $comprobar_inmutabilidad$
BEGIN
    IF (SELECT count(*) FROM vec_bolsa_publica_datos.manifiesto_consumido) <> 1
       OR (SELECT revision FROM vec_bolsa_publica_datos.fuente) <> 'revision-001'
       OR EXISTS (
           SELECT 1 FROM vec_bolsa_publica_datos.categoria_publica
            WHERE etiqueta = 'Etiqueta alterada que no puede publicarse'
       ) THEN
        RAISE EXCEPTION 'una publicacion rechazada dejo estado parcial';
    END IF;
END
$comprobar_inmutabilidad$;
SQL

propietarios=$(docker exec "$contenedor" psql -X --tuples-only --no-align \
    --username postgres --dbname "$base" --command "
SELECT
    (SELECT count(*) FROM pg_catalog.pg_namespace
      WHERE (nspname IN ('vec_bolsa_publica_datos','vec_bolsa_publica_lectura')
             AND pg_get_userbyid(nspowner) = 'vec_bolsa_publica_propietario')
         OR (nspname = 'vec_bolsa_publica_publicacion'
             AND pg_get_userbyid(nspowner) = 'vec_bolsa_publica_publicacion_propietario'))
 || ':' ||
    (SELECT count(*) FROM pg_catalog.pg_class AS objeto
      JOIN pg_catalog.pg_namespace AS esquema ON esquema.oid = objeto.relnamespace
     WHERE esquema.nspname = 'vec_bolsa_publica_datos' AND objeto.relkind = 'r'
       AND pg_get_userbyid(objeto.relowner) = 'vec_bolsa_publica_propietario')
 || ':' ||
    (SELECT count(*) FROM pg_catalog.pg_class AS objeto
      JOIN pg_catalog.pg_namespace AS esquema ON esquema.oid = objeto.relnamespace
     WHERE esquema.nspname = 'vec_bolsa_publica_lectura' AND objeto.relkind = 'v'
       AND pg_get_userbyid(objeto.relowner) = 'vec_bolsa_publica_propietario')
 || ':' ||
    (SELECT count(*) FROM pg_catalog.pg_proc AS objeto
      JOIN pg_catalog.pg_namespace AS esquema ON esquema.oid = objeto.pronamespace
     WHERE esquema.nspname = 'vec_bolsa_publica_datos'
       AND pg_get_userbyid(objeto.proowner) = 'vec_bolsa_publica_propietario')
 || ':' ||
    (SELECT count(*) FROM pg_catalog.pg_proc AS objeto
      JOIN pg_catalog.pg_namespace AS esquema ON esquema.oid = objeto.pronamespace
     WHERE esquema.nspname = 'vec_bolsa_publica_publicacion'
       AND pg_get_userbyid(objeto.proowner) = 'vec_bolsa_publica_publicacion_propietario')")
if [[ "$propietarios" != "3:12:10:1:2" ]]; then
    echo "propietarios fisicos/SECURITY DEFINER inesperados: $propietarios" >&2
    exit 1
fi

membresias=$(docker exec "$contenedor" psql -X --tuples-only --no-align \
    --username postgres --dbname "$base" --command "
SELECT string_agg(
           grupo.rolname || '>' || miembro.rolname || ':' ||
           membresia.admin_option::text || ':' || membresia.inherit_option::text || ':' ||
           membresia.set_option::text,
           ',' ORDER BY grupo.rolname, miembro.rolname
       )
  FROM pg_catalog.pg_auth_members AS membresia
  JOIN pg_catalog.pg_roles AS grupo ON grupo.oid = membresia.roleid
  JOIN pg_catalog.pg_roles AS miembro ON miembro.oid = membresia.member
 WHERE grupo.rolname LIKE 'vec_bolsa_publica_%'")
membresias_esperadas='vec_bolsa_publica_consulta>vec_bolsa_publica_integracion_login:false:true:false,vec_bolsa_publica_propietario>vec_bolsa_publica_migrador:false:false:true,vec_bolsa_publica_publicacion_propietario>vec_bolsa_publica_migrador:false:false:true,vec_bolsa_publica_publicador>vec_bolsa_publica_publicador_login:false:true:false'
if [[ "$membresias" != "$membresias_esperadas" ]]; then
    echo "membresias tecnicas inesperadas: $membresias" >&2
    exit 1
fi

acl_funcion=$(docker exec "$contenedor" psql -X --tuples-only --no-align \
    --username postgres --dbname "$base" --command "
WITH tablas AS (
    SELECT objeto.oid
      FROM pg_catalog.pg_class AS objeto
      JOIN pg_catalog.pg_namespace AS esquema ON esquema.oid = objeto.relnamespace
     WHERE esquema.nspname = 'vec_bolsa_publica_datos' AND objeto.relkind = 'r'
)
SELECT
    (SELECT count(*) FROM tablas WHERE has_table_privilege(
        'vec_bolsa_publica_publicacion_propietario', oid, 'INSERT'))
 || ':' ||
    (SELECT count(*) FROM tablas WHERE has_table_privilege(
        'vec_bolsa_publica_publicacion_propietario', oid, 'DELETE'))
 || ':' ||
    (SELECT count(*) FROM tablas WHERE has_table_privilege(
        'vec_bolsa_publica_publicacion_propietario', oid, 'SELECT,UPDATE,TRUNCATE,REFERENCES,TRIGGER'))
 || ':' ||
    (SELECT count(*) FROM tablas WHERE has_table_privilege(
        'vec_bolsa_publica_publicador', oid, 'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'))
 || ':' ||
    has_schema_privilege('vec_bolsa_publica_publicador', 'vec_bolsa_publica_publicacion', 'USAGE')::text
 || ':' ||
    has_schema_privilege('vec_bolsa_publica_publicador', 'vec_bolsa_publica_datos', 'USAGE,CREATE')::text
 || ':' ||
    has_function_privilege(
        'vec_bolsa_publica_publicador',
        'vec_bolsa_publica_publicacion.publicar_proyeccion_v2(jsonb,text)', 'EXECUTE'
    )::text
 || ':' ||
    has_function_privilege(
        'vec_bolsa_publica_publicador',
        'vec_bolsa_publica_publicacion.objeto_jsonb_exacto_v2(jsonb,text[])', 'EXECUTE'
    )::text
 || ':' ||
    has_function_privilege(
        'vec_bolsa_publica_publicador_login',
        'vec_bolsa_publica_publicacion.publicar_proyeccion_v2(jsonb,text)', 'EXECUTE'
    )::text
 || ':' ||
    has_table_privilege(
        'vec_bolsa_publica_publicador_login',
        'public.proyeccion_publica_prueba', 'SELECT'
    )::text")
if [[ "$acl_funcion" != "12:4:0:0:true:false:false:false:true:false" ]]; then
    echo "ACL de publicacion inesperada: $acl_funcion" >&2
    exit 1
fi

# Incluso un EXECUTE concedido por error a otro LOGIN queda cerrado por la
# identidad exacta que exige la funcion. Se usa un JSON vacio sin datos.
docker exec "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" --command \
    "GRANT USAGE ON SCHEMA vec_bolsa_publica_publicacion
         TO vec_bolsa_publica_integracion_login;
     GRANT EXECUTE ON FUNCTION
         vec_bolsa_publica_publicacion.publicar_proyeccion_v2(jsonb,text)
         TO vec_bolsa_publica_integracion_login" >/dev/null
if docker exec "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --username vec_bolsa_publica_integracion_login --dbname "$base" \
    --command \
    "SELECT vec_bolsa_publica_publicacion.publicar_proyeccion_v2(
         '{}'::jsonb, repeat('9', 64)
     )" >/dev/null 2>&1; then
    echo "un LOGIN ajeno con EXECUTE atraveso la identidad publicadora" >&2
    exit 1
fi
docker exec "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" --command \
    "REVOKE EXECUTE ON FUNCTION
         vec_bolsa_publica_publicacion.publicar_proyeccion_v2(jsonb,text)
         FROM vec_bolsa_publica_integracion_login;
     REVOKE USAGE ON SCHEMA vec_bolsa_publica_publicacion
         FROM vec_bolsa_publica_integracion_login" >/dev/null

if docker exec "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --username vec_bolsa_publica_publicador_login --dbname "$base" \
    --command 'SET ROLE vec_bolsa_publica_publicador' >/dev/null 2>&1; then
    echo "el LOGIN publicador pudo ejecutar SET ROLE" >&2
    exit 1
fi
if docker exec "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --username vec_bolsa_publica_publicador_login --dbname "$base" \
    --command "INSERT INTO vec_bolsa_publica_datos.manifiesto_consumido VALUES (repeat('d',64))" \
    >/dev/null 2>&1; then
    echo "el LOGIN publicador obtuvo DML directo" >&2
    exit 1
fi
if docker exec "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --username vec_bolsa_publica_publicador_login --dbname "$base" \
    --command "SELECT vec_bolsa_publica_publicacion.objeto_jsonb_exacto_v2('{}', ARRAY[]::text[])" \
    >/dev/null 2>&1; then
    echo "el LOGIN publicador pudo ejecutar la funcion auxiliar" >&2
    exit 1
fi

# El inventario propietario no contiene columnas de identidad personal. Las
# palabras se buscan sobre nombres de columna, nunca sobre datos publicados.
pii=$(docker exec "$contenedor" psql -X --tuples-only --no-align \
    --username postgres --dbname "$base" --command "
SELECT count(*)
  FROM information_schema.columns
	 WHERE table_schema IN ('vec_bolsa_publica_datos','vec_bolsa_publica_lectura')
	   AND lower(column_name) ~ '(^|_)(dni|nif|correo|telefono|persona|principal|actor|candidato)($|_)'")
if [[ "$pii" != "0" ]]; then
    echo "la proyeccion contiene columnas potencialmente personales" >&2
    exit 1
fi

puerto=$(docker port "$contenedor" 5432/tcp | sed -n 's/.*://p' | head -n 1)
if [[ ! "$puerto" =~ ^[0-9]+$ ]]; then
    echo "no se pudo resolver el puerto efimero de PostgreSQL" >&2
    exit 1
fi
dsn_lector="postgres://vec_bolsa_publica_integracion_login:${clave_lector}@localhost:${puerto}/${base}?sslmode=verify-full&sslrootcert=${directorio_tls}/ca.crt"
dsn_admin="postgres://postgres:${clave_admin}@localhost:${puerto}/${base}?sslmode=verify-full&sslrootcert=${directorio_tls}/ca.crt"
dsn_publicador="postgres://vec_bolsa_publica_publicador_login:${clave_publicador}@localhost:${puerto}/${base}?sslmode=verify-full&sslrootcert=${directorio_tls}/ca.crt"

GOCACHE="$cache_go" \
VEC_PRUEBA_BOLSA_PUBLICA_DSN="$dsn_lector" \
VEC_PRUEBA_BOLSA_PUBLICA_MANIFIESTO_SHA256="$ancla_inicial" \
	    go test -count=1 -run '^TestIntegracionRaizPublicaSoloArrancaConPostgreSQLAutoritativo$' \
    ./internal/app/composicion/publica

GOCACHE="$cache_go" \
VEC_PRUEBA_BOLSA_PUBLICA_DSN="$dsn_lector" \
VEC_PRUEBA_BOLSA_PUBLICA_ADMIN_DSN="$dsn_admin" \
VEC_PRUEBA_BOLSA_PUBLICA_PUBLICADOR_DSN="$dsn_publicador" \
VEC_PRUEBA_BOLSA_PUBLICA_MANIFIESTO_SHA256="$ancla_inicial" \
    go test -count=1 -run '^TestIntegracionPostgreSQLPublicoTLSACLConsultasYRevocacion$' \
	    ./internal/modules/bolsa/adapters/postgrespublico
# El marcador privado de los rechazos no puede alcanzar el log del servidor,
# aunque la base fuerce logging de parametros para el resto de sesiones.
if docker logs "$contenedor" 2>&1 | grep -F 'DNI-PRIVADO-C3-00000000T' >/dev/null; then
    echo "un valor privado de publicacion alcanzo el log PostgreSQL" >&2
    exit 1
fi

# Un publicador lento conserva el advisory xact lock hasta ROLLBACK/COMMIT.
# MVCC deja que lectores RR terminen sobre A; la migracion sigue esperando el
# candado exclusivo sin observar una ventana intermedia.
docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    >"$directorio_tls/publicador_concurrente.log" 2>&1 <<'SQL' &
SET application_name = 'vec-bolsa-publicador';
SET search_path = 'pg_catalog,pg_temp';
SET statement_timeout = '60s';
SET lock_timeout = '5s';
SET idle_in_transaction_session_timeout = '5s';
SET transaction_timeout = '2min';
SET log_parameter_max_length_on_error = 0;
BEGIN;
\o /dev/null
SELECT pg_catalog.set_config(
    'vec.prueba_proyeccion_publica',
    (SELECT payload::text FROM public.proyeccion_publica_prueba),
    false
);
\o
SET SESSION AUTHORIZATION vec_bolsa_publica_publicador_login;
SELECT vec_bolsa_publica_publicacion.publicar_proyeccion_v2(
    jsonb_set(
        current_setting('vec.prueba_proyeccion_publica')::jsonb,
        '{fuente,revision}', '"revision-bloqueo"'::jsonb
    ),
    repeat('d', 64)
);
SELECT pg_sleep(5);
ROLLBACK;
SQL
pid_publicador=$!

bloqueo_publicador=0
for _ in $(seq 1 50); do
    bloqueo_publicador=$(docker exec "$contenedor" psql -X --tuples-only --no-align \
        --username postgres --dbname "$base" --command "
SELECT count(*)
  FROM pg_catalog.pg_locks AS bloqueo
  JOIN pg_catalog.pg_stat_activity AS sesion ON sesion.pid = bloqueo.pid
 WHERE bloqueo.locktype = 'advisory' AND bloqueo.mode = 'ExclusiveLock'
   AND bloqueo.granted
   AND sesion.application_name = 'vec-bolsa-publicador'")
    [[ "$bloqueo_publicador" == "1" ]] && break
    sleep 0.1
done
if [[ "$bloqueo_publicador" != "1" ]]; then
    echo "el publicador concurrente no tomo el candado exclusivo" >&2
    wait "$pid_publicador" || true
    sed -n '1,120p' "$directorio_tls/publicador_concurrente.log" >&2
    exit 1
fi
lectura_a=$(docker exec "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --tuples-only --no-align \
    --username vec_bolsa_publica_integracion_login --dbname "$base" --command \
    "BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;
     SET LOCAL statement_timeout = '2s';
     SELECT fuente.revision || ':' || count(convocatoria.identificador_publico)::text
       FROM vec_bolsa_publica_lectura.fuente_publica_v2 AS fuente
       CROSS JOIN vec_bolsa_publica_lectura.convocatorias_publicadas_v2 AS convocatoria
      GROUP BY fuente.revision;
     COMMIT" | sed -n '3p')
if [[ "$lectura_a" != "revision-recuperada:2" ]]; then
    echo "un lector MVCC no termino sobre la ultima instantanea confirmada: $lectura_a" >&2
    wait "$pid_publicador" || true
    exit 1
fi

# La reversión sin confirmación espera al publicador, luego falla con
# cualquier contenido y conserva los tres esquemas de forma atómica.
inicio_down=$(date +%s)
{
	printf '%s\n' 'SET ROLE vec_bolsa_publica_migrador;'
	cat "$raiz/deploy/postgresql/bolsa_publica/migraciones/000001_proyeccion_publica.down.sql"
} | if docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 \
	--username postgres --dbname "$base" >/dev/null 2>&1; then
	echo "down acepto una proyeccion poblada sin confirmacion" >&2
	exit 1
fi
espera_down=$(( $(date +%s) - inicio_down ))
if ((espera_down < 1)); then
    echo "la migracion no se serializo con la publicacion en curso" >&2
    wait "$pid_publicador" || true
    exit 1
fi
if ! wait "$pid_publicador"; then
    echo "el publicador concurrente fallo" >&2
    sed -n '1,120p' "$directorio_tls/publicador_concurrente.log" >&2
    exit 1
fi
esquemas_presentes=$(docker exec "$contenedor" psql -X --tuples-only --no-align \
	--username postgres --dbname "$base" --command \
	"SELECT count(*) FROM pg_catalog.pg_namespace
	  WHERE nspname IN ('vec_bolsa_publica_datos','vec_bolsa_publica_lectura')")
if [[ "$esquemas_presentes" != "2" ]]; then
	echo "down parcial tras rechazar una proyeccion poblada" >&2
	exit 1
fi
docker exec "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" --command \
    'DROP TABLE public.proyeccion_publica_prueba'

# Se eliminan las convocatorias y catálogos dejando fuente e historial de
# anclas: también ese control reconstruible exige el token operativo.
docker exec "$contenedor" psql -X --set ON_ERROR_STOP=1 \
	--username postgres --dbname "$base" --command \
	"SET ROLE vec_bolsa_publica_propietario;
	 DELETE FROM vec_bolsa_publica_datos.convocatoria_publica;
	 DELETE FROM vec_bolsa_publica_datos.catalogo_categorias;
	 DELETE FROM vec_bolsa_publica_datos.catalogo_publico"
{
	printf '%s\n' 'SET ROLE vec_bolsa_publica_migrador;'
	cat "$raiz/deploy/postgresql/bolsa_publica/migraciones/000001_proyeccion_publica.down.sql"
} | if docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 \
	--username postgres --dbname "$base" >/dev/null 2>&1; then
	echo "down acepto una fila de fuente sin confirmacion" >&2
	exit 1
fi

{
    printf '%s\n' \
        "SET vec.confirmar_retirada_proyeccion_bolsa_publica = 'RETIRAR_PROYECCION_BOLSA_PUBLICA_RECONSTRUIBLE';" \
        'SET ROLE vec_bolsa_publica_migrador;'
    cat "$raiz/deploy/postgresql/bolsa_publica/migraciones/000001_proyeccion_publica.down.sql"
} | docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base"

# roles_down no puede desmontar el grupo mientras exista el LOGIN consumidor.
if docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 \
	--username postgres --dbname "$base" \
	<"$raiz/deploy/postgresql/bolsa_publica/roles_down.sql" >/dev/null 2>&1; then
	echo "roles_down acepto un miembro activo" >&2
	exit 1
fi
docker exec "$contenedor" psql -X --set ON_ERROR_STOP=1 \
	    --username postgres --dbname "$base" --command \
	    'REVOKE vec_bolsa_publica_consulta FROM vec_bolsa_publica_integracion_login;
	     DROP ROLE vec_bolsa_publica_integracion_login'
docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    <"$raiz/deploy/postgresql/bolsa_publica/roles_down.sql"

echo "Integracion PostgreSQL 18 de Bolsa publica superada con TLS verificado."
