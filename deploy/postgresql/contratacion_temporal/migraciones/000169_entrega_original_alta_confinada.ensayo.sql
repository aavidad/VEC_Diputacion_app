\set ON_ERROR_STOP on
-- Ejecutar sólo en clon sintético como DBA, tras instalar CT169 una vez.
-- Acredita ACL, guardas y denegación real; no sustituye el positivo COSE.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
DO $acl$
DECLARE f oid:='vec_contratacion_temporal.preparar_entrega_peticion_centro_con_original_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=f
     AND p.proowner='vec_contratacion_temporal_propietario'::regrole
     AND p.prosecdef AND p.provolatile='v'
     AND p.proconfig=ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s','idle_in_transaction_session_timeout=20s']::text[])
    OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p, LATERAL pg_catalog.aclexplode(p.proacl) a WHERE p.oid=f AND a.grantee=0)
    OR NOT pg_catalog.has_function_privilege('vec_contratacion_temporal_ejecutor',f,'EXECUTE')
    OR pg_catalog.has_function_privilege('vec_contratacion_temporal_migrador',f,'EXECUTE')
    OR pg_catalog.has_function_privilege('vec_contratacion_temporal_gobernador',f,'EXECUTE') THEN
   RAISE EXCEPTION 'ACL/metadata CT169 incompatible';
 END IF;
END
$acl$;
DO $fechas_civiles$
DECLARE p jsonb:=jsonb_build_object('regla_ref','regla:ct169:fin',
 'catalogo_version',1,'catalogo_huella_sha256',repeat('a',64),'fecha_fin','obligatoria');
BEGIN
 IF vec_contratacion_temporal.periodo_previsto_estructural_v1(jsonb_build_object(
    'inicio','2026-10-01','fin','2026-12-31','politica_fin',p)) IS NOT TRUE
    OR vec_contratacion_temporal.periodo_previsto_estructural_v1(jsonb_build_object(
    'inicio','2026-10-01','causa_fin','retorno_titular',
    'politica_fin',p||jsonb_build_object('fecha_fin','no_aplica','causa_fin','retorno_titular'))) IS NOT TRUE THEN
    RAISE EXCEPTION 'CT169 rechaza el canon civil original con política fin';
 END IF;
 IF vec_contratacion_temporal.periodo_previsto_estructural_v1(jsonb_build_object(
    'inicio','2026-10-01','fin','2026-12-31','politica_fin',p-'regla_ref')) IS NOT FALSE THEN
    RAISE EXCEPTION 'CT169 acepta política fin estructural incompleta';
 END IF;
END
$fechas_civiles$;
CREATE ROLE vec_ct169_ensayo_login LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_contratacion_temporal_ejecutor TO vec_ct169_ensayo_login;
CREATE TEMP TABLE ct169_ensayo_material(material text,accesos bigint,confirmaciones bigint,eventos bigint);
INSERT INTO ct169_ensayo_material
 SELECT jsonb_build_object('actor_ref',s.actor_ref,'perfil_ref',s.perfil_ref,
   'modo','preparar','peticion_ref',s.peticion_ref,'version_esperada',2,
   'clave_alta_candidata','11111111-1111-4111-8111-111111111111',
   'ambito_alta_hmac',s.ambito_alta_hmac,'centro_ref',r.peticion#>>'{solicitud,centro_ref}',
   'categoria_ref',r.peticion#>>'{solicitud,categoria_ref}')::text,
   (SELECT count(*) FROM vec_contratacion_temporal.entrega_peticion_centro_acceso),
   (SELECT count(*) FROM vec_contratacion_temporal.entrega_peticion_centro_confirmacion),
   (SELECT count(*) FROM vec_contratacion_temporal.entrega_peticion_centro_outbox)
 FROM vec_contratacion_temporal.entrega_peticion_centro_reserva s
 JOIN vec_contratacion_temporal.peticion_centro_revision r ON r.peticion_ref=s.peticion_ref AND r.version=2 LIMIT 1;
GRANT SELECT ON ct169_ensayo_material TO vec_ct169_ensayo_login;
SET SESSION AUTHORIZATION vec_ct169_ensayo_login;
DO $negativas$
DECLARE m text;
BEGIN
 BEGIN
  PERFORM * FROM vec_contratacion_temporal.preparar_entrega_peticion_centro_con_original_v1(
    '{"modo":"bandeja"}',NULL,NULL,NULL,NULL,1,1,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'CT169 aceptó bandeja';
 EXCEPTION WHEN SQLSTATE 'P0680' THEN NULL; END;
 BEGIN
  PERFORM * FROM vec_contratacion_temporal.preparar_entrega_peticion_centro_con_original_v1(
    '{"modo":"confirmar"}',NULL,NULL,NULL,NULL,1,1,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'CT169 aceptó confirmar';
 EXCEPTION WHEN SQLSTATE 'P0680' THEN NULL; END;
 BEGIN
  PERFORM * FROM vec_contratacion_temporal.preparar_entrega_peticion_centro_con_original_v1(
    NULL,NULL,NULL,NULL,NULL,1,1,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'CT169 aceptó material nulo';
 EXCEPTION WHEN SQLSTATE 'P0680' THEN NULL; END;
 SELECT material INTO STRICT m FROM ct169_ensayo_material;
 BEGIN
  PERFORM * FROM vec_contratacion_temporal.preparar_entrega_peticion_centro_con_original_v1(
    m,NULL,convert_to('{}','UTF8'),NULL,NULL,1,1,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'CT169 devolvió datos sin decisión CURRENT';
 EXCEPTION WHEN SQLSTATE 'P0683' THEN NULL; END;
 BEGIN
  PERFORM * FROM vec_contratacion_temporal.expediente_alta LIMIT 1;
  RAISE EXCEPTION 'ejecutor CT169 puede leer tabla directamente';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END
$negativas$;
RESET SESSION AUTHORIZATION;
DO $conservacion$
DECLARE r record;
BEGIN
 SELECT * INTO STRICT r FROM ct169_ensayo_material;
 IF r.accesos<>(SELECT count(*) FROM vec_contratacion_temporal.entrega_peticion_centro_acceso)
    OR r.confirmaciones<>(SELECT count(*) FROM vec_contratacion_temporal.entrega_peticion_centro_confirmacion)
    OR r.eventos<>(SELECT count(*) FROM vec_contratacion_temporal.entrega_peticion_centro_outbox) THEN
    RAISE EXCEPTION 'denegación CT169 alteró historia';
 END IF;
END
$conservacion$;
ROLLBACK;
\echo CT169-ACL-DENEGACION-OK
