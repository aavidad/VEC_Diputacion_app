\set ON_ERROR_STOP on
-- AD3-136 DOWN: solo para una base sin consumos de custodia firmada.
-- Retirar antes Documentos9 y AD3-113; nunca ejecutar contra historia conservada.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000136',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $nucleo$
DECLARE
 f regprocedure:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 w regprocedure;
 actual text; anterior text; restaurado text; meta jsonb; deps jsonb; hay_historia boolean;
 antiguo text:=$x$   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.externo.registrar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'documento_externo'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'registrar_documento_externo'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["documento","recibo"]'::jsonb)
 ))$x$;
 ampliado text:=$x$   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.externo.registrar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'documento_externo'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'registrar_documento_externo'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["documento","recibo"]'::jsonb)
   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.firmado.custodiar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'documento_firmado'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'custodiar_documento_firmado'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["documento_firmado.custodia","evidencia_custodia"]'::jsonb)
 ))$x$;
BEGIN
 w:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF current_user<>'vec_autorizacion_atestada_v3_propietario' OR w IS NULL
    OR md5(pg_get_functiondef(w))<>'2f9f7fe8b8508a6b80f95a56942cade5'
    OR to_regprocedure('vec_documentos.custodiar_firmado_v1(bytea,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regclass('vec_autorizacion_atestada_v3.consumo_decision_v3') IS NULL
    OR to_regclass('vec_autorizacion_atestada_v3.atestacion_decision_v3') IS NULL
 THEN RAISE EXCEPTION 'AD3-136 DOWN: retire antes Documentos9 y AD3-113' USING ERRCODE='55000'; END IF;
 SELECT EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.consumo_decision_v3 c
   JOIN vec_autorizacion_atestada_v3.atestacion_decision_v3 a USING (decision_ref)
   WHERE convert_from(a.decision_canonica,'UTF8')::jsonb->>'accion'='documentos.firmado.custodiar') INTO hay_historia;
 IF hay_historia THEN RAISE EXCEPTION 'AD3-136 DOWN: hay custodias consumidas' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc' INTO STRICT actual,meta
   FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
      AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s'];
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
   INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF length(actual)-length(replace(actual,ampliado,''))<>length(ampliado)
    OR length(actual)-length(replace(actual,antiguo,''))<>0
 THEN RAISE EXCEPTION 'AD3-136 DOWN: postimagen incompatible' USING ERRCODE='55000'; END IF;
 anterior:=replace(actual,ampliado,antiguo);
 EXECUTE anterior;
 SELECT pg_get_functiondef(f) INTO STRICT restaurado;
 IF restaurado IS DISTINCT FROM anterior OR replace(restaurado,antiguo,ampliado) IS DISTINCT FROM actual
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'AD3-136 DOWN: restauración inexacta' USING ERRCODE='55000'; END IF;
END $nucleo$;
COMMIT;
