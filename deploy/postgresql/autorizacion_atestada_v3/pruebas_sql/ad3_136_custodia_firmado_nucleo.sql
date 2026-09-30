\set ON_ERROR_STOP on
-- BORRADOR NO ACREDITADO hasta medir la postimagen de AD3-136 tras B.
-- Sonda de lectura; no consume decisiones ni cambia historia.
BEGIN;
SET LOCAL search_path=pg_catalog, pg_temp;
SET LOCAL statement_timeout='10s';
DO $prueba$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
        d text; p record; coincidencias integer;
        -- Medir sólo sobre el linaje completo B→AD136 en PG18.
        esperada_def_sha256 text:=NULL;
        esperada_wrapper_sha256 text:=NULL;
        caso text:=$x$   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.firmado.custodiar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'documento_firmado'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'custodiar_documento_firmado'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["documento_firmado.custodia","evidencia_custodia"]'::jsonb)$x$;
BEGIN
 SELECT q.* INTO STRICT p FROM pg_proc q WHERE q.oid=f;
 d:=pg_get_functiondef(f);
 coincidencias:=(length(d)-length(replace(d,caso,'')))/length(caso);
 IF coincidencias<>1 OR p.proowner<>'vec_autorizacion_atestada_v3_propietario'::regrole
    OR NOT p.prosecdef OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    OR encode(sha256(convert_to(d,'UTF8')),'hex') IS DISTINCT FROM esperada_def_sha256
    OR EXISTS (SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
               WHERE a.grantee<>p.proowner OR a.privilege_type<>'EXECUTE')
 THEN RAISE EXCEPTION 'AD3-136: caso exacto, propietario, configuración o ACL inválidos'; END IF;
 IF NOT (d LIKE '%''documentos.generado.alta''%'
     AND d LIKE '%''documentos.expediente.listar''%'
     AND d LIKE '%''documentos.original.descargar''%'
     AND d LIKE '%''documentos.notificacion.preparar''%'
     AND d LIKE '%''documentos.externo.registrar''%')
 THEN RAISE EXCEPTION 'AD3-136: faltan operaciones documentales previas'; END IF;
 IF encode(sha256(convert_to(pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'UTF8')),'hex')
      IS DISTINCT FROM esperada_wrapper_sha256
 THEN RAISE EXCEPTION 'AD3-136: consumidor AD3-113 modificado'; END IF;
END $prueba$;
ROLLBACK;
