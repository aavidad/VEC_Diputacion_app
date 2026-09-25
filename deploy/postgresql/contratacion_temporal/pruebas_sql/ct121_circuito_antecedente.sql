\set ON_ERROR_STOP on
-- Antecedente real de CT121, antes de instalarla: sobre el fixture
-- ct121_circuito_fixture.sql (expediente B rebobinado a su primer aviso y
-- dobles AD3), con la cuenta de ejecución y en SERIALIZABLE:
--   CT111  contacto efectivo con el plazo ya vencido y expiración confirmada
--          por RRHH (sin respuesta ni justificante);
--   CT119  continuación (consulta y confirmación) con el recibo de Bolsa.
-- Sin CT121 el aviso del sucesor rechaza esa continuación. Deja los recibos
-- y las funciones auxiliares en prueba_ct121c para ct121_circuito_sucesor.sql.
\set exp_b 'expediente:ct:fe4934a1c7a9f9ad91aaccc6026ff7d39a494031d14d8a98dcd0d6a140619ba7'
\set llam_1 'llamamiento:nccfkjnioljdeikkpipkcgcpilbogjnociankdfbapmnaekanagbiioaahphbmgj'
\set llam_2 'llamamiento:eipmgkkfjncbalgebihinpeeajhllfmkoikjhiodlbcdlhaohmgpojefdfilcmoe'
\set org 'organizacion:desarrollo:dipgra'

CREATE SCHEMA prueba_ct121c;
GRANT USAGE ON SCHEMA prueba_ct121c TO vec_ct121c_runtime;
CREATE TABLE prueba_ct121c.caso (
    caso text PRIMARY KEY, funcion text NOT NULL, accion text NOT NULL, tipo text NOT NULL,
    material text NOT NULL, resultado jsonb NOT NULL);
GRANT SELECT, INSERT ON prueba_ct121c.caso TO vec_ct121c_runtime;
CREATE TABLE prueba_ct121c.valor (clave text PRIMARY KEY, valor text NOT NULL);
GRANT SELECT ON prueba_ct121c.valor TO vec_ct121c_runtime;
INSERT INTO prueba_ct121c.valor VALUES ('exp_b', :'exp_b'), ('llam_1', :'llam_1'), ('llam_2', :'llam_2'), ('org', :'org');

