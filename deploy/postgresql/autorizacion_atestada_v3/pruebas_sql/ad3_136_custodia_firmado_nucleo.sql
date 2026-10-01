\set ON_ERROR_STOP on
-- Sonda de lectura post-AD3-136 sobre B c6b29fd4; no consume decisiones.
BEGIN;
SET LOCAL search_path=pg_catalog, pg_temp;
SET LOCAL statement_timeout='10s';
DO $prueba$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
        d text; p record; coincidencias integer;
        -- Captura PG18 post-AD136 SHA256 6984576e2bdc43c7f178c2d7c74d6c0391c63718d164882e4e5db39c0b2eb15a.
        esperada_def_sha256 text:='c5612fc12e96f462ab925157d1a394a18af67339836f88773178768adf6d7da7';
        esperada_fuente_sha256 text:='2829f6cae1ad11d31832c53f1320bb8db976a9d21a4710fbb855d9c8939b9e8c';
        esperada_wrapper_sha256 text:='3e4412dbac79bcce6d0425f4c0fddfbb32602ed2169b9d694c2292fbb83a5d4f';
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
    OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM esperada_fuente_sha256
    OR EXISTS (SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
               WHERE a.grantee<>p.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable)
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
