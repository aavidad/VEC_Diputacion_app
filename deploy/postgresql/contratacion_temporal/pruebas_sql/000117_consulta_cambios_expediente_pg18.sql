\set ON_ERROR_STOP on
-- Petición RRHH p.4 (000117). Fixture desechable PG18.4: ejecutar solo en
-- base vacía de contenedor propio. AD3 se sustituye por dobles explícitos,
-- como en la prueba de 000109: prueba la comparación entre versiones, la
-- minimización, las guardas de permiso y el orden del consumo; no acredita
-- criptografía.
DO $$ BEGIN
    IF pg_catalog.current_setting('server_version_num') <> '180004'
       OR pg_catalog.to_regnamespace('vec_contratacion_temporal') IS NOT NULL THEN
        RAISE EXCEPTION 'CT117 requiere PG18.4 desechable y base vacía';
    END IF;
END $$;
CREATE ROLE vec_contratacion_temporal_propietario NOLOGIN NOINHERIT;
CREATE ROLE vec_contratacion_temporal_consultor_rrhh NOLOGIN INHERIT;
CREATE ROLE vec_ct117_runtime LOGIN INHERIT;
GRANT vec_contratacion_temporal_consultor_rrhh TO vec_ct117_runtime
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
    ('exp:ct117', 1, 1, 'org:ct117', 'cen:ct117', 'uni:antigua'),
    ('exp:ct117', 2, 2, 'org:ct117', 'cen:ct117', 'uni:actual');
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
    -- Centinela: si CT117 llega aquí con un campo excesivo, la prueba recibe
    -- 40001, distinto del 42501 que exige la guarda previa de CT117.
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
        'aud:ct117', pg_catalog.repeat('2',64),
        pg_catalog.date_trunc('microseconds',pg_catalog.clock_timestamp()),true
    )::vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3;
END $$;
-- Precondición de 000117: la consulta mínima de 000109 ya instalada.
CREATE FUNCTION vec_contratacion_temporal.consultar_resumen_seguimiento_rrhh_atestado_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
RETURNS void LANGUAGE sql AS $$ SELECT $$;
CREATE TABLE vec_contratacion_temporal.expediente_version_integral (
    expediente_ref text NOT NULL, version numeric(20,0) NOT NULL,
    agregado_json jsonb NOT NULL, origen_version text NOT NULL,
    operacion_ref text NOT NULL, registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (expediente_ref, version)
);
ALTER TABLE vec_contratacion_temporal.expediente_version_integral ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.expediente_version_integral FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario ON vec_contratacion_temporal.expediente_version_integral
    TO vec_contratacion_temporal_propietario USING (true) WITH CHECK (true);
INSERT INTO vec_contratacion_temporal.expediente_version_integral VALUES
 ('exp:ct117', 1, '{"version":1,"referencia":"exp:ct117","solicitud":{"detalle":"Texto libre con 600123456","grupo_subgrupo":"C2","centro_ref":"centro:ct117:1","fecha":"2027-02-01","vacio":"","cero":"0001-01-01T00:00:00Z","huella_sha256":"aa"},"actuaciones":[{"secuencia":1}],"actualizado_en":"2026-09-25T08:00:00Z"}', 'alta_o2', 'op:1', '2026-09-25 08:00:00+00'),
 ('exp:ct117', 2, '{"version":2,"referencia":"exp:ct117","solicitud":{"detalle":"Texto libre corregido","grupo_subgrupo":"C1","centro_ref":"centro:ct117:1","fecha":"2027-02-01T00:00:00Z","huella_sha256":"bb"},"analisis":{"porcentaje_jornada":10000,"causa_clave":"necesidad_temporal","observaciones":"Llamar al 958000000","periodo":{"inicio":"2027-01-01T00:00:00Z"},"actuacion_registro":{"secuencia":2}},"actuaciones":[{"secuencia":1},{"secuencia":2}]}', 'analisis_o3', 'op:2', '2026-09-25 09:00:00+00'),
 ('exp:ct117', 3, '{"version":3,"referencia":"exp:ct117","solicitud":{"detalle":"Texto libre corregido","grupo_subgrupo":"C1","centro_ref":"centro:ct117:2"},"analisis":{"porcentaje_jornada":5000,"causa_clave":"necesidad_temporal","observaciones":"Llamar al 958000000","periodo":{"inicio":"2027-01-01T00:00:00Z"}},"actuaciones":[{"secuencia":1},{"secuencia":2},{"secuencia":3}]}', 'cobertura_o4', 'op:3', '2026-09-25 10:00:00+00');
