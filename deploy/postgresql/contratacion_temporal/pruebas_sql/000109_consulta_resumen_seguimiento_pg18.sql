\set ON_ERROR_STOP on
-- Fixture desechable PG18.4. Ejecutar solo en base vacía de contenedor propio.
-- AD3 y CT43 se sustituyen por dobles explícitos: prueba la proyección, las
-- guardas de campos/ámbito y el orden del consumo; no acredita criptografía.
DO $$ BEGIN
    IF pg_catalog.current_setting('server_version_num') <> '180004'
       OR pg_catalog.to_regnamespace('vec_contratacion_temporal') IS NOT NULL THEN
        RAISE EXCEPTION 'CT109 requiere PG18.4 desechable y base vacía';
    END IF;
END $$;
CREATE ROLE vec_contratacion_temporal_propietario NOLOGIN NOINHERIT;
CREATE ROLE vec_contratacion_temporal_consultor_rrhh NOLOGIN INHERIT;
CREATE ROLE vec_ct109_runtime LOGIN INHERIT;
GRANT vec_contratacion_temporal_consultor_rrhh TO vec_ct109_runtime
    WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
GRANT CREATE ON DATABASE postgres TO vec_contratacion_temporal_propietario;
SET ROLE vec_contratacion_temporal_propietario;
CREATE SCHEMA vec_contratacion_temporal;
GRANT USAGE ON SCHEMA vec_contratacion_temporal
    TO vec_contratacion_temporal_consultor_rrhh;
CREATE TYPE vec_contratacion_temporal.alcance_consulta_rrhh_v1 AS (
    organizacion_ref text, clase_ambito text, ambito_ref text
);
CREATE TYPE vec_contratacion_temporal.consulta_detalle_rrhh_v1 AS (
    expediente_ref text, version_observada numeric
);
CREATE TYPE vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3 AS (
    capacidad_canonica bytea, decision_canonica bytea,
    motivo_canonico bytea, contexto_actor_canonico bytea,
    persona_version numeric, perfil_version numeric,
    payload_vec_ad_3 bytea, sobre_cose_sign_1 bytea,
    evidencia_verificacion bytea, raiz_publica_spki bytea
);
CREATE TYPE vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3 AS (
    decision_ref text, efecto_ref text, huella_efecto_sha256 text,
    consumo_huella_sha256 text, auditoria_ref text,
    auditoria_huella_sha256 text, consumida_en timestamptz,
    consumo_nuevo boolean
);
CREATE TYPE vec_contratacion_temporal.resumen_publicacion_rrhh_v1 AS (
    expediente_ref text, version numeric
);
CREATE TYPE vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1 AS (
    resumen vec_contratacion_temporal.resumen_publicacion_rrhh_v1
);
CREATE TYPE vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2 AS (
    generada_en timestamptz, total smallint, cursor_huella_sha256 text,
    alcance_huella_sha256 text, expediente_ref text,
    version_expediente numeric, contenido_huella_sha256 text,
    esquema text, acceso_ref text, secuencia numeric, anterior_sha256 text,
    huella_sha256 text, vinculo_identidad_huella_sha256 text,
    registrada_en timestamptz, auditoria_vec_ref text,
    auditoria_vec_huella_sha256 text, consumo_vec_huella_sha256 text,
    resultado_huella_sha256 text, recibo_sello_sha256 text
);
CREATE TYPE vec_contratacion_temporal.resultado_motor_detalle_rrhh_v1 AS (
    generada_en timestamptz,
    detalle vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1,
    cierre vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2
);
CREATE TABLE vec_contratacion_temporal.control_publicacion_rrhh (
    control boolean PRIMARY KEY, ultimo_corte numeric NOT NULL
);
INSERT INTO vec_contratacion_temporal.control_publicacion_rrhh VALUES (true, 2);
CREATE TABLE vec_contratacion_temporal.publicacion_version_rrhh (
    expediente_ref text, version numeric, corte_global numeric,
    organizacion_ref text, centro_ref text, unidad_ref text
);
INSERT INTO vec_contratacion_temporal.publicacion_version_rrhh VALUES
    ('exp:ct109', 1, 1, 'org:ct109', 'cen:ct109', 'uni:antigua'),
    ('exp:ct109', 2, 2, 'org:ct109', 'cen:ct109', 'uni:actual');
