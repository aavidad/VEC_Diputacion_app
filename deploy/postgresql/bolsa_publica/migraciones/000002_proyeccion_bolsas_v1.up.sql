-- B10: proyeccion publica minimizada de las bolsas constituidas. Esta
-- migracion amplia la publicacion V2 sin modificar su historia instalada.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_publica:migracion:000002', 0)
);
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_publica:publicacion:v2', 0)
);

DO $prevalidacion$
BEGIN
    IF current_user <> 'vec_bolsa_publica_migrador'
       OR NOT pg_catalog.pg_has_role(current_user, 'vec_bolsa_publica_propietario', 'SET')
       OR NOT pg_catalog.pg_has_role(current_user, 'vec_bolsa_publica_publicacion_propietario', 'SET') THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'migracion de bolsas publicas rechazada: identidad incorrecta';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class AS objeto
          JOIN pg_catalog.pg_namespace AS esquema ON esquema.oid = objeto.relnamespace
         WHERE (esquema.nspname = 'vec_bolsa_publica_datos'
                AND objeto.relname IN ('bolsa_publica', 'posicion_bolsa_publica'))
            OR (esquema.nspname = 'vec_bolsa_publica_lectura'
                AND objeto.relname IN ('bolsas_v1', 'posiciones_bolsa_v1'))
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS objeto
          JOIN pg_catalog.pg_namespace AS esquema ON esquema.oid = objeto.pronamespace
         WHERE esquema.nspname = 'vec_bolsa_publica_publicacion'
           AND objeto.proname = 'publicar_proyeccion_v3'
           AND pg_catalog.oidvectortypes(objeto.proargtypes) = 'jsonb, jsonb, text'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'migracion de bolsas publicas rechazada: objetos B10 ya existen';
    END IF;
END
$prevalidacion$;

SET LOCAL ROLE vec_bolsa_publica_propietario;

CREATE FUNCTION vec_bolsa_publica_datos.grupos_b10_validos(p_grupos text[])
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_grupos IS NOT NULL
       AND cardinality(p_grupos) BETWEEN 1 AND 8
       AND cardinality(p_grupos) = cardinality(ARRAY(SELECT DISTINCT valor FROM unnest(p_grupos) AS valor))
       AND NOT EXISTS (
           SELECT 1 FROM unnest(p_grupos) AS valor
            WHERE valor IS NULL OR valor !~ '^[A-Z][0-9A-Z-]{0,7}$'
       )
$funcion$;
REVOKE ALL ON FUNCTION vec_bolsa_publica_datos.grupos_b10_validos(text[]) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_publica_datos.grupos_b10_validos(text[])
    TO vec_bolsa_publica_publicacion_propietario;

CREATE TABLE vec_bolsa_publica_datos.bolsa_publica (
    bolsa_ref text PRIMARY KEY CHECK (
        bolsa_ref ~ '^[a-z0-9][a-z0-9:._-]{2,159}$'
    ),
    categoria text NOT NULL CHECK (char_length(categoria) BETWEEN 1 AND 160),
    categoria_clave text NOT NULL CHECK (
        categoria_clave ~ '^[a-z0-9][a-z0-9-]{0,79}$'
    ),
    grupos text[] NOT NULL CHECK (vec_bolsa_publica_datos.grupos_b10_validos(grupos)),
    tipo_lista text NOT NULL CHECK (
        tipo_lista ~ '^[a-z][a-z0-9_-]{0,79}$'
    ),
    vigente_desde timestamptz(6) NOT NULL,
    vigente_hasta timestamptz(6),
    total bigint NOT NULL CHECK (total BETWEEN 0 AND 100000),
    CHECK (vigente_hasta IS NULL OR vigente_desde < vigente_hasta)
);

CREATE TABLE vec_bolsa_publica_datos.posicion_bolsa_publica (
    bolsa_ref text NOT NULL REFERENCES vec_bolsa_publica_datos.bolsa_publica(bolsa_ref)
        ON DELETE CASCADE,
    orden bigint NOT NULL CHECK (orden BETWEEN 1 AND 100000),
    documento_enmascarado text NOT NULL CHECK (documento_enmascarado ~ E'^\\*{3}[0-9]{4}\\*{2}$'),
    estado_clave text NOT NULL CHECK (estado_clave IN (
        'disponible', 'ocupado', 'no_disponible', 'excluido', 'renuncia_pendiente'
    )),
    PRIMARY KEY (bolsa_ref, orden)
);

