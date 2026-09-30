\set ON_ERROR_STOP on
-- AD3-136: la custodia admitida por AD3-113 debe pasar también la ligadura
-- interna del núcleo AD3. La única ampliación es el sexto caso documental.
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
 original text; nuevo text; actual text; meta jsonb; deps jsonb;
 doc9_total integer; doc9_exacto integer;
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
 -- El propietario AD3 carece de USAGE en vec_documentos. Resolver el nombre
 -- mediante regprocedure exigiría ese permiso; el catálogo permite comprobar
 -- la firma y descartar cualquier sobrecarga sin conceder acceso al esquema.
 SELECT count(*),count(*) FILTER (WHERE p.prokind='f' AND p.pronargs=13
   AND p.proowner='vec_documentos_propietario'::regrole
   AND p.proargtypes=ARRAY[
    'pg_catalog.bytea'::regtype::oid,'pg_catalog.jsonb'::regtype::oid,'pg_catalog.jsonb'::regtype::oid,
    'pg_catalog.bytea'::regtype::oid,'pg_catalog.bytea'::regtype::oid,'pg_catalog.bytea'::regtype::oid,
    'pg_catalog.bytea'::regtype::oid,'pg_catalog.numeric'::regtype::oid,'pg_catalog.numeric'::regtype::oid,
    'pg_catalog.bytea'::regtype::oid,'pg_catalog.bytea'::regtype::oid,'pg_catalog.bytea'::regtype::oid,
    'pg_catalog.bytea'::regtype::oid]::oidvector)
   INTO doc9_total,doc9_exacto FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
   WHERE n.nspname='vec_documentos' AND p.proname='custodiar_firmado_v1';
 IF current_user<>'vec_autorizacion_atestada_v3_propietario' OR w IS NULL
    OR md5(pg_get_functiondef(w))<>'b4e998fd572f738c9d30522565af72cd'
    OR doc9_total<>1 OR doc9_exacto<>1
 THEN RAISE EXCEPTION 'AD3-136: AD3-113 o Documentos9 incompatible' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc' INTO STRICT original,meta
   FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
      AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s'];
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
   INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF length(original)-length(replace(original,antiguo,''))<>length(antiguo)
    OR strpos(original,'operacion_documentos_comunes')=0
    OR strpos(original,'documentos.firmado.custodiar')<>0
 THEN RAISE EXCEPTION 'AD3-136: preimagen del núcleo incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,antiguo,ampliado);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR replace(actual,ampliado,antiguo) IS DISTINCT FROM original
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'AD3-136: núcleo alterado fuera del contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;
COMMIT;