CREATE TABLE vec_contratacion_temporal.test_consumos (n integer NOT NULL);
INSERT INTO vec_contratacion_temporal.test_consumos VALUES (0);
GRANT SELECT ON vec_contratacion_temporal.test_consumos
    TO vec_contratacion_temporal_consultor_rrhh;
CREATE FUNCTION vec_contratacion_temporal.canon_alcance_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1)
RETURNS bytea LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog
AS $$ SELECT '\x01'::bytea $$;
CREATE FUNCTION vec_contratacion_temporal.canon_consulta_detalle_rrhh_v1(
    p vec_contratacion_temporal.consulta_detalle_rrhh_v1)
RETURNS bytea LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog
AS $$ SELECT pg_catalog.convert_to(
    'fixture:' || p.expediente_ref || ':' || p.version_observada::text,
    'UTF8') $$;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.canon_consulta_detalle_rrhh_v1(
    vec_contratacion_temporal.consulta_detalle_rrhh_v1)
    TO vec_contratacion_temporal_consultor_rrhh;
CREATE FUNCTION vec_contratacion_temporal.acreditar_contexto_motor_consultas_rrhh_v1(
    p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    p_material vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3)
RETURNS void LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog
AS $$ BEGIN
    -- Centinela: si CT109 llega aquí con un campo excesivo, la prueba recibe
    -- 40001, distinto del 42501 que exige la guarda previa de CT109.
    IF pg_catalog.octet_length(p_material.capacidad_canonica)>32768
       OR pg_catalog.octet_length(p_material.decision_canonica)>524288
       OR pg_catalog.octet_length(p_material.contexto_actor_canonico)>262144
       OR p_material.persona_version>9007199254740991::numeric THEN
        RAISE EXCEPTION USING ERRCODE='40001', MESSAGE='guarda O(1) omitida';
    END IF;
END $$;
CREATE FUNCTION vec_contratacion_temporal.consumir_autorizacion_motor_consultas_rrhh_v1(
    p_tipo text, p_material vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3)
RETURNS vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog
AS $$
DECLARE c jsonb; d jsonb;
BEGIN
    IF p_tipo <> 'detalle' THEN RAISE EXCEPTION 'tipo inesperado'; END IF;
    c := pg_catalog.convert_from(p_material.capacidad_canonica,'UTF8')::jsonb;
    d := pg_catalog.convert_from(p_material.decision_canonica,'UTF8')::jsonb;
    UPDATE vec_contratacion_temporal.test_consumos SET n=n+1;
    RETURN ROW(d->>'decision_ref', c->>'efecto_ref',
        c->>'huella_efecto_sha256', pg_catalog.repeat('1',64),
        'aud:ct109', pg_catalog.repeat('2',64),
        pg_catalog.date_trunc('microseconds',pg_catalog.clock_timestamp()),true
    )::vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3;
END $$;
CREATE FUNCTION vec_contratacion_temporal.motor_consultar_detalle_rrhh_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3)
RETURNS vec_contratacion_temporal.resultado_motor_detalle_rrhh_v1
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog
AS $$ BEGIN RAISE EXCEPTION USING ERRCODE='40001', MESSAGE='motor alcanzado'; END $$;
SET check_function_bodies = off;
\ir ../migraciones/000045_componentes/030_fachada_detalle.sql
RESET check_function_bodies;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_contratacion_temporal_consultor_rrhh;
RESET ROLE;
\ir ../migraciones/000109_consulta_resumen_seguimiento.up.sql

