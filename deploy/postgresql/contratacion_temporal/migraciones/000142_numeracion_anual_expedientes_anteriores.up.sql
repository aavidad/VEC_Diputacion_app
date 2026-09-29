\set ON_ERROR_STOP on
-- CT-000142: número anual para los expedientes dados de alta antes de la
-- numeración correlativa (CT-000103). Su número visible era el identificador
-- técnico «AAAA/CT-<hex>», y la portada los mostraba «Sin numerar».
--
-- La historia no se reescribe: expediente_alta, las versiones integrales y sus
-- publicaciones son de solo adición y conservan el número original, que sigue
-- ligado a los recibos y huellas ya emitidos. La asignación es una actuación
-- nueva en numeracion_anual_asignada (fecha, motivo catalogado, actor de
-- sistema y versión del expediente observada), con un evento en el outbox
-- encadenado del expediente. Las consultas de RRHH (cuadro, totales y detalle)
-- y la lista de expedientes del centro muestran el número asignado mediante
-- numero_visible_vigente_v1; el resto de su lógica no cambia.
--
-- El número sale del mismo contador anual que usan las altas nuevas
-- (siguiente_numero_visible_v1), en orden de fecha de alta. Repetir la
-- asignación no renumera: solo toma expedientes técnicos sin número asignado.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000142', 0)
);

DO $prevalidacion$
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'CT-000142 exige el rol propietario';
    END IF;
    IF pg_catalog.to_regclass(
        'vec_contratacion_temporal.numeracion_anual_asignada'
    ) IS NOT NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.numero_visible_vigente_v1(text,text)'
    ) IS NOT NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.asignar_numeracion_anual_anteriores_v1()'
    ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'CT-000142 ya instalada';
    END IF;
    IF pg_catalog.to_regclass('vec_contratacion_temporal.numeracion_parametros') IS NULL
    OR pg_catalog.to_regclass('vec_contratacion_temporal.numeracion_expedientes') IS NULL
    OR pg_catalog.to_regclass('vec_contratacion_temporal.expediente_alta') IS NULL
    OR pg_catalog.to_regclass('vec_contratacion_temporal.expediente_integral_actual') IS NULL
    OR pg_catalog.to_regclass('vec_contratacion_temporal.expediente_version_integral') IS NULL
    OR pg_catalog.to_regclass('vec_contratacion_temporal.outbox_expediente_integral') IS NULL
    OR pg_catalog.to_regclass('vec_contratacion_temporal.control_cadenas_expediente_integral') IS NULL
    OR pg_catalog.to_regclass('vec_contratacion_temporal.entrega_peticion_centro_confirmacion') IS NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.siguiente_numero_visible_v1(integer)'
    ) IS NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.rechazar_mutacion_historia_v1()'
    ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para CT-000142: faltan CT-000068, CT-000103 o CT-000126';
    END IF;
END
$prevalidacion$;

-- Actuación de numeración: una fila por expediente, de solo adición.
CREATE TABLE vec_contratacion_temporal.numeracion_anual_asignada (
    expediente_ref text PRIMARY KEY
        REFERENCES vec_contratacion_temporal.expediente_alta (expediente_ref),
    numero_anterior text NOT NULL UNIQUE,
    numero_visible text NOT NULL UNIQUE,
    anio smallint NOT NULL,
    version_expediente numeric(20, 0) NOT NULL,
    motivo_clave text NOT NULL,
    actor_ref text NOT NULL,
    operacion_ref text NOT NULL UNIQUE,
    evento_ref text NOT NULL UNIQUE
        REFERENCES vec_contratacion_temporal.outbox_expediente_integral (evento_ref),
    ejecutado_por text NOT NULL,
    asignado_en timestamptz(6) NOT NULL,
    FOREIGN KEY (expediente_ref, version_expediente)
        REFERENCES vec_contratacion_temporal.expediente_version_integral (expediente_ref, version),
    CHECK (numero_anterior ~ '^[0-9]{4}/CT-[0-9a-f]{12,}$'),
    CHECK (numero_visible ~ '^[0-9]{4}/[A-Za-z0-9._-]{1,40}$'
           AND numero_visible !~ '^[0-9]{4}/CT-[0-9a-f]{12,}$'),
    CHECK (anio BETWEEN 1 AND 9999
           AND pg_catalog.left(numero_visible, 4)::integer = anio
           AND pg_catalog.left(numero_anterior, 4)::integer = anio),
    CHECK (version_expediente BETWEEN 1 AND 9007199254740991::numeric),
    -- Motivo del catálogo de actuaciones de numeración; hoy tiene un valor.
    CHECK (motivo_clave IN ('numeracion_anual_expediente_anterior')),
    CHECK (actor_ref = 'sistema:contratacion_temporal:migracion:000142'),
    CHECK (operacion_ref ~ '^operacion:ct-numeracion-anual:[0-9a-f]{32}$'),
    CHECK (evento_ref ~ '^evento:ct-numeracion-anual:[0-9a-f]{32}$'),
    CHECK (pg_catalog.length(ejecutado_por) BETWEEN 1 AND 63),
    CHECK (asignado_en = pg_catalog.date_trunc('microseconds', asignado_en))
);