-- Expediente con 600 cambios de código: la respuesta se recorta y lo indica.
INSERT INTO vec_contratacion_temporal.publicacion_version_rrhh VALUES
    ('exp:ct117b', 2, 2, 'org:ct117', 'cen:ct117', 'uni:actual');
INSERT INTO vec_contratacion_temporal.expediente_version_integral
SELECT 'exp:ct117b', 1, '{"version":1}'::jsonb, 'alta_o2', 'op:b1', timestamptz '2026-09-25 08:00:00+00'
UNION ALL
SELECT 'exp:ct117b', 2, pg_catalog.jsonb_build_object('version', 2, 'analisis',
         pg_catalog.jsonb_object_agg(pg_catalog.format('c%s_clave', pg_catalog.lpad(i::text, 3, '0')), 'codigo_nuevo')),
       'analisis_o3', 'op:b2', timestamptz '2026-09-25 09:00:00+00'
  FROM pg_catalog.generate_series(1, 600) i;
RESET ROLE;
\ir ../migraciones/000117_consulta_cambios_expediente.up.sql

-- Minimización por lista cerrada de campos: ningún identificador ni texto
-- libre sale en claro, ni siquiera con forma de referencia o de código.
DO $minimizacion$
DECLARE r record;
BEGIN
    FOR r IN SELECT * FROM (VALUES
        ('solicitud.persona_ref', '"dni:12345678Z"'::jsonb, '*protegido'),
        ('solicitud.contacto_ref', '"tel:600123456"'::jsonb, '*protegido'),
        ('solicitud.titular_ref', '"nif:B12345678"'::jsonb, '*protegido'),
        ('solicitud.nombre', '"Lucía"'::jsonb, '*protegido'),
        ('solicitud.estado', '"Lucía García"'::jsonb, '*protegido'),
        ('solicitud.causa_clave', '"expediente_12345"'::jsonb, '*protegido'),
        ('solicitud.detalle', '"Texto libre"'::jsonb, '*protegido'),
        ('solicitud.telefono', '600123456'::jsonb, '*protegido'),
        ('persona.fecha_nacimiento', '"1990-05-01"'::jsonb, '*protegido'),
        ('solicitud.inicio', '"no es fecha"'::jsonb, '*protegido'),
        ('solicitud.lista[2]', '"a"'::jsonb, '*protegido'),
        ('solicitud.centro_ref', '"centro:ct117:1"'::jsonb, 'centro:ct117:1'),
        ('solicitud.causa_clave', '"necesidad_temporal"'::jsonb, 'necesidad_temporal'),
        ('coste_previsto.centimos', '123456'::jsonb, '123456'),
        ('fiscalizacion.ginpix_numero', '"2027000123"'::jsonb, '2027000123'),
        ('solicitud.expediente_numero', '12345678'::jsonb, '*protegido'),
        ('solicitud.numero_telefono', '600123456'::jsonb, '*protegido'),
        ('solicitud.numero_visible', '"2027/000123"'::jsonb, '*protegido'),
        ('solicitud.numero_visible', '2027000123'::jsonb, '2027000123'),
        ('declaracion_rc.numero', '"220270001"'::jsonb, '220270001'),
        ('solicitud.persona_ref', '"persona:12345678Z"'::jsonb, '*protegido'),
        ('solicitud.persona_ref', '"persona:ct:x1234567l"'::jsonb, '*protegido'),
        ('solicitud.candidato_ref', '"candidato:Y1234567Z:ct"'::jsonb, '*protegido'),
        ('solicitud.expediente_ref', '"exp:3f2a1b4c-12ab-4cde-9f00-123456789abc"'::jsonb, 'exp:3f2a1b4c-12ab-4cde-9f00-123456789abc'),
        ('analisis.periodo.fin', '"2027-03-31"'::jsonb, '2027-03-31T00:00:00.000000Z'),
        ('analisis.urgente', 'true'::jsonb, 'true')) AS t(ruta, valor, esperado)
    LOOP
        IF vec_contratacion_temporal.valor_traza_cambio_v1(r.ruta, r.valor) IS DISTINCT FROM r.esperado THEN
            RAISE EXCEPTION 'CT117 minimización de % (%): %', r.ruta, r.valor,
                vec_contratacion_temporal.valor_traza_cambio_v1(r.ruta, r.valor);
        END IF;
    END LOOP;
