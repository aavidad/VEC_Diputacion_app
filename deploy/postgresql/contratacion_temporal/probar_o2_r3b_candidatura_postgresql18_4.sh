#!/usr/bin/env bash
set -Eeuo pipefail

readonly prefijo='ct-o2-r3b-pg-20260831'
readonly etiqueta='vec.prueba=ct-o2-r3b-pg-20260831'
readonly imagen='postgres@sha256:882236b897e39051d2368c5ccc6cda944904723506b2dfc97f2a8f5bc9afa382'
readonly ajeno='d6217278d871'
readonly contenedor="${prefijo}-db"
readonly red="${prefijo}-net"
readonly volumen="${prefijo}-datos"

directorio="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd -P)"
raiz="$(git -C "$directorio" rev-parse --show-toplevel)"
if [[ ${VEC_CT_O2_R3B_BD_DESECHABLE:-} != SI ]]; then
    printf 'R3B exige VEC_CT_O2_R3B_BD_DESECHABLE=SI\n' >&2
    exit 64
fi
for herramienta in docker rg flock go; do
    command -v "$herramienta" >/dev/null 2>&1 || {
        printf '%s no disponible\n' "$herramienta" >&2
        exit 69
    }
done

unset PGHOST PGHOSTADDR PGPORT PGDATABASE PGUSER PGPASSWORD PGPASSFILE PGSERVICE
unset PGSERVICEFILE PGOPTIONS PGAPPNAME PGSSLMODE PGCONNECT_TIMEOUT PGCLIENTENCODING
unset PGTARGETSESSIONATTRS PGLOADBALANCEHOSTS PGCHANNELBINDING PGREQUIREAUTH
unset PGSSLCERT PGSSLKEY PGSSLROOTCERT PGSSLCRL PGSSLCRLDIR PGREQUIREPEER
unset PGGSSENCMODE PGKRBSRVNAME PGGSSLIB PGSYSCONFDIR PGLOCALEDIR
unset DATABASE_URL DB_DSN DSN VEC_DATABASE_URL

exec 9>/tmp/vec-postgres-dynamic-20260831.lock
if ! flock -w 900 9; then
    printf 'no se adquirio el bloqueo PostgreSQL dinamico\n' >&2
    exit 75
fi

if [[ -n $(docker ps -aq --filter "name=^/${contenedor}$") ]] ||
   docker network inspect "$red" >/dev/null 2>&1 ||
   [[ -n $(docker volume ls -q --filter "name=^${volumen}$") ]] ||
   find /tmp /var/tmp -maxdepth 1 -name "${prefijo}*" -print -quit | rg -q .; then
    printf 'colision de recursos propios R3B\n' >&2
    exit 65
fi
imagen_id="$(docker image inspect --format '{{.Id}}' "$imagen")"
if [[ $imagen_id != 'sha256:882236b897e39051d2368c5ccc6cda944904723506b2dfc97f2a8f5bc9afa382' ]]; then
    printf 'imagen local R3B divergente\n' >&2
    exit 65
fi
ajeno_antes="$(docker inspect --format '{{.Id}}|{{.State.Status}}|{{.Name}}|{{.Image}}' "$ajeno")"
printf '[R3B] ajeno antes: %s\n' "$ajeno_antes"

temporal="$(mktemp -d "/tmp/${prefijo}.XXXXXX")"
socket="$temporal/socket"
mkdir -m 0777 -- "$socket"
creado_contenedor=0
creada_red=0
creado_volumen=0