ALTER TABLE vec_bolsa_publica_datos.bolsa_publica ENABLE ROW LEVEL SECURITY;
-- El propietario de las vistas debe poder proyectar con sus privilegios; la
-- cuenta lectora no recibe SELECT sobre tablas y solo entra por las vistas.
CREATE POLICY solo_propietario_b10 ON vec_bolsa_publica_datos.bolsa_publica
    TO vec_bolsa_publica_propietario
    USING (current_user = 'vec_bolsa_publica_propietario')
    WITH CHECK (current_user = 'vec_bolsa_publica_propietario');
CREATE POLICY solo_publicador_b10 ON vec_bolsa_publica_datos.bolsa_publica
    TO vec_bolsa_publica_publicacion_propietario
    USING (current_user = 'vec_bolsa_publica_publicacion_propietario')
    WITH CHECK (current_user = 'vec_bolsa_publica_publicacion_propietario');
CREATE POLICY lectura_por_vista_b10 ON vec_bolsa_publica_datos.bolsa_publica
    FOR SELECT TO vec_bolsa_publica_consulta USING (true);
ALTER TABLE vec_bolsa_publica_datos.posicion_bolsa_publica ENABLE ROW LEVEL SECURITY;
CREATE POLICY solo_propietario_b10 ON vec_bolsa_publica_datos.posicion_bolsa_publica
    TO vec_bolsa_publica_propietario
    USING (current_user = 'vec_bolsa_publica_propietario')
    WITH CHECK (current_user = 'vec_bolsa_publica_propietario');
CREATE POLICY solo_publicador_b10 ON vec_bolsa_publica_datos.posicion_bolsa_publica
    TO vec_bolsa_publica_publicacion_propietario
    USING (current_user = 'vec_bolsa_publica_publicacion_propietario')
    WITH CHECK (current_user = 'vec_bolsa_publica_publicacion_propietario');
CREATE POLICY lectura_por_vista_b10 ON vec_bolsa_publica_datos.posicion_bolsa_publica
    FOR SELECT TO vec_bolsa_publica_consulta USING (true);
REVOKE ALL ON vec_bolsa_publica_datos.bolsa_publica,
              vec_bolsa_publica_datos.posicion_bolsa_publica FROM PUBLIC;

CREATE TRIGGER invalidar_manifiesto_bolsa_publica
AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON vec_bolsa_publica_datos.bolsa_publica
FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_publica_datos.invalidar_manifiesto_v1();
CREATE TRIGGER invalidar_manifiesto_posicion_bolsa_publica
AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON vec_bolsa_publica_datos.posicion_bolsa_publica
FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_publica_datos.invalidar_manifiesto_v1();

GRANT INSERT ON vec_bolsa_publica_datos.bolsa_publica,
                vec_bolsa_publica_datos.posicion_bolsa_publica
    TO vec_bolsa_publica_publicacion_propietario;
GRANT DELETE ON vec_bolsa_publica_datos.bolsa_publica
    TO vec_bolsa_publica_publicacion_propietario;

SET LOCAL ROLE vec_bolsa_publica_publicacion_propietario;

CREATE FUNCTION vec_bolsa_publica_publicacion.publicar_proyeccion_v3(
    p_proyeccion_v2 jsonb,
    p_bolsas_v1 jsonb,
    p_ancla_manifiesto_sha256 text
)
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog
SET lock_timeout = '5s'
SET statement_timeout = '60s'
AS $funcion$
DECLARE
    v_bolsa jsonb;
    v_posicion jsonb;
    v_total_posiciones bigint := 0;
    v_bolsa_anterior text := '';
    v_orden_anterior bigint;
    v_grupo_anterior text := '';
    v_grupo text;
