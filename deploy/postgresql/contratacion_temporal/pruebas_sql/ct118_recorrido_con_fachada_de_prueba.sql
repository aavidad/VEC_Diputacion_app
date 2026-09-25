\set ON_ERROR_STOP on
-- Recorrido de CT118 sobre PostgreSQL 18 real con AD3-85 y CT118 instaladas
-- y al menos un expediente sintético. SOLO PRUEBA: dentro de una única
-- transacción se sustituye la fachada AD3-85 por un doble que devuelve un
-- consumo sintético nuevo, para ejercer la escritura de firma, auditoría y
-- outbox, la recuperación idempotente y los conflictos. Todo termina en
-- ROLLBACK: la fachada real, los roles y las filas quedan como estaban.
-- El consumo V3 real lo ejercen la composición y el núcleo AD3.
SELECT a.expediente_ref AS expediente, a.version AS version, v.agregado_json->>'organizacion_ref' AS organizacion
  FROM vec_contratacion_temporal.expediente_integral_actual a
  JOIN vec_contratacion_temporal.expediente_version_integral v USING (expediente_ref, version)
 ORDER BY a.expediente_ref LIMIT 1 \gset

BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path = pg_catalog;
SELECT set_config('prueba_ct118.organizacion', :'organizacion', true),
       set_config('prueba_ct118.expediente', :'expediente', true),
       set_config('prueba_ct118.version', :'version', true);
CREATE ROLE vec_ct118_recorrido_ejecutor LOGIN;
GRANT vec_contratacion_temporal_ejecutor TO vec_ct118_recorrido_ejecutor;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_firma_documento_ct_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE d jsonb := convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 -- Doble de prueba: eco del efecto de la decisión y consumo único.
 RETURN QUERY SELECT 'decision:'||md5(random()::text), d->>'recurso_ref', d->>'contexto_recurso_huella_sha256',
  encode(sha256(convert_to(random()::text,'UTF8')),'hex'), 'auditoria:'||md5(random()::text), clock_timestamp(), true;
END $f$;
RESET ROLE;

CREATE FUNCTION pg_temp.solicitud(p_org text, p_exp text, p_version numeric, p_paso int, p_sec int, p_resultado text,
    p_motivo text, p_original text, p_firmado text, p_clave text) RETURNS text LANGUAGE sql AS $s$
  SELECT jsonb_build_object('OrganizacionRef',p_org,'ExpedienteRef',p_exp,'VersionExpediente',p_version,
    'Documento','informe_definitivo','CatalogoRef','vec.contratacion_temporal.circuito_firma:1',
    'CatalogoHuella',repeat('c',64),'PasoRef','vec.contratacion_temporal.circuito_firma:1:informe_definitivo.p'||p_paso,
    'PasoOrden',p_paso,'Secuencia',p_sec,'Resultado',p_resultado,'MotivoDevolucion',p_motivo,
    'OriginalHuella',p_original,'FirmadoHuella',p_firmado,
    'CertificadoHuella',CASE WHEN p_resultado='firmado' THEN repeat('e',64) END,
    'FirmanteRef',CASE WHEN p_resultado='firmado' THEN 'ref:'||repeat('f',64) END,
    'PoliticaVerificacion',CASE WHEN p_resultado='firmado' THEN 'politica:vec:firma:verificacion-autonoma:v1' END,
    'RevocacionEstado',CASE WHEN p_resultado='firmado' THEN 'vigente' END,
    'SelloTiempoEstado',CASE WHEN p_resultado='firmado' THEN 'no_presente' END,
    'ClaveIdempotencia',p_clave)::text
$s$;
CREATE FUNCTION pg_temp.decision(p_solicitud text) RETURNS bytea LANGUAGE sql AS $d$
  SELECT convert_to(jsonb_build_object('accion','contratacion_temporal.documento.firmar','modulo_id','contratacion_temporal',
    'tipo_recurso','firma_documento_contratacion_temporal','finalidad','gestionar_contratacion_temporal',
    'recurso_ref','operacion-firma-ct:'||(p_solicitud::jsonb->>'ClaveIdempotencia'),
    'contexto_recurso_huella_sha256',encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||(p_solicitud::jsonb->>'OrganizacionRef')||
      '"},"atributos":{"material_sha256":"'||encode(sha256(convert_to(p_solicitud,'UTF8')),'hex')||'"}}','UTF8')),'hex'),
    'principal_id','principal:prueba:rrhh','perfil_activo_ref','perfil:prueba:rrhh')::text,'UTF8')
$d$;
CREATE FUNCTION pg_temp.registrar(p_solicitud text) RETURNS jsonb LANGUAGE sql AS $r$
  SELECT vec_contratacion_temporal.registrar_firma_documento_v1(p_solicitud,'\x',pg_temp.decision(p_solicitud),'\x','\x',1,1,'\x','\x','\x','\x')
$r$;
CREATE FUNCTION pg_temp.debe_fallar(p_sql text, p_codigo text, p_etiqueta text) RETURNS void LANGUAGE plpgsql AS $f$
BEGIN
    BEGIN
        EXECUTE p_sql;
    EXCEPTION WHEN others THEN
        IF SQLSTATE <> p_codigo THEN
            RAISE EXCEPTION 'FALLO %: código % (esperado %): %', p_etiqueta, SQLSTATE, p_codigo, SQLERRM;
        END IF;
        RAISE NOTICE 'OK %', p_etiqueta;
        RETURN;
    END;
    RAISE EXCEPTION 'FALLO %: no se rechazó', p_etiqueta;
