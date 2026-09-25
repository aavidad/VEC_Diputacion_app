\set ON_ERROR_STOP on
-- Pruebas CT119 tras las de CT111 (misma base desechable): la expiración
-- confirmada «expiracion1» y una renuncia nueva continúan con la versión 2;
-- CT60 queda intacta y sigue rechazando la expiración. Dobles AD3: se prueba
-- la ligadura del material, no la criptografía. Datos sintéticos.
DO $$ BEGIN
    IF to_regprocedure('vec_contratacion_temporal.continuar_llamamiento_rrhh_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
       OR to_regclass('prueba_ct111.antes_reinicio') IS NULL THEN
        RAISE EXCEPTION 'CT119 o las pruebas CT111 no están aplicadas';
    END IF;
END $$;
CREATE SCHEMA prueba_ct119;
GRANT USAGE ON SCHEMA prueba_ct119 TO vec_ct111_runtime, vec_ct111_ajeno;
CREATE FUNCTION prueba_ct119.decision(p_material text, p_accion text DEFAULT 'contratacion_temporal.llamamiento.siguiente.continuar',
    p_denegar boolean DEFAULT false, p_actor text DEFAULT 'actor:rrhh:uno') RETURNS bytea
LANGUAGE sql STABLE SET search_path = pg_catalog AS $f$
    SELECT convert_to(jsonb_build_object(
        'accion',p_accion,'modulo_id','contratacion_temporal','tipo_recurso','continuacion_llamamiento_ct',
        'finalidad','gestionar_contratacion_temporal',
        'recurso_ref',(p_material::jsonb)->'Solicitud'->>'ExpedienteRef',
        'contexto_recurso_huella_sha256',encode(sha256(convert_to(
            '{"ambitos":{"organizacion_ref":"'||((p_material::jsonb)->'Solicitud'->>'OrganizacionRef')||
            '"},"atributos":{"material_sha256":"'||encode(sha256(convert_to(p_material,'UTF8')),'hex')||'"}}','UTF8')),'hex'),
        'principal_id',p_actor,'perfil_activo_ref','perfil:rrhh:gestion',
        'denegar',CASE WHEN p_denegar THEN 'si' ELSE 'no' END)::text,'UTF8')
$f$;
CREATE FUNCTION prueba_ct119.v2(p_material text, p_accion text DEFAULT 'contratacion_temporal.llamamiento.siguiente.continuar',
    p_denegar boolean DEFAULT false, p_actor text DEFAULT 'actor:rrhh:uno') RETURNS jsonb
LANGUAGE sql VOLATILE SET search_path = pg_catalog AS $f$
    SELECT vec_contratacion_temporal.continuar_llamamiento_rrhh_v2(p_material,
        '\x01'::bytea,prueba_ct119.decision(p_material,p_accion,p_denegar,p_actor),'\x01'::bytea,'\x01'::bytea,1,1,
        '\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea)
$f$;
CREATE FUNCTION prueba_ct119.v1(p_material text) RETURNS jsonb
LANGUAGE sql VOLATILE SET search_path = pg_catalog AS $f$
    SELECT vec_contratacion_temporal.continuar_llamamiento_rrhh_v1(p_material,
        '\x01'::bytea,prueba_ct119.decision(p_material),'\x01'::bytea,'\x01'::bytea,1,1,
        '\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea)
$f$;
-- Materiales en el orden de json.Marshal de ports.MaterialContinuacionLlamamiento.
CREATE FUNCTION prueba_ct119.solicitud(p_clave uuid, p_n int, p_resolucion text, p_intencion text) RETURNS text
LANGUAGE sql IMMUTABLE SET search_path = pg_catalog AS $f$
    SELECT format('{"ClaveIdempotencia":"%s","OrganizacionRef":"organizacion:ct111","ExpedienteRef":"expediente:ct111:%s","ResolucionRef":"%s","IntencionRef":"%s"}',
        p_clave,p_n,p_resolucion,p_intencion)
$f$;
CREATE FUNCTION prueba_ct119.consulta(p_solicitud text) RETURNS text
LANGUAGE sql IMMUTABLE SET search_path = pg_catalog AS $f$
    SELECT format('{"Etapa":"consulta","Solicitud":%s}',p_solicitud)
$f$;
CREATE FUNCTION prueba_ct119.confirmacion(p_solicitud text, p_intencion text, p_llamamiento text,
    p_confirmada timestamptz) RETURNS text
LANGUAGE sql IMMUTABLE SET search_path = pg_catalog AS $f$
    SELECT format('{"Etapa":"confirmacion","Solicitud":%s,"ReciboBolsa":{"IntencionRef":"%s","TerminalOperacionRef":"operacion:bolsa:terminal:%s","OperacionRef":"operacion:bolsa:siguiente:%s","LlamamientoRef":"%s","PropuestaRef":"propuesta:bolsa:%s","ReciboRef":"recibo:bolsa:%s","AuditoriaRef":"auditoria:bolsa:%s","EventoRef":"evento:bolsa:%s","RegistroSHA256":"%s","ConfirmadaEn":"%s"}}',
        p_solicitud,p_intencion,p_llamamiento,p_llamamiento,p_llamamiento,p_llamamiento,p_llamamiento,p_llamamiento,p_llamamiento,
        repeat('ef',32),to_char(p_confirmada AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))
$f$;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA prueba_ct119 TO vec_ct111_runtime, vec_ct111_ajeno;

-- Renuncia sintética nueva (llamamiento 8, circuito histórico, sin plazo
-- abierto) para comprobar que la versión 2 conserva la continuación de CT60.
INSERT INTO vec_contratacion_temporal.respuesta_recibida_rrhh
SELECT 'justificante:ct111:8','organizacion:ct111','expediente:ct111:8','llamamiento:ct111:8',
       'comunicacion:ct111:8','00000000-0000-4000-8000-000000000008'::uuid,gen_random_uuid(),
       'actor:rrhh:uno','perfil:rrhh:gestion',2,'renuncia','correo:ct111:8',repeat('cd',32),
       rec,mat,mat::jsonb,encode(sha256(convert_to(mat,'UTF8')),'hex'),'recibo:respuesta:ct111:8',
       jsonb_build_object('JustificanteRef','justificante:ct111:8','ReciboRef','recibo:respuesta:ct111:8',
           'Solicitud',mat::jsonb,'Estado','registrada_por_rrhh'),'registrada_por_rrhh',rec+interval '1 minute'
  FROM (SELECT clock_timestamp()-interval '12 hours' AS rec, '{"n":8}' AS mat) x;
SET SESSION AUTHORIZATION vec_ct111_runtime;
CREATE TEMP TABLE r119 (caso text PRIMARY KEY, valor jsonb NOT NULL);
BEGIN ISOLATION LEVEL SERIALIZABLE;
INSERT INTO r119 SELECT 'renuncia8', prueba_ct111.resolver(prueba_ct111.material_resolucion(8,
    '22222222-2222-4222-8222-000000000080','renuncia',prueba_ct111.historica()));
COMMIT;
RESET SESSION AUTHORIZATION;
INSERT INTO r119 SELECT 'expiracion1', valor FROM prueba_ct111.antes_reinicio WHERE caso='expiracion1';
INSERT INTO r119 SELECT 'aceptacion3', valor FROM prueba_ct111.antes_reinicio WHERE caso='aceptacion3';
GRANT SELECT ON r119 TO vec_ct111_runtime;
CREATE TABLE prueba_ct119.solicitudes AS SELECT caso,
    prueba_ct119.solicitud(clave::uuid,n,valor->>'ResolucionRef',valor->'IntencionSiguiente'->>'IntencionRef') AS solicitud,
    valor->'IntencionSiguiente'->>'IntencionRef' AS intencion
  FROM r119 JOIN (VALUES ('expiracion1','33333333-3333-4333-8333-000000000001',1),
                         ('renuncia8','33333333-3333-4333-8333-000000000008',8)) v(c,clave,n) ON v.c=r119.caso;
GRANT SELECT ON prueba_ct119.solicitudes TO vec_ct111_runtime, vec_ct111_ajeno;
SELECT prueba_ct111.exigir((SELECT count(*) FROM prueba_ct119.solicitudes)=2
    AND (SELECT valor->'IntencionSiguiente'->>'Estado' FROM r119 WHERE caso='renuncia8')='pendiente',
    'antecedentes: expiración y renuncia con intención pendiente');

-- 1. Consulta del antecedente.
SET SESSION AUTHORIZATION vec_ct111_runtime;
BEGIN ISOLATION LEVEL SERIALIZABLE;
INSERT INTO r119 SELECT 'consulta-expiracion', prueba_ct119.v2(prueba_ct119.consulta(solicitud))
  FROM prueba_ct119.solicitudes WHERE caso='expiracion1';
INSERT INTO r119 SELECT 'consulta-renuncia', prueba_ct119.v2(prueba_ct119.consulta(solicitud))
  FROM prueba_ct119.solicitudes WHERE caso='renuncia8';
COMMIT;
SELECT prueba_ct111.exigir((SELECT valor->'Resolucion'->>'EstadoPlazo'='expirado'
    AND valor->'Resolucion'->'Solicitud'->>'Respuesta'='expiracion_gobernada'
    AND valor->'ComandoSiguiente'->>'intencion_ref'=(SELECT intencion FROM prueba_ct119.solicitudes WHERE caso='expiracion1')
    AND valor->'ComandoSiguiente'->'justificante_ref'='null'::jsonb
    AND valor->'Seleccion'->>'llamamiento_ref'='llamamiento:ct111:1'
    AND valor->'Seleccion'->>'recibo_ref'='recibo:seleccion:ct111:1'
    FROM r119 WHERE caso='consulta-expiracion'),'consulta de expiración con la selección original');
SELECT prueba_ct111.exigir((SELECT valor->'Resolucion'->'Solicitud'->>'Respuesta'='renuncia' AND NOT valor ? 'Seleccion'
    FROM r119 WHERE caso='consulta-renuncia'),'consulta de renuncia sin selección añadida');

-- 2. Rechazos antes de confirmar.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT prueba_ct111.esperar($s$SELECT prueba_ct119.v1(prueba_ct119.consulta(solicitud))
    FROM prueba_ct119.solicitudes WHERE caso='expiracion1'$s$,'P0602','CT60 sigue sin admitir la expiración');
SELECT prueba_ct111.esperar($s$SELECT prueba_ct119.v2(prueba_ct119.consulta(prueba_ct119.solicitud(
    '33333333-3333-4333-8333-000000000003'::uuid,3,(SELECT valor->>'ResolucionRef' FROM r119 WHERE caso='aceptacion3'),
    'intencion:inexistente')))$s$,'P0602','aceptación no es antecedente de continuación');
SELECT prueba_ct111.esperar($s$SELECT prueba_ct119.v2(prueba_ct119.consulta(replace(solicitud,intencion,'intencion:ajena')))
    FROM prueba_ct119.solicitudes WHERE caso='expiracion1'$s$,'P0602','intención distinta de la durable');
SELECT prueba_ct111.esperar($s$SELECT prueba_ct119.v2(prueba_ct119.consulta(solicitud),
    'contratacion_temporal.llamamiento.respuesta.validacion_manual.registrar')
    FROM prueba_ct119.solicitudes WHERE caso='expiracion1'$s$,'P0603','permiso de otra acción');
SELECT prueba_ct111.esperar($s$SELECT prueba_ct119.v2(prueba_ct119.consulta(solicitud),
    'contratacion_temporal.llamamiento.siguiente.continuar',true)
    FROM prueba_ct119.solicitudes WHERE caso='expiracion1'$s$,'P0583','autorización denegada por el consumidor (doble)');
SELECT prueba_ct111.esperar($s$SELECT prueba_ct119.v2(replace(prueba_ct119.consulta(solicitud),'"Etapa":"consulta"','"Etapa":"consulta","Extra":1'))
    FROM prueba_ct119.solicitudes WHERE caso='expiracion1'$s$,'P0600','campo añadido al material');
SELECT prueba_ct111.esperar($s$SELECT prueba_ct119.v2(prueba_ct119.confirmacion(solicitud,intencion,'llamamiento:ct111:1',
    clock_timestamp()-interval '1 minute')) FROM prueba_ct119.solicitudes WHERE caso='expiracion1'$s$,
    'P0602','recibo Bolsa del mismo llamamiento');
SELECT prueba_ct111.esperar($s$SELECT prueba_ct119.v2(prueba_ct119.confirmacion(solicitud,intencion,'llamamiento:bolsa:siguiente:1',
    clock_timestamp()+interval '1 hour')) FROM prueba_ct119.solicitudes WHERE caso='expiracion1'$s$,
    'P0600','recibo Bolsa posterior al registro');
COMMIT;
BEGIN ISOLATION LEVEL READ COMMITTED;
SELECT prueba_ct111.esperar($s$SELECT prueba_ct119.v2(prueba_ct119.consulta(solicitud))
    FROM prueba_ct119.solicitudes WHERE caso='expiracion1'$s$,'P0603','aislamiento no serializable');
COMMIT;

-- 3. Confirmaciones, replay y claves divergentes. El material exacto se
-- conserva: el replay debe repetir los mismos bytes autorizados.
CREATE TEMP TABLE m119 (caso text PRIMARY KEY, material text NOT NULL);
INSERT INTO m119 SELECT 'continuacion1', prueba_ct119.confirmacion(solicitud,intencion,
    'llamamiento:bolsa:siguiente:1',clock_timestamp()) FROM prueba_ct119.solicitudes WHERE caso='expiracion1';
INSERT INTO m119 SELECT 'continuacion8', prueba_ct119.confirmacion(solicitud,intencion,
    'llamamiento:bolsa:siguiente:8',clock_timestamp()) FROM prueba_ct119.solicitudes WHERE caso='renuncia8';
BEGIN ISOLATION LEVEL SERIALIZABLE;
INSERT INTO r119 SELECT caso, prueba_ct119.v2(material) FROM m119 ORDER BY caso;
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT prueba_ct111.esperar($s$SELECT prueba_ct119.v2(prueba_ct119.confirmacion(replace(solicitud,
    '33333333-3333-4333-8333-000000000001','33333333-3333-4333-8333-000000000011'),intencion,
    'llamamiento:bolsa:siguiente:1',clock_timestamp()))
    FROM prueba_ct119.solicitudes WHERE caso='expiracion1'$s$,'P0601','otra clave para la misma intención');
SELECT prueba_ct111.esperar($s$SELECT prueba_ct119.v2(prueba_ct119.confirmacion(solicitud,intencion,
    'llamamiento:bolsa:otro:1',clock_timestamp()))
    FROM prueba_ct119.solicitudes WHERE caso='expiracion1'$s$,'P0601','misma clave con otro recibo Bolsa');
SELECT prueba_ct111.esperar($s$SELECT prueba_ct119.v2((SELECT material FROM m119 WHERE caso='continuacion1'),
    'contratacion_temporal.llamamiento.siguiente.continuar',false,'actor:rrhh:otro')$s$,
    'P0601','replay con otra identidad');
COMMIT;
RESET SESSION AUTHORIZATION;
SELECT prueba_ct111.exigir((SELECT valor->>'Estado'='confirmado' AND valor->>'LlamamientoAnteriorRef'='llamamiento:ct111:1'
    AND valor->'ReciboBolsa'->>'LlamamientoRef'='llamamiento:bolsa:siguiente:1'
    FROM r119 WHERE caso='continuacion1'),'continuación confirmada tras la expiración');
SELECT prueba_ct111.exigir((SELECT valor->>'Estado'='confirmado' AND valor->>'LlamamientoAnteriorRef'='llamamiento:ct111:8'
    FROM r119 WHERE caso='continuacion8'),'continuación tras renuncia con la versión 2');
SET SESSION AUTHORIZATION vec_ct111_runtime;
BEGIN ISOLATION LEVEL SERIALIZABLE;
INSERT INTO r119 SELECT 'continuacion1-replay', prueba_ct119.v2(material) FROM m119 WHERE caso='continuacion1';
COMMIT;
RESET SESSION AUTHORIZATION;
SELECT prueba_ct111.exigir((SELECT r.valor->>'Estado'='replay_confirmado' AND r.valor-'Estado'=o.valor-'Estado'
    FROM r119 r, r119 o WHERE r.caso='continuacion1-replay' AND o.caso='continuacion1'),
    'replay devuelve el recibo original sin efecto nuevo');

-- 4. ACL, historia inmutable y conteos.
SET SESSION AUTHORIZATION vec_ct111_ajeno;
SELECT prueba_ct111.esperar($s$SELECT prueba_ct119.v2(prueba_ct119.consulta(solicitud))
    FROM prueba_ct119.solicitudes WHERE caso='expiracion1'$s$,'42501','identidad sin rol ejecutor');
RESET SESSION AUTHORIZATION;
SELECT prueba_ct111.exigir(NOT has_function_privilege('vec_ct111_ajeno',
    'vec_contratacion_temporal.continuar_llamamiento_rrhh_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    AND NOT has_function_privilege('vec_contratacion_temporal_migrador',
    'vec_contratacion_temporal.continuar_llamamiento_rrhh_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    AND has_function_privilege('vec_contratacion_temporal_ejecutor',
    'vec_contratacion_temporal.continuar_llamamiento_rrhh_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE'),
    'EXECUTE solo para el ejecutor');
SELECT prueba_ct111.esperar($s$UPDATE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh
    SET continuacion_actor_ref='actor:mutado' WHERE continuacion_clave IS NOT NULL$s$,'55000','confirmación inmutable');
SELECT prueba_ct111.exigir((SELECT count(*) FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh
    WHERE continuacion_clave IS NOT NULL)=2
    AND (SELECT count(*) FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh)=6,
    'dos continuaciones y seis resoluciones, sin duplicados');
CREATE TABLE prueba_ct119.antes_reinicio AS SELECT r.*, m.material FROM r119 r LEFT JOIN m119 m USING (caso);
GRANT SELECT ON prueba_ct119.antes_reinicio TO vec_ct111_runtime;