BEGIN
    -- La V3 es la unica frontera operativa. La V2 se invoca dentro de esta
    -- misma transaccion para que ancla, convocatorias y B10 cambien juntos.
    IF session_user <> 'vec_bolsa_publica_publicador_login'
       OR current_user <> 'vec_bolsa_publica_publicacion_propietario'
       OR NOT pg_catalog.pg_has_role(session_user, 'vec_bolsa_publica_publicador', 'MEMBER') THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'publicacion rechazada: identidad publicadora incorrecta';
    END IF;
    IF p_bolsas_v1 IS NULL
       OR pg_catalog.jsonb_typeof(p_bolsas_v1) <> 'object'
       OR p_bolsas_v1 - ARRAY['generado_en','bolsas'] <> '{}'::jsonb
       OR NOT (p_bolsas_v1 ?& ARRAY['generado_en','bolsas'])
       OR pg_catalog.jsonb_typeof(p_bolsas_v1->'generado_en') <> 'string'
       OR pg_catalog.jsonb_typeof(p_bolsas_v1->'bolsas') <> 'array'
       OR pg_catalog.jsonb_array_length(p_bolsas_v1->'bolsas') > 128
       OR pg_catalog.octet_length(p_bolsas_v1::text) > 67108864 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'publicacion rechazada: contrato de bolsas publicas invalido';
    END IF;
    IF p_bolsas_v1->>'generado_en' <> p_proyeccion_v2#>>'{fuente,actualizada_en}' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'publicacion rechazada: instantanea B10 distinta de la fuente';
    END IF;

    FOR v_bolsa IN SELECT value FROM pg_catalog.jsonb_array_elements(p_bolsas_v1->'bolsas') LOOP
        IF vec_bolsa_publica_publicacion.objeto_jsonb_exacto_v2(
               v_bolsa, ARRAY['bolsa_ref','categoria','categoria_clave','grupos','tipo_lista','vigente_desde','vigente_hasta','total','posiciones']
           ) IS DISTINCT FROM true
           OR pg_catalog.jsonb_typeof(v_bolsa->'bolsa_ref') <> 'string'
           OR pg_catalog.jsonb_typeof(v_bolsa->'categoria') <> 'string'
           OR pg_catalog.jsonb_typeof(v_bolsa->'categoria_clave') <> 'string'
           OR pg_catalog.jsonb_typeof(v_bolsa->'grupos') <> 'array'
           OR pg_catalog.jsonb_typeof(v_bolsa->'tipo_lista') <> 'string'
           OR pg_catalog.jsonb_typeof(v_bolsa->'vigente_desde') <> 'string'
           OR pg_catalog.jsonb_typeof(v_bolsa->'vigente_hasta') NOT IN ('string','null')
           OR pg_catalog.jsonb_typeof(v_bolsa->'total') <> 'number'
           OR (v_bolsa->>'total') !~ '^(0|[1-9][0-9]{0,5})$'
           OR (v_bolsa->>'total')::numeric > 100000
           OR pg_catalog.jsonb_typeof(v_bolsa->'posiciones') <> 'array'
           OR pg_catalog.jsonb_array_length(v_bolsa->'grupos') NOT BETWEEN 1 AND 8
           OR pg_catalog.jsonb_array_length(v_bolsa->'posiciones') <> (v_bolsa->>'total')::integer
           OR v_bolsa->>'bolsa_ref' <= v_bolsa_anterior THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'publicacion rechazada: bolsa publica invalida';
        END IF;
        v_grupo_anterior := '';
        FOR v_grupo IN SELECT value FROM pg_catalog.jsonb_array_elements_text(v_bolsa->'grupos') LOOP
            IF v_grupo IS NULL OR v_grupo !~ '^[A-Z][0-9A-Z-]{0,7}$' OR v_grupo <= v_grupo_anterior THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'publicacion rechazada: grupos de bolsa invalidos';
            END IF;
            v_grupo_anterior := v_grupo;
        END LOOP;
        v_orden_anterior := 0;
        FOR v_posicion IN SELECT value FROM pg_catalog.jsonb_array_elements(v_bolsa->'posiciones') LOOP
            IF vec_bolsa_publica_publicacion.objeto_jsonb_exacto_v2(
                   v_posicion, ARRAY['orden','documento_enmascarado','estado_clave']
               ) IS DISTINCT FROM true
               OR pg_catalog.jsonb_typeof(v_posicion->'orden') <> 'number'
               OR (v_posicion->>'orden') !~ '^[1-9][0-9]{0,5}$'
               OR (v_posicion->>'orden')::numeric > 100000
               OR pg_catalog.jsonb_typeof(v_posicion->'documento_enmascarado') <> 'string'
               OR pg_catalog.jsonb_typeof(v_posicion->'estado_clave') <> 'string'
               OR v_posicion->>'documento_enmascarado' !~ E'^\\*{3}[0-9]{4}\\*{2}$'
               OR v_posicion->>'estado_clave' NOT IN ('disponible','ocupado','no_disponible','excluido','renuncia_pendiente')
               OR (v_posicion->>'orden')::bigint <= v_orden_anterior THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'publicacion rechazada: posicion publica invalida';
            END IF;
            v_orden_anterior := (v_posicion->>'orden')::bigint;
        END LOOP;
        v_total_posiciones := v_total_posiciones + (v_bolsa->>'total')::bigint;
        IF v_total_posiciones > 1000000 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'publicacion rechazada: demasiadas posiciones publicas';
        END IF;
        v_bolsa_anterior := v_bolsa->>'bolsa_ref';
    END LOOP;

    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended('vec_bolsa_publica:publicacion:v2', 0)
    );
    DELETE FROM vec_bolsa_publica_datos.bolsa_publica;
    FOR v_bolsa IN SELECT value FROM pg_catalog.jsonb_array_elements(p_bolsas_v1->'bolsas') LOOP
        INSERT INTO vec_bolsa_publica_datos.bolsa_publica(
            bolsa_ref, categoria, categoria_clave, grupos, tipo_lista,
            vigente_desde, vigente_hasta, total
        ) VALUES (
            v_bolsa->>'bolsa_ref', v_bolsa->>'categoria', v_bolsa->>'categoria_clave',
            ARRAY(SELECT value FROM pg_catalog.jsonb_array_elements_text(v_bolsa->'grupos')),
            v_bolsa->>'tipo_lista', (v_bolsa->>'vigente_desde')::timestamptz,
            NULLIF(v_bolsa->>'vigente_hasta', '')::timestamptz, (v_bolsa->>'total')::bigint
        );
        INSERT INTO vec_bolsa_publica_datos.posicion_bolsa_publica(
            bolsa_ref, orden, documento_enmascarado, estado_clave
        )
        SELECT v_bolsa->>'bolsa_ref', (posicion->>'orden')::bigint,
               posicion->>'documento_enmascarado', posicion->>'estado_clave'
          FROM pg_catalog.jsonb_array_elements(v_bolsa->'posiciones') AS posicion;
    END LOOP;
    -- La V2 inserta fuente al final. Sus triggers ven el DML B10 anterior y el
    -- testigo final es exactamente p_ancla_manifiesto_sha256 para el conjunto.
    PERFORM vec_bolsa_publica_publicacion.publicar_proyeccion_v2(
        p_proyeccion_v2, p_ancla_manifiesto_sha256
    );
