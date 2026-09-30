\set ON_ERROR_STOP on
-- Sonda de lectura tras AD3-136; no consume decisiones ni cambia historia.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL statement_timeout='10s';
DO $prueba$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
        d text; p record; coincidencias integer;
        caso text:=$x$   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.firmado.custodiar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'documento_firmado'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'custodiar_documento_firmado'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["documento_firmado.custodia","evidencia_custodia"]'::jsonb)$x$;
BEGIN
 SELECT q.* INTO STRICT p FROM pg_proc q WHERE q.oid=f;
 d:=pg_get_functiondef(f);
 coincidencias:=(length(d)-length(replace(d,caso,'')))/length(caso);
 IF coincidencias<>1 OR p.proowner<>'vec_autorizacion_atestada_v3_propietario'::regrole
    OR NOT p.prosecdef OR p.proconfig<>ARRAY['search_path=pg_catalog','lock_timeout=2s']
    OR EXISTS (SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE a.grantee=0)
 THEN RAISE EXCEPTION 'AD3-136: caso exacto, propietario, configuración o ACL inválidos'; END IF;
 IF NOT (d LIKE '%''documentos.generado.alta''%'
     AND d LIKE '%''documentos.expediente.listar''%'
     AND d LIKE '%''documentos.original.descargar''%'
     AND d LIKE '%''documentos.notificacion.preparar''%'
     AND d LIKE '%''documentos.externo.registrar''%')
 THEN RAISE EXCEPTION 'AD3-136: faltan operaciones documentales previas'; END IF;
 IF md5(pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure))
      <>'b4e998fd572f738c9d30522565af72cd'
 THEN RAISE EXCEPTION 'AD3-136: consumidor AD3-113 modificado'; END IF;
END $prueba$;
ROLLBACK;