limpiar() {
    local estado=$?
    local residuo=0
    trap - EXIT
    if (( creado_contenedor )); then
        docker rm -f -v "$contenedor" >/dev/null 2>&1 || residuo=1
    fi
    if (( creada_red )); then
        docker network rm -- "$red" >/dev/null 2>&1 || residuo=1
    fi
    if (( creado_volumen )); then
        docker volume rm -- "$volumen" >/dev/null 2>&1 || residuo=1
    fi
    rm -rf -- "$temporal"
    if [[ -n $(docker ps -aq --filter "label=$etiqueta") ]] ||
       [[ -n $(docker network ls -q --filter "label=$etiqueta") ]] ||
       [[ -n $(docker volume ls -q --filter "label=$etiqueta") ]] ||
       find /tmp /var/tmp -maxdepth 1 -name "${prefijo}*" -print -quit | rg -q .; then
        residuo=1
    fi
    ajeno_despues="$(docker inspect --format '{{.Id}}|{{.State.Status}}|{{.Name}}|{{.Image}}' "$ajeno" 2>/dev/null || true)"
    printf '[R3B] ajeno despues: %s\n' "$ajeno_despues"
    if [[ $ajeno_despues != "$ajeno_antes" ]]; then
        printf 'el contenedor ajeno cambio durante R3B\n' >&2
        residuo=1
    fi
    if (( residuo )); then
        printf 'limpieza R3B incompleta\n' >&2
        exit 1
    fi
    exit "$estado"
}
trap limpiar EXIT INT TERM

docker network create --internal --label "$etiqueta" "$red" >/dev/null
creada_red=1
docker volume create --label "$etiqueta" "$volumen" >/dev/null
creado_volumen=1
creado_contenedor=1
docker run -d --pull=never --name "$contenedor" --label "$etiqueta" \
    --network "$red" --memory=1536m --cpus=2 --pids-limit=256 \
    --mount "type=volume,src=$volumen,dst=/var/lib/postgresql" \
    --mount "type=bind,src=$socket,dst=/var/run/postgresql" \
    --mount "type=bind,src=$raiz,dst=/repo,readonly" \
    --env POSTGRES_HOST_AUTH_METHOD=trust --env POSTGRES_INITDB_ARGS='--encoding=UTF8' \
    "$imagen" -c listen_addresses= -c unix_socket_directories=/var/run/postgresql >/dev/null

listo=0
for _ in {1..80}; do
    if docker exec "$contenedor" pg_isready -q -h /var/run/postgresql -U postgres 2>/dev/null; then
        listo=1
        break
    fi
    if [[ $(docker inspect --format '{{.State.Running}}' "$contenedor") != true ]]; then
        break
    fi
    sleep 0.25
done
if (( ! listo )); then
    printf 'PostgreSQL R3B no quedo listo\n' >&2
    exit 1
fi

psql_usuario() {
    local usuario=$1
    shift
    docker exec -i "$contenedor" psql -X --no-psqlrc -h /var/run/postgresql \
        -U "$usuario" -d postgres --set ON_ERROR_STOP=1 --set VERBOSITY=terse "$@"
}

archivo() {
    local usuario=$1
    local ruta=$2
    psql_usuario "$usuario" --file "/repo/deploy/postgresql/${ruta}"
}

valor() {
    psql_usuario postgres --tuples-only --no-align --command "$1"
}

esperar_fallo() {
    local descripcion=$1
    shift
    local salida
    if salida="$("$@" 2>&1)"; then
        printf 'se esperaba rechazo: %s\n%s\n' "$descripcion" "$salida" >&2
        return 1
    fi
    printf '[R3B] rechazo verificado: %s\n' "$descripcion"
}

psql_usuario postgres >/dev/null <<'SQL'
DO $cierre_publico$
BEGIN
    EXECUTE pg_catalog.format(
        'REVOKE ALL PRIVILEGES ON DATABASE %I FROM PUBLIC',
        pg_catalog.current_database()
    );
END
$cierre_publico$;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SQL

printf '[R3B] PostgreSQL, contexto de actor y autorizacion V3\n'
for ruta in \
    contexto_actor_v1/roles_up.sql \
    contexto_actor_v1/migraciones/000001_contexto_actor_v1.up.sql \
    contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql \
    contexto_actor_v1/pruebas_sql/fixtures_sinteticos.sql \
    autorizacion/pruebas_sql/fixture_contexto_actor_v3.sql \
    contratacion_temporal/pruebas_sql/fixture_contexto_actor_b_o3.sql