EXCEPTION
    WHEN data_exception OR integrity_constraint_violation THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'publicacion rechazada: contenido invalido';
END
$funcion$;
REVOKE ALL ON FUNCTION vec_bolsa_publica_publicacion.publicar_proyeccion_v3(jsonb, jsonb, text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_publica_publicacion.objeto_jsonb_exacto_v2(jsonb, text[])
    TO vec_bolsa_publica_publicacion_propietario;
REVOKE EXECUTE ON FUNCTION vec_bolsa_publica_publicacion.publicar_proyeccion_v2(jsonb, text)
    FROM vec_bolsa_publica_publicador_login;
GRANT EXECUTE ON FUNCTION vec_bolsa_publica_publicacion.publicar_proyeccion_v3(jsonb, jsonb, text)
    TO vec_bolsa_publica_publicador_login;

SET LOCAL ROLE vec_bolsa_publica_propietario;
CREATE VIEW vec_bolsa_publica_lectura.bolsas_v1
WITH (security_barrier = true, security_invoker = false) AS
SELECT bolsa_ref, categoria, categoria_clave, grupos, tipo_lista,
       vigente_desde, vigente_hasta, total
  FROM vec_bolsa_publica_datos.bolsa_publica;
CREATE VIEW vec_bolsa_publica_lectura.posiciones_bolsa_v1
WITH (security_barrier = true, security_invoker = false) AS
SELECT bolsa_ref, orden, documento_enmascarado, estado_clave
  FROM vec_bolsa_publica_datos.posicion_bolsa_publica;
REVOKE ALL ON vec_bolsa_publica_lectura.bolsas_v1,
              vec_bolsa_publica_lectura.posiciones_bolsa_v1 FROM PUBLIC;
GRANT SELECT ON vec_bolsa_publica_lectura.bolsas_v1,
                vec_bolsa_publica_lectura.posiciones_bolsa_v1
    TO vec_bolsa_publica_consulta;
COMMIT;