\connect postgres vec_ct109_runtime
SET timezone='UTC';
CREATE TEMP TABLE ct109_caso AS
WITH entrada AS (
    SELECT ROW('org:ct109','organizacion','org:ct109')::
               vec_contratacion_temporal.alcance_consulta_rrhh_v1 AS alcance,
           ROW('exp:ct109',2)::
               vec_contratacion_temporal.consulta_detalle_rrhh_v1 AS consulta
), huella AS (
    SELECT entrada.*,
       pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
         '{"ambitos":{"ambito_ref":"org:ct109","clase_ambito":"organizacion",'
         || '"organizacion_ref":"org:ct109"},"atributos":{"consulta_dominio":"'
         || 'vec.contratacion_temporal.consulta_rrhh.detalle.v1'
         || '","consulta_huella_sha256":"'
         || pg_catalog.encode(pg_catalog.sha256(
              vec_contratacion_temporal.canon_consulta_detalle_rrhh_v1(consulta)
            ),'hex') || '"}}', 'UTF8')), 'hex') AS contexto_huella
    FROM entrada
), decision AS (
    SELECT huella.*,
       pg_catalog.convert_to(pg_catalog.jsonb_build_object(
           'decision_ref','dec:ct109','principal_id','per:ct109',
           'perfil_activo_ref','perfil:ct109','garantia_minima','sustancial',
           'version_rol_ref','rol:rrhh_interno_certificado_seguimiento_ct_123:v1',
           'accion','contratacion_temporal.expediente.consultar',
           'modulo_id','contratacion_temporal',
           'tipo_recurso','expediente_contratacion_temporal',
           'finalidad','tramitacion_expediente_contratacion_temporal',
           'recurso_ref','exp:ct109',
           'contexto_recurso_huella_sha256',contexto_huella,
           'campos_permitidos',
           ('["actuaciones","alcance","eficacia_administrativa",'
           || '"ejercicio_sintetico","esquema","estado_clave",'
           || '"expediente_ref","firma_oficial","periodo",'
           || '"recibo_incorporacion_ref","registrado_en","seguimiento_ref",'
           || '"version_expediente","version_seguimiento"]')::jsonb
       )::text,'UTF8') AS decision_bytes
    FROM huella
)
SELECT decision.*,
   pg_catalog.convert_to(pg_catalog.jsonb_build_object(
       'audiencia_consumo',
       'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1',
       'operacion','contratacion_temporal.expediente.consultar',
       'efecto_ref','exp:ct109',
       'huella_efecto_sha256',contexto_huella,
       'huella_decision_sha256',
       pg_catalog.encode(pg_catalog.sha256(decision_bytes),'hex'),
       'relleno',pg_catalog.repeat('x',512)
   )::text,'UTF8') AS capacidad_bytes,
   pg_catalog.convert_to(pg_catalog.jsonb_build_object(
       'principal_ref','per:ct109','perfil_activo_ref','perfil:ct109',
       'persona_version','1','perfil_version','1')::text,'UTF8')
       AS contexto_actor_bytes
FROM decision;

BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
DO $probar$
DECLARE v record; v_n integer; d bytea; c bytea; fallo boolean;
        decision_j jsonb; capacidad_j jsonb; caso integer;
        capacidad_larga bytea; decision_larga bytea;
        contexto_largo bytea;