END
$minimizacion$;

\connect postgres vec_ct117_runtime
SET timezone='UTC';
CREATE TEMP TABLE ct117_caso AS
WITH entrada AS (
    SELECT ROW('org:ct117','organizacion','org:ct117')::
               vec_contratacion_temporal.alcance_consulta_rrhh_v1 AS alcance,
           ROW('exp:ct117',0)::
               vec_contratacion_temporal.consulta_detalle_rrhh_v1 AS consulta
), huella AS (
    SELECT entrada.*,
       pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
         '{"ambitos":{"ambito_ref":"org:ct117","clase_ambito":"organizacion",'
         || '"organizacion_ref":"org:ct117"},"atributos":{"consulta_dominio":"'
         || 'vec.contratacion_temporal.consulta_rrhh.detalle.v1'
         || '","consulta_huella_sha256":"'
         || pg_catalog.encode(pg_catalog.sha256(
              vec_contratacion_temporal.canon_consulta_detalle_rrhh_v1(consulta)
            ),'hex') || '"}}', 'UTF8')), 'hex') AS contexto_huella
    FROM entrada
), decision AS (
    SELECT huella.*,
       pg_catalog.convert_to(pg_catalog.jsonb_build_object(
           'decision_ref','dec:ct117','principal_id','per:ct117',
           'perfil_activo_ref','perfil:ct117','garantia_minima','alto',
           'version_rol_ref','rol:rrhh_desarrollo:v1',
           'accion','contratacion_temporal.expediente.consultar',
           'modulo_id','contratacion_temporal',
           'tipo_recurso','expediente_contratacion_temporal',
           'finalidad','tramitacion_expediente_contratacion_temporal',
           'recurso_ref','exp:ct117',
           'contexto_recurso_huella_sha256',contexto_huella,
           'campos_permitidos','[]'::jsonb
       )::text,'UTF8') AS decision_bytes
    FROM huella
)
SELECT decision.*,
   pg_catalog.convert_to(pg_catalog.jsonb_build_object(
       'audiencia_consumo',
       'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1',
       'operacion','contratacion_temporal.expediente.consultar',
       'efecto_ref','exp:ct117',
       'huella_efecto_sha256',contexto_huella,
       'huella_decision_sha256',
       pg_catalog.encode(pg_catalog.sha256(decision_bytes),'hex'),
       'relleno',pg_catalog.repeat('x',512)
   )::text,'UTF8') AS capacidad_bytes,
   pg_catalog.convert_to(pg_catalog.jsonb_build_object(
       'principal_ref','per:ct117','perfil_activo_ref','perfil:ct117',
       'persona_version','1','perfil_version','1')::text,'UTF8')
       AS contexto_actor_bytes
FROM decision;

CREATE TEMP TABLE ct117_caso_b AS
WITH entrada AS (
    SELECT ROW('org:ct117','organizacion','org:ct117')::
               vec_contratacion_temporal.alcance_consulta_rrhh_v1 AS alcance,
           ROW('exp:ct117b',0)::
               vec_contratacion_temporal.consulta_detalle_rrhh_v1 AS consulta
), huella AS (
    SELECT entrada.*,
       pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
         '{"ambitos":{"ambito_ref":"org:ct117","clase_ambito":"organizacion",'
         || '"organizacion_ref":"org:ct117"},"atributos":{"consulta_dominio":"'
         || 'vec.contratacion_temporal.consulta_rrhh.detalle.v1'
         || '","consulta_huella_sha256":"'
         || pg_catalog.encode(pg_catalog.sha256(
              vec_contratacion_temporal.canon_consulta_detalle_rrhh_v1(consulta)
            ),'hex') || '"}}', 'UTF8')), 'hex') AS contexto_huella
    FROM entrada
), decision AS (
    SELECT huella.*,
       pg_catalog.convert_to(pg_catalog.jsonb_build_object(
           'decision_ref','dec:ct117','principal_id','per:ct117',
           'perfil_activo_ref','perfil:ct117','garantia_minima','alto',
           'version_rol_ref','rol:rrhh_desarrollo:v1',
           'accion','contratacion_temporal.expediente.consultar',
           'modulo_id','contratacion_temporal',
           'tipo_recurso','expediente_contratacion_temporal',
           'finalidad','tramitacion_expediente_contratacion_temporal',
           'recurso_ref','exp:ct117b',
           'contexto_recurso_huella_sha256',contexto_huella,
           'campos_permitidos','[]'::jsonb
       )::text,'UTF8') AS decision_bytes
    FROM huella
)
SELECT decision.*,
   pg_catalog.convert_to(pg_catalog.jsonb_build_object(
       'audiencia_consumo',
       'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1',
       'operacion','contratacion_temporal.expediente.consultar',
       'efecto_ref','exp:ct117b',
       'huella_efecto_sha256',contexto_huella,
       'huella_decision_sha256',
       pg_catalog.encode(pg_catalog.sha256(decision_bytes),'hex'),
       'relleno',pg_catalog.repeat('x',512)
   )::text,'UTF8') AS capacidad_bytes,
   pg_catalog.convert_to(pg_catalog.jsonb_build_object(
       'principal_ref','per:ct117','perfil_activo_ref','perfil:ct117',
       'persona_version','1','perfil_version','1')::text,'UTF8')
       AS contexto_actor_bytes