do
    archivo postgres "$ruta" >/dev/null
done
psql_usuario postgres --command \
    'CREATE ROLE vec_contexto_r3b_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS; GRANT vec_contexto_actor_v1_runtime TO vec_contexto_r3b_runtime WITH ADMIN FALSE, INHERIT TRUE, SET FALSE' >/dev/null
psql_usuario vec_contexto_r3b_runtime >/dev/null <<'SQL'
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SELECT count(*) FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(
 'oca_registro_v3_000000000000000000000000','rca_registro_v3_000000000000000000000000',
 'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa','prf_sintetico_cccccccccccccccccccccccc',
 'certificado','alto',clock_timestamp());
COMMIT;
SQL
psql_usuario postgres >/dev/null <<'SQL'
CREATE EXTENSION pgcrypto WITH SCHEMA public;
REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA public FROM PUBLIC;
SQL
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
    autorizacion/migraciones/000007_revalidacion_viva_decision_contexto_actor_v3.up.sql
do
    archivo postgres "$ruta" >/dev/null
done

printf '[R3B] migraciones CT 000001..000005, 000046 y AD-3\n'
archivo postgres contratacion_temporal/roles_up.sql >/dev/null
archivo postgres contratacion_temporal/migraciones_autorizacion/000001_revalidacion_analisis_v3.up.sql >/dev/null
psql_usuario postgres --command \
    'CREATE ROLE vec_ct_r3b_migrador LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS; GRANT CONNECT ON DATABASE postgres TO vec_ct_r3b_migrador; GRANT vec_contratacion_temporal_migrador TO vec_ct_r3b_migrador WITH ADMIN FALSE, INHERIT FALSE, SET TRUE; CREATE ROLE vec_ct_r3b_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS; GRANT CONNECT ON DATABASE postgres TO vec_ct_r3b_runtime; GRANT vec_contratacion_temporal_ejecutor TO vec_ct_r3b_runtime WITH ADMIN FALSE, INHERIT TRUE, SET FALSE' >/dev/null
archivo vec_ct_r3b_migrador contratacion_temporal/migraciones/000001_preparacion_altas.up.sql >/dev/null
archivo vec_ct_r3b_migrador contratacion_temporal/migraciones/000002_rotacion_hmac.up.sql >/dev/null
archivo postgres autorizacion_atestada_v3/roles_up.sql >/dev/null
psql_usuario postgres --command \
    'CREATE ROLE vec_ad3_r3b_migrador LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS; GRANT CONNECT ON DATABASE postgres TO vec_ad3_r3b_migrador; GRANT vec_autorizacion_atestada_v3_migrador TO vec_ad3_r3b_migrador WITH ADMIN FALSE, INHERIT FALSE, SET TRUE' >/dev/null
archivo vec_ad3_r3b_migrador autorizacion_atestada_v3/migraciones/000001_gobierno_y_registro_v3.up.sql >/dev/null
archivo vec_ad3_r3b_migrador autorizacion_atestada_v3/migraciones/000002_consumidor_capacidad_v3.up.sql >/dev/null
for migracion in \
    000003_expediente_confirmacion_atestada \
    000004_integridad_agregado_alta \
    000005_funcion_confirmar_alta_atestada
do
    archivo vec_ct_r3b_migrador "contratacion_temporal/migraciones/${migracion}.up.sql" >/dev/null
done

psql_usuario postgres >/dev/null <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
INSERT INTO vec_contratacion_temporal.identidad_reserva_alta VALUES (
 'hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v2:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
 'reserva:alta:r3b:backfill','expediente:ct:r3b:backfill','2026/R3B-BACKFILL',
 'recibo:alta:r3b:backfill',
 'hmac-sha256:vec.contratacion-temporal.huella-peticion/v2:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
 'organizacion:dipgra','actor:rrhh:r3b','perfil:rrhh:r3b','2026-08-31 09:00:00.123456+00');