BEGIN
    SELECT * INTO STRICT v FROM ct109_caso;
    SELECT * INTO STRICT v FROM vec_contratacion_temporal
       .consultar_resumen_seguimiento_rrhh_atestado_v1(
        v.alcance,v.consulta,'uni:actual',v.capacidad_bytes,
        v.decision_bytes,'\x01'::bytea,v.contexto_actor_bytes,
        1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,
        pg_catalog.decode(pg_catalog.repeat('01',44),'hex'));
    IF v.expediente_ref <> 'exp:ct109' OR v.version_expediente <> 2
       OR v.organizacion_ref <> 'org:ct109' OR v.unidad_ref <> 'uni:actual'
       OR v.auditoria_vec_ref <> 'aud:ct109'
       OR v.consumo_vec_huella_sha256 <> pg_catalog.repeat('1',64) THEN
        RAISE EXCEPTION 'CT109 salida positiva incorrecta';
    END IF;
    SELECT t.n INTO STRICT v_n FROM vec_contratacion_temporal.test_consumos t;
    IF v_n <> 1 THEN RAISE EXCEPTION 'CT109 consumo positivo incorrecto'; END IF;

    capacidad_larga := pg_catalog.convert_to(
        '{"relleno":"'||pg_catalog.repeat('x',32768)||'"}','UTF8');
    decision_larga := pg_catalog.convert_to(
        '{"relleno":"'||pg_catalog.repeat('x',524288)||'"}','UTF8');
    contexto_largo := pg_catalog.convert_to(
        '{"relleno":"'||pg_catalog.repeat('x',262144)||'"}','UTF8');
    FOR caso IN 1..4 LOOP
        fallo := false;
        BEGIN
            PERFORM * FROM vec_contratacion_temporal
              .consultar_resumen_seguimiento_rrhh_atestado_v1(
                (SELECT alcance FROM ct109_caso),
                (SELECT consulta FROM ct109_caso),'uni:actual',
                CASE WHEN caso=1 THEN capacidad_larga
                     ELSE (SELECT capacidad_bytes FROM ct109_caso) END,
                CASE WHEN caso=2 THEN decision_larga
                     ELSE (SELECT decision_bytes FROM ct109_caso) END,
                '\x01'::bytea,
                CASE WHEN caso=3 THEN contexto_largo
                     ELSE (SELECT contexto_actor_bytes FROM ct109_caso) END,
                CASE WHEN caso=4 THEN 9007199254740992::numeric ELSE 1 END,
                1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,
                pg_catalog.decode(pg_catalog.repeat('01',44),'hex'));
        EXCEPTION WHEN SQLSTATE '42501' THEN fallo := true;
        END;
        IF NOT fallo THEN
            RAISE EXCEPTION 'CT109 aceptó material excesivo en caso %',caso;
        END IF;
    END LOOP;
    SELECT t.n INTO STRICT v_n FROM vec_contratacion_temporal.test_consumos t;
    IF v_n <> 1 THEN RAISE EXCEPTION 'CT109 consumió material excesivo'; END IF;

    SELECT decision_bytes, capacidad_bytes INTO STRICT d,c FROM ct109_caso;
    d := pg_catalog.convert_to(
        pg_catalog.jsonb_set(pg_catalog.convert_from(d,'UTF8')::jsonb,
            '{campos_permitidos}', '["dato_personal"]'::jsonb)::text,'UTF8');
    c := pg_catalog.convert_to(
        pg_catalog.jsonb_set(pg_catalog.convert_from(c,'UTF8')::jsonb,
            '{huella_decision_sha256}',
            pg_catalog.to_jsonb(pg_catalog.encode(pg_catalog.sha256(d),'hex'))
        )::text,'UTF8');
    fallo := false;
    BEGIN
        PERFORM * FROM vec_contratacion_temporal
          .consultar_resumen_seguimiento_rrhh_atestado_v1(
            (SELECT alcance FROM ct109_caso),
            (SELECT consulta FROM ct109_caso),'uni:actual',c,d,
            '\x01'::bytea,(SELECT contexto_actor_bytes FROM ct109_caso),
            1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,
            pg_catalog.decode(pg_catalog.repeat('01',44),'hex'));
    EXCEPTION WHEN SQLSTATE '42501' THEN fallo := true;
    END;
    IF NOT fallo THEN RAISE EXCEPTION 'CT109 aceptó campos ajenos'; END IF;
    SELECT t.n INTO STRICT v_n FROM vec_contratacion_temporal.test_consumos t;
    IF v_n <> 1 THEN RAISE EXCEPTION 'CT109 consumió antes de cotejar campos'; END IF;

    fallo := false;
    BEGIN
        PERFORM * FROM vec_contratacion_temporal
          .consultar_resumen_seguimiento_rrhh_atestado_v1(
            (SELECT alcance FROM ct109_caso),
            (SELECT consulta FROM ct109_caso),'uni:antigua',
            (SELECT capacidad_bytes FROM ct109_caso),
            (SELECT decision_bytes FROM ct109_caso),
            '\x01'::bytea,(SELECT contexto_actor_bytes FROM ct109_caso),
            1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,
            pg_catalog.decode(pg_catalog.repeat('01',44),'hex'));
    EXCEPTION WHEN SQLSTATE '42501' THEN fallo := true;
    END;
    IF NOT fallo THEN RAISE EXCEPTION 'CT109 aceptó unidad histórica'; END IF;
    SELECT t.n INTO STRICT v_n FROM vec_contratacion_temporal.test_consumos t;
    IF v_n <> 1 THEN RAISE EXCEPTION 'CT109 no revirtió consumo fallido'; END IF;

    fallo := false;
    BEGIN
        PERFORM * FROM vec_contratacion_temporal
          .consultar_detalle_rrhh_atestado_v1(
            (SELECT alcance FROM ct109_caso),
            (SELECT consulta FROM ct109_caso),
            (SELECT capacidad_bytes FROM ct109_caso),
            (SELECT decision_bytes FROM ct109_caso),
            '\x01'::bytea,(SELECT contexto_actor_bytes FROM ct109_caso),
            1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,
            pg_catalog.decode(pg_catalog.repeat('01',44),'hex'));
    EXCEPTION WHEN SQLSTATE '42501' THEN fallo := true;
    END;
    IF NOT fallo THEN RAISE EXCEPTION 'CT45 expuso detalle con campos limitados'; END IF;
    SELECT t.n INTO STRICT v_n FROM vec_contratacion_temporal.test_consumos t;
    IF v_n <> 1 THEN RAISE EXCEPTION 'CT45 consumió antes de cotejar campos'; END IF;

    -- Con garantía alta, los 14 campos también deben ser insuficientes.
    -- Con [] y rol interno, el veto nominal prevalece. Una decisión alta de
    -- rol distinto llega al doble del motor (40001), probando la ubicación
    -- exacta de las guardas y la conservación del camino histórico.
    FOR caso IN 1..3 LOOP
        SELECT pg_catalog.convert_from(decision_bytes,'UTF8')::jsonb,
               pg_catalog.convert_from(capacidad_bytes,'UTF8')::jsonb
          INTO STRICT decision_j, capacidad_j FROM ct109_caso;
        decision_j := pg_catalog.jsonb_set(
            decision_j,'{garantia_minima}','"alto"'::jsonb);
        IF caso >= 2 THEN
            decision_j := pg_catalog.jsonb_set(
                decision_j,'{campos_permitidos}','[]'::jsonb);
        END IF;
        IF caso = 3 THEN
            decision_j := pg_catalog.jsonb_set(
                decision_j,'{version_rol_ref}',
                '"rol:rrhh_desarrollo:v1"'::jsonb);
        END IF;
        d := pg_catalog.convert_to(decision_j::text,'UTF8');
        capacidad_j := pg_catalog.jsonb_set(capacidad_j,
            '{huella_decision_sha256}',
            pg_catalog.to_jsonb(pg_catalog.encode(pg_catalog.sha256(d),'hex')));
        c := pg_catalog.convert_to(capacidad_j::text,'UTF8');
        fallo := false;
        BEGIN
            PERFORM * FROM vec_contratacion_temporal
              .consultar_detalle_rrhh_atestado_v1(
                (SELECT alcance FROM ct109_caso),
                (SELECT consulta FROM ct109_caso),c,d,
                '\x01'::bytea,(SELECT contexto_actor_bytes FROM ct109_caso),
                1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,
                pg_catalog.decode(pg_catalog.repeat('01',44),'hex'));
        EXCEPTION
            WHEN SQLSTATE '42501' THEN fallo := (caso IN (1,2));
            WHEN SQLSTATE '40001' THEN fallo := (caso = 3);
        END;
        IF NOT fallo THEN
            RAISE EXCEPTION 'CT45 guarda incorrecta en caso %',caso;
        END IF;
    END LOOP;
END
$probar$;
COMMIT;
SELECT 'CT109 fixture sintético: OK' AS resultado;