CREATE TRIGGER numeracion_anual_asignada_inmutable
BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.numeracion_anual_asignada
FOR EACH ROW
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();

CREATE TRIGGER numeracion_anual_asignada_no_truncar
BEFORE TRUNCATE
ON vec_contratacion_temporal.numeracion_anual_asignada
FOR EACH STATEMENT
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();

ALTER TABLE vec_contratacion_temporal.numeracion_anual_asignada
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.numeracion_anual_asignada
    FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_total
    ON vec_contratacion_temporal.numeracion_anual_asignada
    TO vec_contratacion_temporal_propietario
    USING (true) WITH CHECK (true);
REVOKE ALL ON TABLE vec_contratacion_temporal.numeracion_anual_asignada
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_lector_resultado_cobertura,
    vec_contratacion_temporal_consultor_rrhh;

-- Número que se muestra: el asignado si existe; si no, el original. Solo la
-- invocan funciones del propietario.
CREATE FUNCTION vec_contratacion_temporal.numero_visible_vigente_v1(
    p_expediente_ref text,
    p_numero_original text
)
RETURNS text
LANGUAGE sql
STABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT COALESCE(
        (SELECT asignada.numero_visible
           FROM vec_contratacion_temporal.numeracion_anual_asignada asignada
          WHERE asignada.expediente_ref = p_expediente_ref),
        p_numero_original
    )
$funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.numero_visible_vigente_v1(text, text)
FROM PUBLIC;