INSERT INTO vec_contratacion_temporal.alias_ambito_alta VALUES
 ('hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v2:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
  'hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v2:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',2,'2026-08-31 09:00:00.123456+00'),
 ('hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v1:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc',
  'hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v2:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',1,'2026-08-31 09:00:00.123456+00');
INSERT INTO vec_contratacion_temporal.alias_huella_alta VALUES
 ('hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v2:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',2,
  'hmac-sha256:vec.contratacion-temporal.huella-peticion/v2:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb','2026-08-31 09:00:00.123456+00'),
 ('hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v2:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',1,
  'hmac-sha256:vec.contratacion-temporal.huella-peticion/v1:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd','2026-08-31 09:00:00.123456+00');
INSERT INTO vec_contratacion_temporal.reserva_alta_version VALUES (
 'hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v2:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
 1,'reservada',NULL,NULL,NULL,NULL,'2026-08-31 09:00:00.123456+00',NULL);
INSERT INTO vec_contratacion_temporal.reserva_alta_actual VALUES (
 'hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v2:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',1);
COMMIT;
SQL

psql_usuario postgres >/dev/null <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
CREATE FUNCTION vec_contratacion_temporal.instante_utc_json_canonico_v2(
 p_valor jsonb, p_fecha_civil boolean
) RETURNS boolean LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
DECLARE v_texto text; v_instante timestamptz;
BEGIN
 IF jsonb_typeof(p_valor)<>'string' OR p_fecha_civil IS NULL THEN RETURN false; END IF;
 v_texto:=p_valor#>>'{}'; v_instante:=v_texto::timestamptz;
 RETURN (p_fecha_civil AND v_texto~'^[0-9]{4}-[0-9]{2}-[0-9]{2}T00:00:00Z$') OR
        (NOT p_fecha_civil AND v_texto~('^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-2][0-9]:' ||
         '[0-5][0-9]:[0-5][0-9]([.][0-9]{0,5}[1-9])?Z$'));
EXCEPTION WHEN OTHERS THEN RETURN false;
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.instante_utc_json_canonico_v2(jsonb,boolean) FROM PUBLIC;
COMMIT;
SQL
archivo postgres contratacion_temporal/migraciones/000046_ejecuciones_seleccion_llamamiento_o6.up.sql >/dev/null
archivo vec_ct_r3b_migrador contratacion_temporal/migraciones/000047_candidatura_alta_durable_o2_r3b.up.sql >/dev/null

preflight="$(valor "SELECT concat_ws('|',current_setting('server_version_num'),
 (to_regclass('vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6') IS NOT NULL),
 (to_regclass('vec_contratacion_temporal.candidatura_alta_tecnica') IS NOT NULL),
 (to_regprocedure('vec_contratacion_temporal.confirmar_alta_atestada_v2(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)') IS NOT NULL))")"
if [[ $preflight != '180004|t|t|t' ]]; then
    printf 'preflight R3B incompatible: %s\n' "$preflight" >&2
    exit 1
fi

printf '[R3B] adaptador Go, backfill, replay, concurrencia, rotacion y ACL\n'
(
    cd -- "$raiz"
    VEC_CT_O2_R3B_INTEGRACION_PG=SI \
    VEC_CT_O2_R3B_RUNTIME_DSN="host=$socket port=5432 dbname=postgres user=vec_ct_r3b_runtime sslmode=disable" \
    VEC_CT_O2_R3B_ADMIN_DSN="host=$socket port=5432 dbname=postgres user=postgres sslmode=disable" \
    GOPROXY=off go test ./internal/modules/contrataciontemporal/adapters/postgres \
        -run '^TestCandidaturaAltaPostgreSQL18DeExtremoATerminal$' -count=1
)

psql_usuario postgres >/dev/null <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DELETE FROM vec_contratacion_temporal.politica_generaciones_hmac_alta;
INSERT INTO vec_contratacion_temporal.politica_generaciones_hmac_alta VALUES
 (2,0,'activa',date_trunc('microseconds',clock_timestamp())),
 (1,1,'retenida',date_trunc('microseconds',clock_timestamp()));
