#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-bolsa-convocatorias-pg-${USER:-usuario}-$$"
base=vec_bolsa_convocatorias_prueba
fixture_vectores_kms="internal/modules/bolsa/application/gobiernoconvocatorias/testdata/vectores_kms_confirmacion_borrador_v1"
salida_carrera_1=
salida_carrera_2=

generar_clave() {
    local destino=$1 valor
    valor=$(od -An -N32 -tx1 /dev/urandom | tr -d '[:space:]')
    if [[ ${#valor} -ne 64 || $valor == *[!0-9a-f]* ]]; then
        echo "no se pudo generar una clave de prueba" >&2
        exit 1
    fi
    printf -v "$destino" '%s' "$valor"
}

clave_admin=
clave_ejecutor=
clave_proyector=
clave_registrador=
clave_verificador=
generar_clave clave_admin
generar_clave clave_ejecutor
generar_clave clave_proyector
generar_clave clave_registrador
generar_clave clave_verificador

limpiar() {
    docker rm -f "$contenedor" >/dev/null 2>&1 || true
    [[ -z $salida_carrera_1 ]] || rm -f "$salida_carrera_1"
    [[ -z $salida_carrera_2 ]] || rm -f "$salida_carrera_2"
}
trap limpiar EXIT INT TERM

esperar_postgresql() {
    local proceso
    for _ in $(seq 1 120); do
        proceso=$(docker exec "$contenedor" sh -c \
            'tr -d "\n" </proc/1/comm' 2>/dev/null || true)
        if [[ $proceso == postgres ]] \
           && docker exec "$contenedor" pg_isready \
                --username postgres --dbname "$base" >/dev/null 2>&1 \
           && docker exec "$contenedor" psql -X --tuples-only --no-align \
                --username postgres --dbname "$base" \
                --command 'SELECT 1' >/dev/null 2>&1; then
            return 0
        fi
        sleep 0.25
    done
    echo "PostgreSQL no alcanzo un postmaster final estable" >&2
    docker logs --tail 30 "$contenedor" >&2 || true
    return 1
}

docker run --detach --rm --name "$contenedor" \
    --publish 127.0.0.1::5432 \
    --env POSTGRES_DB="$base" --env POSTGRES_PASSWORD="$clave_admin" \
    "$imagen" >/dev/null
esperar_postgresql

docker exec "$contenedor" mkdir -p /tmp/vec-vectores-kms-borrador-v1
docker cp "$raiz/$fixture_vectores_kms/." \
    "$contenedor:/tmp/vec-vectores-kms-borrador-v1/"

psql_archivo() {
    docker exec --interactive "$contenedor" psql -X --quiet \
        --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
        < "$raiz/$1"
}

rechazar_runtime() {
    local usuario=$1 clave=$2 consulta=$3 descripcion=$4
    if docker exec --env PGPASSWORD="$clave" "$contenedor" \
        psql -X --quiet --set ON_ERROR_STOP=1 --host 127.0.0.1 \
        --username "$usuario" --dbname "$base" \
        --command "$consulta" >/dev/null 2>&1; then
        echo "ACL invalida: $descripcion" >&2
        exit 1
    fi
}

psql_archivo deploy/postgresql/autorizacion/roles_up.sql
psql_archivo deploy/postgresql/autorizacion/roles_v2_up.sql
psql_archivo deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql
psql_archivo deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql
psql_archivo deploy/postgresql/autorizacion/migraciones/000003_proyeccion_motivos_autorizacion_v2.up.sql
psql_archivo deploy/postgresql/autorizacion/migraciones/000004_registro_decisiones_solicitud_ligada_v2.up.sql
psql_archivo deploy/postgresql/bolsa_convocatorias/roles_up.sql
psql_archivo deploy/postgresql/bolsa_convocatorias/migraciones_autorizacion/000001_revalidacion_convocatorias.up.sql
psql_archivo deploy/postgresql/bolsa_convocatorias/migraciones_autorizacion/000002_revalidacion_borradores_v2.up.sql
psql_archivo deploy/postgresql/bolsa_convocatorias/migraciones_autorizacion/000003_revalidacion_lectura_borradores_solicitud_ligada_v2.up.sql
psql_archivo deploy/postgresql/bolsa_convocatorias/migraciones/000001_almacen_convocatorias.up.sql
psql_archivo deploy/postgresql/bolsa_convocatorias/migraciones/000002_consulta_exacta_cerrada.up.sql
psql_archivo deploy/postgresql/bolsa_convocatorias/migraciones/000003_borradores_durables_cerrados.up.sql
psql_archivo deploy/postgresql/bolsa_convocatorias/migraciones/000004_confirmacion_kms_procedencia.up.sql
psql_archivo deploy/postgresql/bolsa_convocatorias/migraciones/000005_lectura_borrador_cifrado_completo.up.sql
psql_archivo deploy/postgresql/bolsa_convocatorias/migraciones/000006_preparacion_kms_instante_real.up.sql

docker exec --interactive \
    --env CLAVE_EJECUTOR="$clave_ejecutor" \
    --env CLAVE_PROYECTOR="$clave_proyector" \
    --env CLAVE_REGISTRADOR="$clave_registrador" \
    --env CLAVE_VERIFICADOR="$clave_verificador" \
    "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
\getenv clave_ejecutor CLAVE_EJECUTOR
\getenv clave_proyector CLAVE_PROYECTOR
\getenv clave_registrador CLAVE_REGISTRADOR
\getenv clave_verificador CLAVE_VERIFICADOR
CREATE ROLE vec_convocatorias_ejecutor_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_ejecutor';
CREATE ROLE vec_convocatorias_proyector_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_proyector';
CREATE ROLE vec_convocatorias_registrador_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_registrador';
CREATE ROLE vec_convocatorias_verificador_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_verificador';
GRANT vec_bolsa_convocatorias_ejecutor_consulta
    TO vec_convocatorias_ejecutor_prueba;
GRANT vec_bolsa_convocatorias_proyector_gobierno
    TO vec_convocatorias_proyector_prueba;
GRANT vec_bolsa_convocatorias_registrador_atestacion
    TO vec_convocatorias_registrador_prueba;
GRANT vec_bolsa_convocatorias_verificador_recibo
    TO vec_convocatorias_verificador_prueba;
SQL

psql_archivo deploy/postgresql/bolsa_convocatorias/pruebas_sql/acl_cierre.sql
psql_archivo deploy/postgresql/bolsa_convocatorias/pruebas_sql/vectores_confirmacion_kms.sql
psql_archivo deploy/postgresql/bolsa_convocatorias/pruebas_sql/preparacion_kms_instante_real.sql
psql_archivo deploy/postgresql/bolsa_convocatorias/pruebas_sql/verificador_recibo_acl.sql
psql_archivo deploy/postgresql/bolsa_convocatorias/pruebas_sql/acl_cierre.sql
psql_archivo deploy/postgresql/bolsa_convocatorias/pruebas_sql/borradores_durables.sql
psql_archivo deploy/postgresql/bolsa_convocatorias/pruebas_sql/lectura_borrador_cifrado_completo.sql

# Dos snapshots SERIALIZABLE compiten con ventanas solapadas [g3,g2] y
# [g2,g1]. El advisory lock solo crea una barrera de prueba; no forma parte
# del algoritmo productivo.
sql_carrera_primaria="BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SELECT pg_advisory_xact_lock(82194017);
SELECT pg_sleep(1);
SELECT r.*
  FROM public.fixture_reserva_borrador_concurrente AS f
 CROSS JOIN LATERAL
      vec_bolsa_convocatorias.reservar_decision_borrador_interna_v1(
          f.reserva, f.material, f.version_canonica,
          f.decision_canonica, f.contexto
      ) AS r;
COMMIT;"
sql_carrera_solapada=${sql_carrera_primaria/f.reserva, f.material/f.reserva_solapada, f.material}
sql_carrera_solapada=${sql_carrera_solapada/f.decision_canonica, f.contexto/f.decision_canonica_solapada, f.contexto}
salida_carrera_1=$(mktemp)
salida_carrera_2=$(mktemp)
set +e
docker exec "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --set VERBOSITY=verbose \
    --username postgres --dbname "$base" \
    --command "$sql_carrera_primaria" >"$salida_carrera_1" 2>&1 &
pid_carrera_1=$!
docker exec "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --set VERBOSITY=verbose \
    --username postgres --dbname "$base" \
    --command "$sql_carrera_solapada" >"$salida_carrera_2" 2>&1 &
pid_carrera_2=$!
wait "$pid_carrera_1"
estado_carrera_1=$?
wait "$pid_carrera_2"
estado_carrera_2=$?
set -e
if [[ $estado_carrera_1 -eq 0 && $estado_carrera_2 -eq 0 ]] \
   || [[ $estado_carrera_1 -ne 0 && $estado_carrera_2 -ne 0 ]]; then
    echo "la carrera SERIALIZABLE no produjo un ganador unico" >&2
    exit 1
fi
if [[ $estado_carrera_1 -ne 0 ]]; then
    salida_perdedora=$salida_carrera_1
else
    salida_perdedora=$salida_carrera_2
fi
if ! grep -Eq 'ERROR:[[:space:]]+40001:' "$salida_perdedora"; then
    echo "la conexion perdedora no acredito SQLSTATE 40001" >&2
    sed -n '1,20p' "$salida_perdedora" >&2
    exit 1
fi

# Reintento de la conexion perdedora: relee la reserva ya creada por el
# ganador, no duplica historia ni consumo.
docker exec "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
DO $reintento$
DECLARE
    fixture record;
    fila record;
    consulta record;
    primaria jsonb;
	lease_inicia timestamptz;
	lease_vence timestamptz;
BEGIN
    SELECT * INTO STRICT fixture
      FROM public.fixture_reserva_borrador_concurrente;
    SELECT r.* INTO STRICT fila
      FROM vec_bolsa_convocatorias.reservar_decision_borrador_interna_v1(
          fixture.reserva, fixture.material, fixture.version_canonica,
          fixture.decision_canonica, fixture.contexto
      ) AS r;
    IF fila.estado <> 'en_curso' OR fila.revision <> 1
       OR fila.cercado <> 1
       OR fila.identidades_consultadas IS DISTINCT FROM
          fixture.reserva -> 'identidades_consulta'
       OR fila.identidad_primaria IS NULL THEN
        RAISE EXCEPTION 'reintento de ventana primaria incorrecto: %', fila;
    END IF;
    primaria := fila.identidad_primaria;
	lease_inicia := fila.arrendamiento_inicia_en;
	lease_vence := fila.arrendamiento_vence_en;
	IF NOT (
		(lease_inicia = (fixture.reserva ->> 'arrendamiento_inicia_en')::timestamptz
		 AND lease_vence = (fixture.reserva ->> 'arrendamiento_vence_en')::timestamptz)
		OR
		(lease_inicia = (fixture.reserva_solapada ->> 'arrendamiento_inicia_en')::timestamptz
		 AND lease_vence = (fixture.reserva_solapada ->> 'arrendamiento_vence_en')::timestamptz)
	) THEN
		RAISE EXCEPTION 'reintento no devolvio el lease exacto del ganador';
	END IF;
    SELECT r.* INTO STRICT fila
      FROM vec_bolsa_convocatorias.reservar_decision_borrador_interna_v1(
          fixture.reserva_solapada, fixture.material,
		  fixture.version_canonica, fixture.decision_canonica_solapada,
          fixture.contexto
      ) AS r;
    IF fila.estado <> 'en_curso' OR fila.revision <> 1
       OR fila.cercado <> 1
       OR fila.identidades_consultadas IS DISTINCT FROM
          fixture.reserva_solapada -> 'identidades_consulta'
       OR fila.identidad_primaria IS DISTINCT FROM primaria
	   OR fila.arrendamiento_inicia_en <> lease_inicia
	   OR fila.arrendamiento_vence_en <> lease_vence
       OR (SELECT count(*)
             FROM vec_bolsa_convocatorias.diario_borrador_version AS h
            WHERE h.recurso_ref = 'convocatoria-concurrente#1') <> 1
       OR (SELECT count(*)
             FROM vec_bolsa_convocatorias.uso_decision_borrador AS u
            WHERE u.recurso_ref = 'convocatoria-concurrente#1') <> 1
       OR (SELECT count(*)
             FROM vec_bolsa_convocatorias.identidad_alias_borrador AS a
            WHERE a.primario_localizador_hmac IN (
                decode(repeat('1', 64), 'hex'),
                decode(repeat('3', 64), 'hex')
            )) <> 3 THEN
        RAISE EXCEPTION 'reintento concurrente no fue idempotente: %', fila;
    END IF;
    SELECT c.* INTO STRICT consulta
      FROM vec_bolsa_convocatorias.consultar_identidades_borrador_interna_v1(
          fixture.reserva -> 'identidades_consulta'
      ) AS c;
    IF consulta.identidades_consultadas IS DISTINCT FROM
       fixture.reserva -> 'identidades_consulta'
       OR consulta.identidad_primaria IS DISTINCT FROM primaria THEN
        RAISE EXCEPTION
            'reintento no dejo resolucion completa de la ventana primaria: %',
            consulta;
    END IF;
END
$reintento$;
COMMIT;
SQL

# Ejercita la frontera publica con session_user de cuentas LOGIN de minimo
# privilegio. Prueba recorridos validos hasta el cuerpo y denegaciones
# nominales antes de continuar con la persistencia y recuperacion.
psql_archivo deploy/postgresql/bolsa_convocatorias/pruebas_sql/wrappers_runtime.sql

# Regresion concurrente de la ventana temporal: dblink solo coordina dos
# conexiones del contenedor efimero y no forma parte del esquema productivo.
docker exec "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    --command 'CREATE EXTENSION dblink' >/dev/null
docker exec --interactive --env CLAVE_EJECUTOR="$clave_ejecutor" \
    "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/bolsa_convocatorias/pruebas_sql/lectura_borrador_concurrencia_temporal.sql"

# La reserva ganadora y su puntero deben sobrevivir a una parada/arranque del
# PostgreSQL real antes de que otra instancia prosiga el protocolo.
docker restart "$contenedor" >/dev/null
esperar_postgresql
docker exec "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
DO $persistencia_reinicio$
DECLARE
    fila record;
BEGIN
    SELECT d.* INTO STRICT fila
      FROM public.fixture_reserva_borrador_concurrente AS f
     CROSS JOIN LATERAL
          vec_bolsa_convocatorias.consultar_diario_borrador_interna_v1(
              f.reserva -> 'identidad'
          ) AS d;
    IF fila.estado <> 'reservado' OR fila.revision <> 1
       OR fila.cercado <> 1 THEN
        RAISE EXCEPTION 'reserva no sobrevivio al reinicio: %', fila;
    END IF;
END
$persistencia_reinicio$;
COMMIT;
SQL

# Recuperacion completa: un instante cliente futuro no vence el lease; una
# reclamacion directa se rechaza; reconciliar hace CAS revision+cercado y solo
# una concesion PDP nueva puede reclamar. El segundo lease se cierra tambien
# como no_aplicado, siempre con prueba durable de ausencia atomica.
docker exec "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL timezone = 'UTC';
DO $recuperacion_lease$
DECLARE
    fixture record;
    fila record;
    ahora timestamptz;
    ahora_texto text;
    emitida_texto text;
    vence_texto text;
    lease_vence_texto text;
    decision jsonb;
    decision_bytes bytea;
    decision_huella text;
    evidencia_huella text := encode(
        sha256(convert_to('{}', 'UTF8')), 'hex'
    );
    atestacion jsonb;
    proyeccion_decision jsonb;
    reserva_nueva jsonb;
BEGIN
    SELECT * INTO STRICT fixture
      FROM public.fixture_reserva_borrador_concurrente;
    SELECT r.* INTO STRICT fila
      FROM vec_bolsa_convocatorias.reconciliar_operacion_borrador_interna_v1(
          fixture.reserva -> 'identidad', 'reservado', 1, 1,
          clock_timestamp() + interval '100 years'
      ) AS r;
    IF fila.estado <> 'reservado' OR fila.revision <> 1
       OR fila.cercado <> 1 THEN
        RAISE EXCEPTION 'el reloj cliente vencio el lease: %', fila;
    END IF;

    PERFORM pg_sleep(GREATEST(
        0.0,
        extract(epoch FROM (
            (fixture.reserva ->> 'arrendamiento_vence_en')::timestamptz
            - clock_timestamp()
        )) + 0.05
    ));
    BEGIN
        PERFORM *
          FROM vec_bolsa_convocatorias.reclamar_reserva_borrador_interna_v1(
              1, 1, fixture.reserva, fixture.material,
              fixture.version_canonica, fixture.decision_canonica,
              fixture.contexto
          );
        RAISE EXCEPTION 'se reclamo un lease sin reconciliar';
    EXCEPTION WHEN serialization_failure THEN
        NULL;
    END;
    SELECT r.* INTO STRICT fila
      FROM vec_bolsa_convocatorias.reconciliar_operacion_borrador_interna_v1(
          fixture.reserva -> 'identidad', 'reservado', 1, 1,
          clock_timestamp()
      ) AS r;
    IF fila.estado <> 'no_aplicado' OR fila.revision <> 2
       OR fila.cercado <> 2 OR fila.prueba_desenlace_ref IS NULL
       OR fila.huella_prueba_desenlace_sha256 IS NULL THEN
        RAISE EXCEPTION 'reconciliacion no elevo CAS+cercado: %', fila;
    END IF;
    SELECT r.* INTO STRICT fila
      FROM vec_bolsa_convocatorias.reconciliar_operacion_borrador_interna_v1(
          fixture.reserva -> 'identidad', 'no_aplicado', 2, 2,
          clock_timestamp()
      ) AS r;
    IF fila.estado <> 'no_aplicado' OR fila.revision <> 2
       OR fila.cercado <> 2 OR fila.prueba_desenlace_ref IS NULL THEN
        RAISE EXCEPTION 'replay no_aplicado no fue exacto: %', fila;
    END IF;
    BEGIN
        PERFORM *
          FROM vec_bolsa_convocatorias.reconciliar_operacion_borrador_interna_v1(
              fixture.reserva -> 'identidad', 'no_aplicado', 1, 1,
              clock_timestamp()
          );
        RAISE EXCEPTION 'no_aplicado acepto control terminal obsoleto';
    EXCEPTION WHEN serialization_failure THEN
        NULL;
    END;

    ahora := date_trunc('microseconds', clock_timestamp());
    ahora_texto := to_char(ahora AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
    emitida_texto := to_char(
        (ahora - interval '1 second') AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    );
    vence_texto := to_char(
        (ahora + interval '4 minutes') AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    );
    lease_vence_texto := to_char(
        (ahora + interval '2 seconds') AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    );
    decision := convert_from(fixture.decision_canonica, 'UTF8')::jsonb
        || jsonb_build_object(
            'decision_ref', 'decision-concurrente-reclamacion',
            'emitida_en', emitida_texto, 'valida_hasta', vence_texto
        );
    decision_bytes := convert_to(decision::text, 'UTF8');
    decision_huella := encode(sha256(decision_bytes), 'hex');
    INSERT INTO vec_bolsa_convocatorias.atestacion_autorizacion_version(
        decision_ref, atestacion_ref, version, estado,
        huella_decision_sha256, evidencia_canonica,
        huella_evidencia_sha256, sobre_cose_sign1,
        huella_sobre_sha256, clave_id, revision_confianza,
        verificada_en, valida_desde, valida_hasta, registrada_en
    ) VALUES (
        'decision-concurrente-reclamacion',
        'atestacion-concurrente-reclamacion', 1, 'activa',
        decision_huella, convert_to('{}', 'UTF8'), evidencia_huella,
        decode(repeat('cf', 16), 'hex'),
        encode(sha256(decode(repeat('cf', 16), 'hex')), 'hex'),
        'clave-concurrente-reclamacion', 'confianza-concurrente', ahora,
        ahora - interval '1 minute', ahora + interval '5 minutes', ahora
    );
    INSERT INTO vec_bolsa_convocatorias.atestacion_autorizacion_actual
    VALUES (
        'decision-concurrente-reclamacion',
        'atestacion-concurrente-reclamacion', 1, 'activa', ahora
    );
    INSERT INTO vec_bolsa_convocatorias.atestacion_pdp_borrador
    VALUES (
        'decision-concurrente-reclamacion',
        'atestacion-concurrente-reclamacion', 1, 'activa',
        decision_huella, evidencia_huella,
        'verificador-concurrente-reclamacion', ahora, ahora
    );
    atestacion := jsonb_build_object(
        'decision_ref', 'decision-concurrente-reclamacion',
        'atestacion_ref', 'atestacion-concurrente-reclamacion',
        'version', 1, 'estado', 'activa',
        'huella_atestacion_sha256', evidencia_huella,
        'verificador_ref', 'verificador-concurrente-reclamacion',
        'verificada_en', ahora_texto
    );
    proyeccion_decision := fixture.reserva -> 'decision'
        || jsonb_build_object(
            'decision_ref', 'decision-concurrente-reclamacion',
            'huella_decision_sha256', decision_huella,
            'emitida_en', emitida_texto,
            'verificada_en', ahora_texto, 'valida_hasta', vence_texto,
            'atestacion_pdp', atestacion
        );
    reserva_nueva := fixture.reserva || jsonb_build_object(
        'solicitada_en', ahora_texto,
        'arrendamiento_inicia_en', ahora_texto,
        'arrendamiento_vence_en', lease_vence_texto,
        'decision', proyeccion_decision
    );
    UPDATE public.fixture_reserva_borrador_concurrente
       SET reserva = reserva_nueva, decision_canonica = decision_bytes;
    SELECT r.* INTO STRICT fila
      FROM vec_bolsa_convocatorias.reclamar_reserva_borrador_interna_v1(
          2, 2, reserva_nueva, fixture.material,
          fixture.version_canonica, decision_bytes, fixture.contexto
      ) AS r;
    IF fila.estado <> 'reservado' OR fila.revision <> 3
       OR fila.cercado <> 3 THEN
        RAISE EXCEPTION 'reclamacion PDP nueva incorrecta: %', fila;
    END IF;
    BEGIN
        PERFORM *
          FROM vec_bolsa_convocatorias.reclamar_reserva_borrador_interna_v1(
              2, 2, reserva_nueva, fixture.material,
              fixture.version_canonica, decision_bytes, fixture.contexto
          );
        RAISE EXCEPTION 'un cercado obsoleto reclamo el lease';
    EXCEPTION WHEN serialization_failure THEN
        NULL;
    END;
    SELECT r.* INTO STRICT fila
      FROM vec_bolsa_convocatorias.reconciliar_operacion_borrador_interna_v1(
          reserva_nueva -> 'identidad', 'reservado', 3, 3,
          clock_timestamp() + interval '100 years'
      ) AS r;
    IF fila.estado <> 'reservado' OR fila.revision <> 3
       OR fila.cercado <> 3 THEN
        RAISE EXCEPTION 'reloj cliente altero segundo lease: %', fila;
    END IF;
    PERFORM pg_sleep(GREATEST(
        0.0,
        extract(epoch FROM (
            (reserva_nueva ->> 'arrendamiento_vence_en')::timestamptz
            - clock_timestamp()
        )) + 0.05
    ));
    SELECT r.* INTO STRICT fila
      FROM vec_bolsa_convocatorias.reconciliar_operacion_borrador_interna_v1(
          reserva_nueva -> 'identidad', 'reservado', 3, 3,
          clock_timestamp()
      ) AS r;
    IF fila.estado <> 'no_aplicado' OR fila.revision <> 4
       OR fila.cercado <> 4 OR fila.prueba_desenlace_ref IS NULL
       OR fila.huella_prueba_desenlace_sha256 IS NULL THEN
        RAISE EXCEPTION 'segundo lease no probo no-aplicacion: %', fila;
    END IF;
END
$recuperacion_lease$;
COMMIT;
SQL

rechazar_runtime vec_convocatorias_ejecutor_prueba "$clave_ejecutor" \
    'SELECT * FROM vec_bolsa_convocatorias.version_convocatoria' \
    'el ejecutor pudo leer tablas'
rechazar_runtime vec_convocatorias_ejecutor_prueba "$clave_ejecutor" \
    "SELECT * FROM vec_bolsa_convocatorias.obtener_version_exacta_v1('{}','{}',''::bytea,''::bytea)" \
    'el ejecutor pudo invocar la consulta antes del registrador COSE'
rechazar_runtime vec_convocatorias_proyector_prueba "$clave_proyector" \
    'INSERT INTO vec_bolsa_convocatorias.version_convocatoria DEFAULT VALUES' \
    'el proyector reservado pudo escribir tablas'
rechazar_runtime vec_convocatorias_registrador_prueba "$clave_registrador" \
    'INSERT INTO vec_bolsa_convocatorias.atestacion_autorizacion_version DEFAULT VALUES' \
    'el registrador reservado pudo fabricar atestaciones'

# Invariantes de huella e inmutabilidad: todo se revierte y no deja fixtures.
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
DO $huella_rechazada$
BEGIN
    BEGIN
        INSERT INTO vec_bolsa_convocatorias.version_convocatoria(
            convocatoria_id, secuencia, referencia, estado,
            version_canonica, huella_version_sha256, registrada_en
        ) VALUES (
            'conv-prueba', 1, 'conv-prueba#1', 'borrador',
            convert_to('{}', 'UTF8'), repeat('0', 64), clock_timestamp()
        );
        RAISE EXCEPTION 'se acepto una huella falsa';
    EXCEPTION WHEN check_violation THEN
        NULL;
    END;
END
$huella_rechazada$;
INSERT INTO vec_bolsa_convocatorias.version_convocatoria(
    convocatoria_id, secuencia, referencia, estado,
    version_canonica, huella_version_sha256, registrada_en
) VALUES (
    'conv-prueba', 1, 'conv-prueba#1', 'borrador',
    convert_to('{}', 'UTF8'),
    encode(sha256(convert_to('{}', 'UTF8')), 'hex'), clock_timestamp()
);
DO $inmutable$
BEGIN
    BEGIN
        UPDATE vec_bolsa_convocatorias.version_convocatoria
           SET estado = 'publicada'
         WHERE convocatoria_id = 'conv-prueba' AND secuencia = 1;
        RAISE EXCEPTION 'se permitio mutar historia';
    EXCEPTION WHEN object_not_in_prerequisite_state THEN
        NULL;
    END;
END
$inmutable$;
ROLLBACK;
SQL

docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
REVOKE vec_bolsa_convocatorias_ejecutor_consulta
    FROM vec_convocatorias_ejecutor_prueba;
REVOKE vec_bolsa_convocatorias_proyector_gobierno
    FROM vec_convocatorias_proyector_prueba;
REVOKE vec_bolsa_convocatorias_registrador_atestacion
    FROM vec_convocatorias_registrador_prueba;
REVOKE vec_bolsa_convocatorias_verificador_recibo
    FROM vec_convocatorias_verificador_prueba;
DROP FUNCTION public.probar_lectura_borrador_concurrente_runtime();
DROP FUNCTION IF EXISTS public.probar_ejecutor_borrador_runtime();
DROP FUNCTION IF EXISTS public.probar_proyector_borrador_runtime();
DROP FUNCTION IF EXISTS public.texto_selector_borrador_runtime_valido(text);
DROP FUNCTION public.convertir_decision_borrador_runtime_v2(text);
DROP FUNCTION public.solicitud_borrador_runtime_v2_canonica(
    jsonb,bytea,jsonb
);
DROP FUNCTION public.crear_decision_borrador_runtime_prueba(
    text,text,text,text,text,jsonb,bytea,interval
);
DROP TABLE public.fixture_decision_borrador_runtime;
DROP TABLE public.fixture_reserva_borrador_concurrente;
REVOKE USAGE ON SCHEMA public
    FROM vec_autorizacion_propietario,
         vec_convocatorias_ejecutor_prueba,
         vec_convocatorias_proyector_prueba;
DROP ROLE vec_convocatorias_ejecutor_prueba;
DROP ROLE vec_convocatorias_proyector_prueba;
DROP ROLE vec_convocatorias_registrador_prueba;
DROP ROLE vec_convocatorias_verificador_prueba;
REVOKE USAGE ON SCHEMA public
    FROM vec_bolsa_convocatorias_propietario;
DROP EXTENSION dblink;
SQL

# El positivo de lectura deja una fila KMS acreditada e inmutable. Esta base
# es efimera: se exige de forma explicita la via destructiva del down.
psql_archivo deploy/postgresql/bolsa_convocatorias/migraciones/000006_preparacion_kms_instante_real.down.sql
psql_archivo deploy/postgresql/bolsa_convocatorias/migraciones/000005_lectura_borrador_cifrado_completo.down.sql
docker exec --interactive \
    --env PGOPTIONS="-c vec.confirmar_destruccion_borradores_convocatorias=DESTRUIR_HISTORIA_BORRADORES_CONVOCATORIAS_IRREVERSIBLE" \
    "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    --file - \
    < "$raiz/deploy/postgresql/bolsa_convocatorias/migraciones/000004_confirmacion_kms_procedencia.down.sql"
psql_archivo deploy/postgresql/bolsa_convocatorias/pruebas_sql/down_confirmacion_kms_limpio.sql

# La carrera dejo historia real: un down sin confirmacion debe negarse. Se
# prueba primero el rechazo y luego la via destructiva explicita en esta BD
# efimera de integracion.
if psql_archivo \
    deploy/postgresql/bolsa_convocatorias/migraciones/000003_borradores_durables_cerrados.down.sql \
    >/dev/null 2>&1; then
    echo "000003 down destruyo historia sin confirmacion" >&2
    exit 1
fi
docker exec --interactive \
    --env PGOPTIONS="-c vec.confirmar_destruccion_borradores_convocatorias=DESTRUIR_HISTORIA_BORRADORES_CONVOCATORIAS_IRREVERSIBLE" \
    "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    --file - \
    < "$raiz/deploy/postgresql/bolsa_convocatorias/migraciones/000003_borradores_durables_cerrados.down.sql"
psql_archivo deploy/postgresql/bolsa_convocatorias/pruebas_sql/down_borradores_limpio.sql
psql_archivo deploy/postgresql/bolsa_convocatorias/migraciones/000002_consulta_exacta_cerrada.down.sql
docker exec --interactive \
    --env PGOPTIONS="-c vec.confirmar_destruccion_bolsa_convocatorias=DESTRUIR_HISTORIA_BOLSA_CONVOCATORIAS_IRREVERSIBLE" \
    "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    --file - \
    < "$raiz/deploy/postgresql/bolsa_convocatorias/migraciones/000001_almacen_convocatorias.down.sql"
psql_archivo deploy/postgresql/bolsa_convocatorias/migraciones_autorizacion/000003_revalidacion_lectura_borradores_solicitud_ligada_v2.down.sql
psql_archivo deploy/postgresql/bolsa_convocatorias/migraciones_autorizacion/000002_revalidacion_borradores_v2.down.sql
psql_archivo deploy/postgresql/bolsa_convocatorias/pruebas_sql/down_autorizacion_borradores_limpio.sql
psql_archivo deploy/postgresql/bolsa_convocatorias/migraciones_autorizacion/000001_revalidacion_convocatorias.down.sql
psql_archivo deploy/postgresql/bolsa_convocatorias/roles_down.sql

echo 'integracion PostgreSQL de convocatorias: OK'