END $f$;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA pg_temp TO PUBLIC;
SET SESSION AUTHORIZATION vec_ct118_recorrido_ejecutor;
SET LOCAL statement_timeout = '15s';

-- Paso 1 firmado sobre el borrador «a…», paso 2 devuelto y paso 1 de nuevo.
DO $recorrido$
DECLARE r jsonb; s text; org text := current_setting('prueba_ct118.organizacion'); exp text := current_setting('prueba_ct118.expediente');
    ver numeric := current_setting('prueba_ct118.version')::numeric;
BEGIN
    s := pg_temp.solicitud(org,exp,ver,1,1,'firmado',NULL,repeat('a',64),repeat('1',64),'clave-recorrido-000001');
    r := pg_temp.registrar(s);
    IF r->>'YaRegistrada' <> 'false' OR (r->>'Secuencia')::int <> 1 OR r->>'PerfilRef' <> 'perfil:prueba:rrhh'
       OR r->>'SolicitudHuella' <> encode(sha256(convert_to(s,'UTF8')),'hex') THEN
        RAISE EXCEPTION 'FALLO primera firma: %', r;
    END IF;
    -- La misma clave con el mismo material recupera el recibo, sin escribir.
    IF pg_temp.registrar(s)->>'YaRegistrada' <> 'true' OR pg_temp.registrar(s)->>'ReciboRef' <> r->>'ReciboRef' THEN
        RAISE EXCEPTION 'FALLO recuperación idempotente';
    END IF;
    r := pg_temp.registrar(pg_temp.solicitud(org,exp,ver,2,2,'devuelto','Falta la fecha de efectos',NULL,NULL,'clave-recorrido-000002'));
    IF (r->>'Secuencia')::int <> 2 OR r->>'Resultado' <> 'devuelto' THEN
        RAISE EXCEPTION 'FALLO devolución: %', r;
    END IF;
    RAISE NOTICE 'OK firma, recuperación y devolución';
END $recorrido$;

SELECT pg_temp.debe_fallar(format($q$SELECT pg_temp.registrar(pg_temp.solicitud(%L,%L,%s,1,3,'firmado',NULL,repeat('a',64),repeat('2',64),'clave-recorrido-000001'))$q$,
    :'organizacion',:'expediente',:'version'),'P1181','clave reutilizada con otro material');
SELECT pg_temp.debe_fallar(format($q$SELECT pg_temp.registrar(pg_temp.solicitud(%L,%L,%s,1,2,'firmado',NULL,repeat('a',64),repeat('2',64),'clave-recorrido-000003'))$q$,
    :'organizacion',:'expediente',:'version'),'P1183','secuencia ya ocupada');
SELECT pg_temp.debe_fallar(format($q$SELECT pg_temp.registrar(pg_temp.solicitud(%L,%L,%s,1,3,'firmado',NULL,repeat('a',64),repeat('2',64),'clave-recorrido-000004'))$q$,
    :'organizacion',:'expediente',:version+1),'P1182','versión del expediente distinta');
SELECT pg_temp.debe_fallar(format($q$SELECT pg_temp.registrar(pg_temp.solicitud(%L,%L,%s,2,3,'firmado',NULL,repeat('9',64),repeat('2',64),'clave-recorrido-000005'))$q$,
    :'organizacion',:'expediente',:'version'),'P1184','paso 2 sin continuar la firma del paso 1');
SELECT pg_temp.debe_fallar(format($q$SELECT pg_temp.registrar(pg_temp.solicitud('organizacion:ajena',%L,%s,1,3,'firmado',NULL,repeat('a',64),repeat('2',64),'clave-recorrido-000006'))$q$,
    :'expediente',:'version'),'42501','organización distinta de la del expediente');

-- Paso 2 continúa exactamente la firma del paso 1 («1…»).
DO $cadena$
DECLARE r jsonb; org text := current_setting('prueba_ct118.organizacion'); exp text := current_setting('prueba_ct118.expediente');
BEGIN
    r := pg_temp.registrar(pg_temp.solicitud(org,exp,current_setting('prueba_ct118.version')::numeric,2,3,'firmado',NULL,repeat('1',64),repeat('2',64),'clave-recorrido-000007'));
    IF (r->>'Secuencia')::int <> 3 THEN RAISE EXCEPTION 'FALLO cadena: %', r; END IF;
    IF jsonb_array_length(vec_contratacion_temporal.consultar_firmas_documento_v1(org,exp)) <> 3 THEN
        RAISE EXCEPTION 'FALLO consulta de la historia';
    END IF;
    RAISE NOTICE 'OK cadena entre pasos y consulta';
END $cadena$;
RESET SESSION AUTHORIZATION;

-- Una fila de firma, auditoría y outbox por registro, y nada más.
DO $conteo$
BEGIN
    IF (SELECT count(*) FROM vec_contratacion_temporal.firma_documento_v1) <> 3
       OR (SELECT count(*) FROM vec_contratacion_temporal.firma_documento_auditoria_v1) <> 3
       OR (SELECT count(*) FROM vec_contratacion_temporal.firma_documento_outbox_v1) <> 3
       OR (SELECT count(*) FROM vec_contratacion_temporal.firma_documento_outbox_v1 WHERE tipo='contratacion_temporal.documento.devuelto') <> 1 THEN
        RAISE EXCEPTION 'FALLO filas de firma, auditoría y outbox';
    END IF;
    RAISE NOTICE 'OK firma, auditoría y outbox 1:1';
END $conteo$;
ROLLBACK;