-- Asigna el número anual a los expedientes técnicos que aún no lo tienen, en
-- orden de alta. Bloquea el puntero de versión de cada expediente mientras
-- registra la actuación, de modo que la versión anotada es la vigente. Devuelve
-- cuántos ha numerado; una segunda llamada devuelve 0.
CREATE FUNCTION vec_contratacion_temporal.asignar_numeracion_anual_anteriores_v1()
RETURNS integer
LANGUAGE plpgsql
VOLATILE
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '5s'
AS $funcion$
DECLARE
    v_fila record;
    v_nuevo text;
    v_anio integer;
    v_sufijo text;
    v_ahora timestamptz(6);
    v_secuencia numeric(20, 0);
    v_anterior text;
    v_payload bytea;
    v_evento text;
    v_operacion text;
    v_total integer := 0;
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'numeración anual no autorizada';
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended(
            'vec_contratacion_temporal:numeracion_anual_anteriores', 0
        )
    );
    FOR v_fila IN
        SELECT alta.expediente_ref, alta.numero_visible, actual.version
          FROM vec_contratacion_temporal.expediente_alta alta
          JOIN vec_contratacion_temporal.expediente_integral_actual actual
            ON actual.expediente_ref = alta.expediente_ref
         WHERE alta.numero_visible ~ '^[0-9]{4}/CT-[0-9a-f]{12,}$'
           AND NOT EXISTS (
               SELECT 1
                 FROM vec_contratacion_temporal.numeracion_anual_asignada asignada
                WHERE asignada.expediente_ref = alta.expediente_ref
           )
         ORDER BY alta.creada_en, alta.expediente_ref COLLATE "C"
           FOR UPDATE OF actual
    LOOP
        v_anio := pg_catalog.left(v_fila.numero_visible, 4)::integer;
        v_nuevo := vec_contratacion_temporal.siguiente_numero_visible_v1(v_anio);
        IF v_nuevo IS NULL
           OR v_nuevo ~ '^[0-9]{4}/CT-[0-9a-f]{12,}$'
           OR EXISTS (
               SELECT 1 FROM vec_contratacion_temporal.expediente_alta otra
                WHERE otra.numero_visible = v_nuevo
           ) THEN
            RAISE EXCEPTION USING ERRCODE = '23505',
                MESSAGE = 'número anual ya usado por otro expediente';
        END IF;
        v_sufijo := pg_catalog.left(pg_catalog.encode(pg_catalog.sha256(
            pg_catalog.convert_to('ct142:' || v_fila.expediente_ref, 'UTF8')
        ), 'hex'), 32);
        v_evento := 'evento:ct-numeracion-anual:' || v_sufijo;
        v_operacion := 'operacion:ct-numeracion-anual:' || v_sufijo;
        v_ahora := pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp());

        SELECT control.secuencia_outbox, control.cabeza_outbox_sha256
          INTO STRICT v_secuencia, v_anterior
          FROM vec_contratacion_temporal.control_cadenas_expediente_integral control
         WHERE control.control_id
           FOR UPDATE;
        IF v_secuencia >= 9007199254740991::numeric THEN
            RAISE EXCEPTION USING ERRCODE = '22003',
                MESSAGE = 'límite de outbox alcanzado';
        END IF;
        v_secuencia := v_secuencia + 1;
        v_payload := pg_catalog.convert_to(pg_catalog.jsonb_build_object(
            'esquema', 'vec.contratacion-temporal.numero-expediente-asignado.v1',
            'expediente_ref', v_fila.expediente_ref,
            'version_expediente', v_fila.version,
            'numero_anterior', v_fila.numero_visible,
            'numero_visible', v_nuevo,
            'motivo_clave', 'numeracion_anual_expediente_anterior',
            'actor_ref', 'sistema:contratacion_temporal:migracion:000142',
            'asignado_en', pg_catalog.to_char(
                v_ahora, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
        )::text, 'UTF8');
        INSERT INTO vec_contratacion_temporal.outbox_expediente_integral (
            evento_ref, secuencia, operacion_ref, expediente_ref,
            version_expediente, tipo_evento, payload_canonico,
            payload_huella_sha256, anterior_sha256, huella_sha256, registrada_en
        ) VALUES (
            v_evento, v_secuencia, v_operacion, v_fila.expediente_ref,
            v_fila.version, 'contratacion_temporal.numero_expediente_asignado',
            v_payload, pg_catalog.encode(pg_catalog.sha256(v_payload), 'hex'),
            v_anterior, pg_catalog.encode(pg_catalog.sha256(
                v_anterior::bytea || v_payload), 'hex'), v_ahora
        );
        UPDATE vec_contratacion_temporal.control_cadenas_expediente_integral
           SET secuencia_outbox = v_secuencia,
               cabeza_outbox_sha256 = pg_catalog.encode(pg_catalog.sha256(
                   v_anterior::bytea || v_payload), 'hex'),
               actualizada_en = v_ahora
         WHERE control_id;

        INSERT INTO vec_contratacion_temporal.numeracion_anual_asignada (
            expediente_ref, numero_anterior, numero_visible, anio,
            version_expediente, motivo_clave, actor_ref, operacion_ref,
            evento_ref, ejecutado_por, asignado_en
        ) VALUES (
            v_fila.expediente_ref, v_fila.numero_visible, v_nuevo, v_anio,
            v_fila.version, 'numeracion_anual_expediente_anterior',
            'sistema:contratacion_temporal:migracion:000142', v_operacion,
            v_evento, pg_catalog.left(session_user::text, 63), v_ahora
        );
        v_total := v_total + 1;
    END LOOP;
    RETURN v_total;
END
$funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.asignar_numeracion_anual_anteriores_v1()
FROM PUBLIC;

-- Las consultas muestran el número vigente. Cada función se reconstruye desde
-- su definición instalada, con huella previa y posterior fijadas, un único
-- fragmento sustituido y el mismo propietario, ACL y configuración.
DO $consultas$
DECLARE
    v_cambio record;
    v_antes record;
    v_despues record;
    v_definicion text;
    v_apariciones integer;
BEGIN
    FOR v_cambio IN
        SELECT *
          FROM (VALUES
            ('vec_contratacion_temporal.materializar_cuadro_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1)',
             '45a6f552b1544a74f717c6dbd424a348305f037b51efc19db075a0e226e879bb',
             'b5a878d579c79b5550cd2d42108c9a25d4c8db44efc1c536398a14418788951f',
             $a1$               publicada.numero_visible,
               publicada.version,$a1$,
             $n1$               vec_contratacion_temporal.numero_visible_vigente_v1(
                   publicada.expediente_ref, publicada.numero_visible
               ) AS numero_visible,
               publicada.version,$n1$),
            ('vec_contratacion_temporal.contar_totales_cuadro_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,text)',
             'ead046ef1e96d3ca2fe39171bbc94496ca10dda894402866de87ca19d12ef72f',
             'cf50cae432619bdbfa45a249639559374f9c2cc88771e4c1908e897a808464b0',
             $a2$               publicada.numero_visible, publicada.fase_clave,$a2$,
             $n2$               vec_contratacion_temporal.numero_visible_vigente_v1(
                   publicada.expediente_ref, publicada.numero_visible
               ) AS numero_visible, publicada.fase_clave,$n2$),
            ('vec_contratacion_temporal.materializar_detalle_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,numeric)',
             '0826469968ccba7c35bd6d3e7ca7e493aa871b1b753a7af929bb9c33f0893a2f',
             '6ee713225c5fbaa9052f451d6e0212cd44752aedb6c824269ceca52e80e3f77d',
             $a3$    v_resumen := ROW(
        v_fila.expediente_ref,
        v_fila.organizacion_ref,
        v_fila.numero_visible,$a3$,
             $n3$    v_resumen := ROW(
        v_fila.expediente_ref,
        v_fila.organizacion_ref,
        vec_contratacion_temporal.numero_visible_vigente_v1(
            v_fila.expediente_ref, v_fila.numero_visible
        ),$n3$),
            ('vec_contratacion_temporal.expedientes_centro_ct124(jsonb,text)',
             '8e42e37d9f3a3aa7637d6177b5206dd4d1f3bb8d54b626200f5606e18c7519cb',
             'f698ff4755a5b0b2c6f6070e6dc28cf7174dad2d061ce807942c7d0e85d8954b',
             $a4$c.expediente_ref, c.numero_visible, ac.version$a4$,
             $n4$c.expediente_ref, vec_contratacion_temporal.numero_visible_vigente_v1(c.expediente_ref, c.numero_visible) AS numero_visible, ac.version$n4$)
          ) AS cambio(firma, huella_antes, huella_despues, anterior, nuevo)
    LOOP
        SELECT p.oid, p.prosrc AS cuerpo, pg_catalog.pg_get_functiondef(p.oid) AS definicion,
               p.proacl AS acl, p.proowner AS propietario,
               p.proconfig AS configuracion, p.prosecdef AS definidor
          INTO v_antes
          FROM pg_catalog.pg_proc p
         WHERE p.oid = pg_catalog.to_regprocedure(v_cambio.firma)
           AND p.proowner = 'vec_contratacion_temporal_propietario'::regrole;
        IF NOT FOUND
           OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
                  v_antes.cuerpo, 'UTF8')), 'hex')
              IS DISTINCT FROM v_cambio.huella_antes THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'CT-000142: consulta incompatible ' || v_cambio.firma;
        END IF;
        v_apariciones := (pg_catalog.length(v_antes.definicion)
            - pg_catalog.length(pg_catalog.replace(
                v_antes.definicion, v_cambio.anterior, '')))
            / pg_catalog.length(v_cambio.anterior);
        IF v_apariciones <> 1 THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'CT-000142: fragmento ambiguo en ' || v_cambio.firma;
        END IF;
        v_definicion := pg_catalog.replace(
            v_antes.definicion, v_cambio.anterior, v_cambio.nuevo);
        EXECUTE v_definicion;
        SELECT p.prosrc AS cuerpo, pg_catalog.pg_get_functiondef(p.oid) AS definicion,
               p.proacl AS acl, p.proowner AS propietario,
               p.proconfig AS configuracion, p.prosecdef AS definidor
          INTO STRICT v_despues
          FROM pg_catalog.pg_proc p
         WHERE p.oid = v_antes.oid;
        IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
               v_despues.cuerpo, 'UTF8')), 'hex')
              IS DISTINCT FROM v_cambio.huella_despues
           OR v_despues.definicion IS DISTINCT FROM v_definicion
           OR v_despues.acl IS DISTINCT FROM v_antes.acl
           OR v_despues.propietario IS DISTINCT FROM v_antes.propietario
           OR v_despues.configuracion IS DISTINCT FROM v_antes.configuracion
           OR v_despues.definidor IS DISTINCT FROM v_antes.definidor THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'CT-000142: definición o permisos alterados en ' || v_cambio.firma;
        END IF;
    END LOOP;
END
$consultas$;

SELECT vec_contratacion_temporal.asignar_numeracion_anual_anteriores_v1();

DO $postcondicion$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.expediente_alta alta
         WHERE alta.numero_visible ~ '^[0-9]{4}/CT-[0-9a-f]{12,}$'
           AND NOT EXISTS (
               SELECT 1
                 FROM vec_contratacion_temporal.numeracion_anual_asignada asignada
                WHERE asignada.expediente_ref = alta.expediente_ref
           )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'CT-000142: quedan expedientes con número técnico';
    END IF;
    IF has_function_privilege('public',
           'vec_contratacion_temporal.numero_visible_vigente_v1(text,text)', 'EXECUTE')
       OR has_function_privilege('public',
           'vec_contratacion_temporal.asignar_numeracion_anual_anteriores_v1()', 'EXECUTE')
       OR has_table_privilege('public',
           'vec_contratacion_temporal.numeracion_anual_asignada', 'SELECT') THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'CT-000142: PUBLIC conserva privilegios';
    END IF;
END
$postcondicion$;
COMMIT;