FROM decision;

BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
DO $probar$
DECLARE v record; v_n integer; d bytea; c bytea; fallo boolean; caso integer;
        decision_j jsonb; capacidad_j jsonb; esperado jsonb;
BEGIN
    SELECT * INTO STRICT v FROM ct117_caso;
    SELECT * INTO STRICT v FROM vec_contratacion_temporal
       .consultar_cambios_expediente_rrhh_atestado_v1(
        v.alcance,v.consulta,v.capacidad_bytes,
        v.decision_bytes,'\x01'::bytea,v.contexto_actor_bytes,
        1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,
        pg_catalog.decode(pg_catalog.repeat('01',44),'hex'));
    -- La publicación visible es la versión 2: la 3 no aparece.
    esperado := pg_catalog.jsonb_build_array(
      pg_catalog.jsonb_build_object('version_expediente',2,'registrada_en','2026-09-25T09:00:00.000000Z','origen_version','analisis_o3','operacion_ref','op:2','ruta','analisis.causa_clave','valor_anterior',NULL,'valor_nuevo','necesidad_temporal'),
      pg_catalog.jsonb_build_object('version_expediente',2,'registrada_en','2026-09-25T09:00:00.000000Z','origen_version','analisis_o3','operacion_ref','op:2','ruta','analisis.observaciones','valor_anterior',NULL,'valor_nuevo','*protegido'),
      pg_catalog.jsonb_build_object('version_expediente',2,'registrada_en','2026-09-25T09:00:00.000000Z','origen_version','analisis_o3','operacion_ref','op:2','ruta','analisis.periodo.inicio','valor_anterior',NULL,'valor_nuevo','2027-01-01T00:00:00.000000Z'),
      pg_catalog.jsonb_build_object('version_expediente',2,'registrada_en','2026-09-25T09:00:00.000000Z','origen_version','analisis_o3','operacion_ref','op:2','ruta','analisis.porcentaje_jornada','valor_anterior',NULL,'valor_nuevo','10000'),
      pg_catalog.jsonb_build_object('version_expediente',2,'registrada_en','2026-09-25T09:00:00.000000Z','origen_version','analisis_o3','operacion_ref','op:2','ruta','solicitud.detalle','valor_anterior','*protegido','valor_nuevo','*protegido'),
      pg_catalog.jsonb_build_object('version_expediente',2,'registrada_en','2026-09-25T09:00:00.000000Z','origen_version','analisis_o3','operacion_ref','op:2','ruta','solicitud.grupo_subgrupo','valor_anterior','C2','valor_nuevo','C1'));
    IF v.expediente_ref <> 'exp:ct117' OR v.version_expediente <> 2
       OR v.cambios IS DISTINCT FROM esperado OR v.recortado
       OR v.auditoria_vec_ref <> 'aud:ct117'
       OR v.consumo_vec_huella_sha256 <> pg_catalog.repeat('1',64) THEN
        RAISE EXCEPTION 'CT117 salida positiva incorrecta: %', v.cambios;
    END IF;
    IF v.cambios::text ~ '600123456|958000000|Texto libre|sha256:' THEN
        RAISE EXCEPTION 'CT117 expuso texto libre en claro';
    END IF;
    SELECT t.n INTO STRICT v_n FROM vec_contratacion_temporal.test_consumos t;
    IF v_n <> 1 THEN RAISE EXCEPTION 'CT117 consumo positivo incorrecto'; END IF;

    -- 600 cambios: se entregan 500 y la respuesta indica el recorte.
    SELECT * INTO STRICT v FROM ct117_caso_b;
    SELECT * INTO STRICT v FROM vec_contratacion_temporal
       .consultar_cambios_expediente_rrhh_atestado_v1(
        v.alcance,v.consulta,v.capacidad_bytes,
        v.decision_bytes,'\x01'::bytea,v.contexto_actor_bytes,
        1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,
        pg_catalog.decode(pg_catalog.repeat('01',44),'hex'));
    IF NOT v.recortado OR pg_catalog.jsonb_array_length(v.cambios) <> 500
       OR v.cambios->0->>'ruta' <> 'analisis.c001_clave' OR v.cambios->499->>'ruta' <> 'analisis.c500_clave'
       OR pg_catalog.octet_length(v.cambios::text) > 196608 THEN
        RAISE EXCEPTION 'CT117 recorte incorrecto: % cambios, recortado %', pg_catalog.jsonb_array_length(v.cambios), v.recortado;
    END IF;

    -- Campos limitados, garantía inferior o rol de seguimiento: rechazo
    -- antes de consumir. Otra organización: rechazo y consumo revertido.
    FOR caso IN 1..4 LOOP
        SELECT pg_catalog.convert_from(decision_bytes,'UTF8')::jsonb,
               pg_catalog.convert_from(capacidad_bytes,'UTF8')::jsonb
          INTO STRICT decision_j, capacidad_j FROM ct117_caso;
        IF caso = 1 THEN decision_j := pg_catalog.jsonb_set(decision_j,'{campos_permitidos}','["estado_clave"]'::jsonb); END IF;
        IF caso = 2 THEN decision_j := pg_catalog.jsonb_set(decision_j,'{garantia_minima}','"sustancial"'::jsonb); END IF;
        IF caso = 3 THEN decision_j := pg_catalog.jsonb_set(decision_j,'{version_rol_ref}','"rol:rrhh_interno_certificado_seguimiento_ct_1:v1"'::jsonb); END IF;
        d := pg_catalog.convert_to(decision_j::text,'UTF8');
        capacidad_j := pg_catalog.jsonb_set(capacidad_j,'{huella_decision_sha256}',
            pg_catalog.to_jsonb(pg_catalog.encode(pg_catalog.sha256(d),'hex')));
        c := pg_catalog.convert_to(capacidad_j::text,'UTF8');
        fallo := false;
        BEGIN
            PERFORM * FROM vec_contratacion_temporal
              .consultar_cambios_expediente_rrhh_atestado_v1(
                CASE WHEN caso = 4 THEN ROW('org:otra','organizacion','org:otra')::vec_contratacion_temporal.alcance_consulta_rrhh_v1
                     ELSE (SELECT alcance FROM ct117_caso) END,
                (SELECT consulta FROM ct117_caso),c,d,
                '\x01'::bytea,(SELECT contexto_actor_bytes FROM ct117_caso),
                1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,
                pg_catalog.decode(pg_catalog.repeat('01',44),'hex'));
        EXCEPTION WHEN SQLSTATE '42501' THEN fallo := true;
        END;
        IF NOT fallo THEN RAISE EXCEPTION 'CT117 guarda incorrecta en caso %', caso; END IF;
        SELECT t.n INTO STRICT v_n FROM vec_contratacion_temporal.test_consumos t;
        IF v_n <> 2 THEN RAISE EXCEPTION 'CT117 consumió en el caso rechazado %', caso; END IF;
    END LOOP;

    -- Las funciones auxiliares y la historia no son accesibles directamente.
    fallo := false;
    BEGIN
        PERFORM 1 FROM vec_contratacion_temporal.expediente_version_integral;
    EXCEPTION WHEN insufficient_privilege THEN fallo := true;
    END;
    IF NOT fallo OR pg_catalog.has_function_privilege('vec_contratacion_temporal.valor_traza_cambio_v1(text,jsonb)','EXECUTE')
       OR pg_catalog.has_function_privilege('vec_contratacion_temporal.hojas_instantanea_expediente_v1(jsonb)','EXECUTE') THEN
        RAISE EXCEPTION 'CT117 abre la historia o sus auxiliares';
    END IF;
END
$probar$;
COMMIT;
SELECT 'CT117 fixture sintético: OK' AS resultado;