CREATE FUNCTION prueba_ct121c.v(p text) RETURNS text LANGUAGE sql STABLE SET search_path = pg_catalog AS
$f$ SELECT valor FROM prueba_ct121c.valor WHERE clave=p $f$;
CREATE FUNCTION prueba_ct121c.r(p_caso text) RETURNS jsonb LANGUAGE sql STABLE SET search_path = pg_catalog AS
$f$ SELECT resultado FROM prueba_ct121c.caso WHERE caso=p_caso $f$;
CREATE FUNCTION prueba_ct121c.instante(p timestamptz) RETURNS text LANGUAGE sql IMMUTABLE SET search_path = pg_catalog AS
$f$ SELECT to_char(p AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') $f$;
-- Decisión del doble AD3 ligada al material exacto, como la emite la frontera.
CREATE FUNCTION prueba_ct121c.decision(p_material text, p_accion text, p_tipo text) RETURNS bytea
LANGUAGE sql STABLE SET search_path = pg_catalog AS $f$
    WITH m AS (SELECT p_material::jsonb AS j)
    SELECT convert_to(jsonb_build_object(
        'accion',p_accion,'modulo_id','contratacion_temporal','tipo_recurso',p_tipo,
        'finalidad','gestionar_contratacion_temporal',
        'recurso_ref',coalesce(j#>>'{Solicitud,ExpedienteRef}',j#>>'{solicitud,ExpedienteRef}',j->>'ExpedienteRef'),
        'contexto_recurso_huella_sha256',encode(sha256(convert_to(
            '{"ambitos":{"organizacion_ref":"'||coalesce(j#>>'{Solicitud,OrganizacionRef}',j#>>'{solicitud,OrganizacionRef}',j->>'OrganizacionRef')||
            '"},"atributos":{"material_sha256":"'||encode(sha256(convert_to(p_material,'UTF8')),'hex')||'"}}','UTF8')),'hex'),
        'principal_id','per_ct121_rrhh','perfil_activo_ref','prf_ct121_rrhh')::text,'UTF8')
      FROM m
$f$;
CREATE FUNCTION prueba_ct121c.llamar(p_funcion text, p_material text, p_accion text, p_tipo text) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SET search_path = pg_catalog AS $f$
DECLARE v jsonb;
BEGIN
    EXECUTE format('SELECT vec_contratacion_temporal.%I($1,$2,$3,$2,$2,1,1,$2,$2,$2,$2)', p_funcion)
       INTO v USING p_material, '\x01'::bytea, prueba_ct121c.decision(p_material,p_accion,p_tipo);
    RETURN v;
END
$f$;
CREATE FUNCTION prueba_ct121c.paso(p_caso text, p_funcion text, p_material text, p_accion text, p_tipo text) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SET search_path = pg_catalog AS $f$
DECLARE v jsonb := prueba_ct121c.llamar(p_funcion,p_material,p_accion,p_tipo);
BEGIN
    INSERT INTO prueba_ct121c.caso VALUES (p_caso,p_funcion,p_accion,p_tipo,p_material,v);
    RETURN v;
END
$f$;
CREATE FUNCTION prueba_ct121c.repetir(p_caso text) RETURNS jsonb
LANGUAGE sql VOLATILE SET search_path = pg_catalog AS $f$
    SELECT prueba_ct121c.llamar(funcion,material,accion,tipo) FROM prueba_ct121c.caso WHERE caso=p_caso
$f$;
CREATE FUNCTION prueba_ct121c.exigir(p_condicion boolean, p_caso text) RETURNS text
LANGUAGE plpgsql SET search_path = pg_catalog AS $f$
BEGIN
    IF p_condicion IS NOT TRUE THEN RAISE EXCEPTION 'FALLO %', p_caso; END IF;
    RETURN 'ok';
END
$f$;
CREATE FUNCTION prueba_ct121c.referencia(p_entrada text) RETURNS text LANGUAGE sql IMMUTABLE SET search_path = pg_catalog AS
$f$ SELECT format('{"Referencia":"vec.bolsa.reglas:3:%s","Version":3,"HuellaSHA256":"%s"}',p_entrada,repeat('ab',32)) $f$;
-- Publicaciones de la propuesta, leídas aquí (la cuenta de ejecución no lee la tabla).
INSERT INTO prueba_ct121c.valor
SELECT 'publicacion:'||componente, format('{"Referencia":"%s","Version":%s,"HuellaSHA256":"%s"}',referencia,version,huella_sha256)
  FROM vec_contratacion_temporal.publicacion_propuesta_formalizacion_desarrollo;
CREATE FUNCTION prueba_ct121c.publicacion(p_componente text) RETURNS text LANGUAGE sql STABLE SET search_path = pg_catalog AS
$f$ SELECT prueba_ct121c.v('publicacion:'||p_componente) $f$;
-- Material del aviso del sucesor (CT62), ligado al recibo de la continuación.
CREATE FUNCTION prueba_ct121c.material_aviso() RETURNS text LANGUAGE sql STABLE SET search_path = pg_catalog AS $f$
    SELECT format('{"solicitud":{"ClaveIdempotencia":"c1210000-0000-4000-8000-000000000004","OrganizacionRef":"%s","ExpedienteRef":"%s","LlamamientoRef":"%s","VersionEsperada":1,"PruebaEntregaRef":"%s","TipoAntecedente":"continuacion_confirmada"},"canal":{"Referencia":"canal:ct:desarrollo:registro-local:v1","Version":1,"HuellaSHA256":"c2346af19aa8489c3b878b2c780fcf0251129e0a397ed6e3c32c4438c44a88dc"},"politica":{"Referencia":"politica:ct:desarrollo:registro-local:v2","Version":2,"HuellaSHA256":"24d08a85ca5e62b8b3832df49d208ccea098f2b202909639d900b9d42da0bacb"}}',
        prueba_ct121c.v('org'),prueba_ct121c.v('exp_b'),prueba_ct121c.v('llam_2'),prueba_ct121c.r('continuacion')->>'ReciboRef')
$f$;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA prueba_ct121c TO vec_ct121c_runtime;

SELECT comunicacion_ref AS aviso_1 FROM vec_contratacion_temporal.comunicacion_llamamiento_local
 WHERE expediente_ref=:'exp_b' AND llamamiento_ref=:'llam_1' \gset
INSERT INTO prueba_ct121c.valor VALUES ('aviso_1', :'aviso_1');

SET SESSION AUTHORIZATION vec_ct121c_runtime;
SET timezone = 'UTC';

-- 1. CT111: contacto efectivo hace dos días con el plazo vencido hace uno.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT prueba_ct121c.paso('contacto','registrar_evento_plazo_llamamiento_v1',
    format('{"Solicitud":{"ClaveIdempotencia":"c1210000-0000-4000-8000-000000000001","OrganizacionRef":"%s","ExpedienteRef":"%s","LlamamientoRef":"%s","ComunicacionRef":"%s","VersionComunicacionEsperada":2,"Tipo":"contacto_efectivo","InstanteEn":"%s","PruebaRef":"prueba:contacto:llamada"},"Plazo":{"RespuestaHasta":"%s","UltimoDia":"%s","Politica":%s,"TratamientoFueraDePlazo":"exige_causa_justificada","ConfirmacionExpiracion":"rrhh","CriterioRespuesta":%s,"CriterioExpiracion":%s,"ReglaEjemplo":true}}',
        prueba_ct121c.v('org'),prueba_ct121c.v('exp_b'),prueba_ct121c.v('llam_1'),prueba_ct121c.v('aviso_1'),
        prueba_ct121c.instante(date_trunc('second',clock_timestamp())-interval '2 days'),
        prueba_ct121c.instante(date_trunc('second',clock_timestamp())-interval '1 day'),
        to_char(((clock_timestamp()-interval '1 day') AT TIME ZONE 'Europe/Madrid')::date,'YYYY-MM-DD'),
        prueba_ct121c.referencia('b05.plazo_respuesta'),prueba_ct121c.referencia('b07.fuera_de_plazo'),
        prueba_ct121c.referencia('b08.sin_respuesta_baja')),
    'contratacion_temporal.llamamiento.respuesta.validacion_manual.registrar','resolucion_manual_respuesta_ct') IS NOT NULL;
COMMIT;
SELECT prueba_ct121c.exigir(prueba_ct121c.r('contacto')->>'Estado'='registrado','contacto efectivo registrado');

-- 2. CT111: expiración confirmada por RRHH, sin respuesta ni justificante.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT prueba_ct121c.paso('expiracion','registrar_resolucion_manual_respuesta_rrhh_v1',
    format('{"Solicitud":{"ClaveIdempotencia":"c1210000-0000-4000-8000-000000000002","OrganizacionRef":"%s","ExpedienteRef":"%s","LlamamientoRef":"%s","ComunicacionRef":"%s","VersionEsperada":2,"Respuesta":"expiracion_gobernada","PruebaRespuestaRef":"","RevisionRespuestaRRHH":true,"RevisionPlazoRRHH":true,"CriterioValidacionRef":"vec.bolsa.reglas:3:b08.sin_respuesta_baja"},"Politica":%s}',
        prueba_ct121c.v('org'),prueba_ct121c.v('exp_b'),prueba_ct121c.v('llam_1'),prueba_ct121c.v('aviso_1'),
        prueba_ct121c.referencia('b08.sin_respuesta_baja')),
    'contratacion_temporal.llamamiento.respuesta.validacion_manual.registrar','resolucion_manual_respuesta_ct') IS NOT NULL;
COMMIT;
SELECT prueba_ct121c.exigir(prueba_ct121c.r('expiracion')->>'Estado'='confirmado'
    AND prueba_ct121c.r('expiracion')->'IntencionSiguiente'->>'Estado'='pendiente','expiración confirmada con intención pendiente');

-- 3. CT119: continuación tras la expiración (consulta y confirmación).
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT prueba_ct121c.paso('continuacion-consulta','continuar_llamamiento_rrhh_v2',
    format('{"Etapa":"consulta","Solicitud":{"ClaveIdempotencia":"c1210000-0000-4000-8000-000000000003","OrganizacionRef":"%s","ExpedienteRef":"%s","ResolucionRef":"%s","IntencionRef":"%s"}}',
        prueba_ct121c.v('org'),prueba_ct121c.v('exp_b'),prueba_ct121c.r('expiracion')->>'ResolucionRef',
        prueba_ct121c.r('expiracion')->'IntencionSiguiente'->>'IntencionRef'),
    'contratacion_temporal.llamamiento.siguiente.continuar','continuacion_llamamiento_ct') IS NOT NULL;
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT prueba_ct121c.paso('continuacion','continuar_llamamiento_rrhh_v2',
    format('{"Etapa":"confirmacion","Solicitud":{"ClaveIdempotencia":"c1210000-0000-4000-8000-000000000003","OrganizacionRef":"%s","ExpedienteRef":"%s","ResolucionRef":"%s","IntencionRef":"%s"},"ReciboBolsa":{"IntencionRef":"%s","TerminalOperacionRef":"operacion-expiracion-rrhh:ct121","OperacionRef":"operacion-siguiente-rrhh:ct121","LlamamientoRef":"%s","PropuestaRef":"propuesta:bolsa:ct121","ReciboRef":"recibo:bolsa:ct121","AuditoriaRef":"auditoria:bolsa:ct121","EventoRef":"evento:bolsa:ct121","RegistroSHA256":"%s","ConfirmadaEn":"%s"}}',
        prueba_ct121c.v('org'),prueba_ct121c.v('exp_b'),prueba_ct121c.r('expiracion')->>'ResolucionRef',
        prueba_ct121c.r('expiracion')->'IntencionSiguiente'->>'IntencionRef',
        prueba_ct121c.r('expiracion')->'IntencionSiguiente'->>'IntencionRef',prueba_ct121c.v('llam_2'),
        repeat('ef',32),prueba_ct121c.instante(clock_timestamp())),
    'contratacion_temporal.llamamiento.siguiente.continuar','continuacion_llamamiento_ct') IS NOT NULL;
COMMIT;
SELECT prueba_ct121c.exigir(prueba_ct121c.r('continuacion')->>'Estado'='confirmado'
    AND prueba_ct121c.r('continuacion')->>'LlamamientoAnteriorRef'=prueba_ct121c.v('llam_1'),'continuación confirmada tras la expiración');

-- Sin CT121, el aviso del sucesor no admite la continuación de una expiración.
CREATE TEMP TABLE sin_ct121 (estado text);
BEGIN ISOLATION LEVEL SERIALIZABLE;
DO $sin$
BEGIN
    PERFORM prueba_ct121c.llamar('registrar_comunicacion_llamamiento_local_v1',prueba_ct121c.material_aviso(),
        'contratacion_temporal.llamamiento.comunicacion.registrar','comunicacion_llamamiento_contratacion_temporal');
    INSERT INTO sin_ct121 VALUES ('admitido');
EXCEPTION WHEN others THEN
    INSERT INTO sin_ct121 VALUES (SQLSTATE);
END
$sin$;
COMMIT;
SELECT prueba_ct121c.exigir(estado='42501','sin CT121 el aviso del sucesor tras la expiración se rechaza ('||estado||')') FROM sin_ct121;
RESET SESSION AUTHORIZATION;
SELECT prueba_ct121c.exigir(NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.comunicacion_llamamiento_local
    WHERE llamamiento_ref=prueba_ct121c.v('llam_2')),'el rechazo no deja aviso');
SELECT 'CT121 antecedente OK' AS resultado;