COMMIT;
SQL

printf '[R3B] ligadura V2 real, canon O2-05, mutacion y revocacion\n'
psql_usuario postgres --command \
    'GRANT EXECUTE ON FUNCTION public.gen_random_bytes(integer) TO vec_autorizacion_atestada_v3_propietario' >/dev/null
archivo postgres autorizacion_atestada_v3/pruebas_sql/preparar_entorno_o2_05.sql >/dev/null
archivo postgres autorizacion_atestada_v3/pruebas_sql/ayudantes_o2_05.sql >/dev/null
psql_usuario postgres >/dev/null <<'SQL'
CREATE FUNCTION public.resolver_vector_r3b(p_caso text)
RETURNS TABLE(resultado text,reserva_ref text,expediente_ref text,numero_visible text,
 recibo_ref text,ambito_hmac text,huella_peticion_hmac text,organizacion_ref text,
 actor_ref text,perfil_ref text,instante_efecto timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE v public.vectores_o2_05%ROWTYPE; a jsonb; s jsonb; p jsonb;
BEGIN
 SELECT * INTO STRICT v FROM public.vectores_o2_05 WHERE caso=p_caso;
 a:=convert_from(v.alta,'UTF8')::jsonb; s:=convert_from(v.sellos,'UTF8')::jsonb;
 p:=jsonb_build_array(s->'activo')||(s->'retenidos');
 RETURN QUERY SELECT * FROM vec_contratacion_temporal.resolver_candidatura_alta_tecnica_v1(
  ARRAY(SELECT x.valor->>'ambito_hmac' FROM jsonb_array_elements(p)
        WITH ORDINALITY x(valor,orden) ORDER BY x.orden),
  ARRAY(SELECT x.valor->>'huella_hmac' FROM jsonb_array_elements(p)
        WITH ORDINALITY x(valor,orden) ORDER BY x.orden),
  a->>'organizacion_ref',a->>'actor_ref',a->>'perfil_ref',a->>'reserva_ref',
  a->>'expediente_ref',a->>'numero_visible',a->>'recibo_ref',(a->>'creado_en')::timestamptz);
END $f$;
CREATE FUNCTION public.invocar_vector_r3b(p_caso text)
RETURNS TABLE(expediente_ref text,numero_visible text,version numeric,recibo_ref text,
 auditoria_ref text,evento_ref text,confirmada_en timestamptz,recibo_huella_sha256 text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE v public.vectores_o2_05%ROWTYPE;
BEGIN
 SELECT * INTO STRICT v FROM public.vectores_o2_05 WHERE caso=p_caso;
 RETURN QUERY SELECT * FROM vec_contratacion_temporal.confirmar_alta_atestada_v2(
  v.capacidad,v.decision,v.motivo,v.contexto,v.persona_version,v.perfil_version,
  v.payload,v.cose,v.evidencia,v.spki,v.alta,v.sellos);
END $f$;
REVOKE ALL ON FUNCTION public.resolver_vector_r3b(text),public.invocar_vector_r3b(text) FROM PUBLIC;
GRANT USAGE ON SCHEMA public TO vec_ct_r3b_runtime;
GRANT EXECUTE ON FUNCTION public.resolver_vector_r3b(text),public.invocar_vector_r3b(text)
 TO vec_ct_r3b_runtime;
SQL

preparar() {
    psql_usuario postgres --command \
        "SELECT public.preparar_vector_o2_05('$1','valido',${2:-1})" >/dev/null
}
resolver_vector() {
    psql_usuario vec_ct_r3b_runtime >/dev/null <<SQL
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC'; SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT * FROM public.resolver_vector_r3b('$1'); COMMIT;
SQL
}
confirmar_vector() {
    psql_usuario vec_ct_r3b_runtime >/dev/null <<SQL
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC'; SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT * FROM public.invocar_vector_r3b('$1'); COMMIT;
SQL
}

printf '[R3B] adaptador publico Go, doce entradas y ocho columnas reales\n'
emisor_publico="$temporal/confianza-atestacion-r3b.test"
adaptador_publico="$temporal/confirmacion-alta-r3b.test"
(
    cd -- "$raiz"
    GOPROXY=off go test -c -o "$emisor_publico" \
        ./internal/vec/adapters/seguridad/confianzaatestacion
    GOPROXY=off go test -c -o "$adaptador_publico" \
        ./internal/modules/contrataciontemporal/adapters/postgres
)
preparar adaptador_publico
psql_usuario postgres >/dev/null <<'SQL'
WITH original AS (
    SELECT caso, convert_from(alta, 'UTF8')::jsonb AS a
      FROM public.vectores_o2_05
     WHERE caso = 'adaptador_publico'
), normalizado AS (
    SELECT caso,
           jsonb_set(jsonb_set(jsonb_set(jsonb_set(
               a, '{solicitud,periodo,inicio}',
               to_jsonb(left(a #>> '{solicitud,periodo,inicio}', 10))
           ), '{solicitud,periodo,fin}',
               to_jsonb(left(a #>> '{solicitud,periodo,fin}', 10))
           ), '{solicitud,rc,fecha}', '""'::jsonb
           ), '{solicitud,rc,importe,moneda}', '"EUR"'::jsonb) AS a
      FROM original
)
UPDATE public.vectores_o2_05 AS v
   SET alta = vec_contratacion_temporal.reconstruir_efecto_alta_v2(n.a)
  FROM normalizado AS n
 WHERE v.caso = n.caso;
SQL
entrada_publica="$temporal/entrada-publica-r3b.json"
bundle_publico="$temporal/bundle-publico-r3b.json"
valor "SELECT public.exportar_entrada_go_o2_05('adaptador_publico')" \
    >"$entrada_publica"
chmod 600 "$entrada_publica"
VEC_O205_VECTOR_ENTRADA="$entrada_publica" \
VEC_O205_VECTOR_SALIDA="$bundle_publico" \
    "$emisor_publico" -test.run '^TestGenerarVectorO205ParaSQL$' -test.count=1
docker cp "$bundle_publico" "$contenedor:/tmp/bundle-publico-r3b.json"
docker exec "$contenedor" chmod 644 /tmp/bundle-publico-r3b.json
psql_usuario postgres --command \
    "SELECT public.aplicar_bundle_go_o2_05('adaptador_publico',pg_catalog.pg_read_file('/tmp/bundle-publico-r3b.json')::jsonb)" \
    >/dev/null
docker exec "$contenedor" rm -f /tmp/bundle-publico-r3b.json
VEC_CT_O2_R3B_INTEGRACION_PG=SI \
VEC_CT_O2_R3B_RUNTIME_DSN="host=$socket port=5432 dbname=postgres user=vec_ct_r3b_runtime sslmode=disable" \
VEC_CT_O2_R3B_ADMIN_DSN="host=$socket port=5432 dbname=postgres user=postgres sslmode=disable" \
VEC_CT_O2_R3B_VECTOR_ENTRADA="$entrada_publica" \
VEC_CT_O2_R3B_VECTOR_BUNDLE="$bundle_publico" \
    "$adaptador_publico" \
        -test.run '^TestConfirmacionAltaPublicaPostgreSQL18DesdeDosPools$' \
        -test.count=1
rm -f "$entrada_publica" "$bundle_publico" "$emisor_publico" "$adaptador_publico"

preparar ligada
resolver_vector ligada
confirmar_vector ligada
confirmar_vector ligada
[[ "$(valor "SELECT concat_ws('|',(SELECT count(*) FROM vec_contratacion_temporal.expediente_alta WHERE expediente_ref='expediente:ct:o205:ligada'),(SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3 WHERE decision_ref='decision:ct:o205:ligada'))")" == '1|1' ]]

preparar coordenada
resolver_vector coordenada
psql_usuario postgres --command \
    "SELECT public.mutar_efecto_o2_05('coordenada','expediente_ref','\"expediente:ct:o205:alterada\"'::jsonb)" >/dev/null
esperar_fallo 'mutacion de coordenada ligada' confirmar_vector coordenada
[[ "$(valor "SELECT concat_ws('|',(SELECT count(*) FROM vec_contratacion_temporal.expediente_alta WHERE expediente_ref='expediente:ct:o205:alterada'),(SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3 WHERE decision_ref='decision:ct:o205:coordenada'))")" == '0|0' ]]

preparar sesion_revocada 2
resolver_vector sesion_revocada
psql_usuario postgres --command \
    "BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE; SELECT public.durabilizar_decision_o2_05('sesion_revocada'); COMMIT" >/dev/null
psql_usuario postgres >/dev/null <<'SQL'
BEGIN; SET LOCAL ROLE vec_autorizacion_propietario;
INSERT INTO vec_autorizacion.control_sesion_v1(
 control_sesion_ref,revision,sesion_ref,estado,huella_sha256,
 sesion_revalidada_en,sesion_valida_hasta
) VALUES ('cse_registro_v3_0000000000000000000000',2,
 'ses_registro_v3_0000000000000000000000','revocada',repeat('a',64),
 clock_timestamp(),clock_timestamp()+interval '2 hours');
UPDATE vec_autorizacion.control_sesion_actual_v1 SET revision=2,
 actualizada_en=clock_timestamp(),acto_ref='acto:sesion:r3b:revocada'
 WHERE sesion_ref='ses_registro_v3_0000000000000000000000';
COMMIT;
SQL
esperar_fallo 'revocacion viva antes del efecto' confirmar_vector sesion_revocada
[[ "$(valor "SELECT concat_ws('|',(SELECT count(*) FROM vec_autorizacion.decision_concedida_contexto_actor_v3 WHERE decision_ref='decision:ct:o205:sesion_revocada'),(SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3 WHERE decision_ref='decision:ct:o205:sesion_revocada'),(SELECT count(*) FROM vec_contratacion_temporal.expediente_alta WHERE expediente_ref='expediente:ct:o205:sesion_revocada'))")" == '1|0|0' ]]

printf '[R3B] down protegido, retirada explicita y reinstalacion\n'
psql_usuario postgres --command \
    'REVOKE EXECUTE ON FUNCTION public.resolver_vector_r3b(text),public.invocar_vector_r3b(text) FROM vec_ct_r3b_runtime; DROP FUNCTION public.resolver_vector_r3b(text),public.invocar_vector_r3b(text)' >/dev/null
esperar_fallo 'down ordinario con historia candidata' archivo vec_ct_r3b_migrador \
    contratacion_temporal/migraciones/000047_candidatura_alta_durable_o2_r3b.down.sql
docker exec -i --env \
    PGOPTIONS='-c vec.confirmar_destruccion_contratacion_temporal=DESTRUIR_HISTORIA_CONTRATACION_TEMPORAL_IRREVERSIBLE' \
    "$contenedor" psql -X --no-psqlrc -h /var/run/postgresql -U vec_ct_r3b_migrador \
    -d postgres --set ON_ERROR_STOP=1 --file \
    /repo/deploy/postgresql/contratacion_temporal/migraciones/000047_candidatura_alta_durable_o2_r3b.down.sql >/dev/null
archivo vec_ct_r3b_migrador contratacion_temporal/migraciones/000047_candidatura_alta_durable_o2_r3b.up.sql >/dev/null
[[ "$(valor "SELECT concat_ws('|',(to_regclass('vec_contratacion_temporal.candidatura_alta_tecnica') IS NOT NULL),has_function_privilege('vec_ct_r3b_runtime','vec_contratacion_temporal.confirmar_alta_atestada_v2(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)','EXECUTE'),has_function_privilege('vec_ct_r3b_runtime','vec_contratacion_temporal.confirmar_alta_atestada_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)','EXECUTE'))")" == 'true|true|false' ]]

printf '[CT-O2-R3B:PG18.4] evidencia completa; instalacion tras 000046, V2 y residuos cero\n'
