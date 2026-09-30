\set ON_ERROR_STOP on
-- AD3-113 DOWN: devuelve el consumidor documental con recuperación a AD3-62
-- exacta. Se niega si queda instalada la función de custodia de Documentos
-- (Documentos 000009 se retira antes) o si ya se consumió alguna decisión
-- documentos.firmado.custodiar: esa historia no se deja sin consumidor.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000113',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $m$
DECLARE
 f regprocedure;
 actual text; original text; meta jsonb; acl aclitem[]; consumidas boolean;
 lista text:=$x$      AND d->>'finalidad'='registrar_documento_externo' AND d->'campos_permitidos'='["documento","recibo"]'::jsonb))$x$;
 lista_nueva text:=$x$      AND d->>'finalidad'='registrar_documento_externo' AND d->'campos_permitidos'='["documento","recibo"]'::jsonb)
     OR (accion='documentos.firmado.custodiar' AND d->>'tipo_recurso'='documento_firmado'
      AND d->>'finalidad'='custodiar_documento_firmado' AND d->'campos_permitidos'='["documento_firmado.custodia","evidencia_custodia"]'::jsonb))$x$;
 replay text:=$x$  IF accion NOT IN ('documentos.generado.alta','documentos.notificacion.preparar','documentos.externo.registrar')$x$;
 replay_nuevo text:=$x$  IF accion NOT IN ('documentos.generado.alta','documentos.notificacion.preparar','documentos.externo.registrar','documentos.firmado.custodiar')$x$;
BEGIN
 f:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF current_user<>'vec_autorizacion_atestada_v3_propietario' OR f IS NULL
    OR EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
               WHERE n.nspname='vec_documentos' AND p.proname='custodiar_firmado_v1')
 THEN RAISE EXCEPTION 'AD3-113 DOWN: retire antes Documentos 000009' USING ERRCODE='55000'; END IF;
 -- En la principal las tablas del núcleo existen siempre; solo un entorno de
 -- prueba con núcleo sustituido puede no tenerlas (y entonces no hay consumos).
 IF to_regclass('vec_autorizacion_atestada_v3.consumo_decision_v3') IS NOT NULL THEN
  EXECUTE $q$SELECT EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.consumo_decision_v3 c
             JOIN vec_autorizacion_atestada_v3.atestacion_decision_v3 a USING (decision_ref)
            WHERE convert_from(a.decision_canonica,'UTF8')::jsonb->>'accion'='documentos.firmado.custodiar')$q$ INTO consumidas;
  IF consumidas THEN
   RAISE EXCEPTION 'AD3-113 DOWN: hay custodias de documentos firmados consumidas' USING ERRCODE='55000';
  END IF;
 END IF;
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl INTO STRICT actual,meta,acl FROM pg_proc p WHERE p.oid=f;
 IF md5(actual)<>'b4e998fd572f738c9d30522565af72cd' THEN
  RAISE EXCEPTION 'AD3-113 DOWN: consumidor distinto del instalado por AD3-113' USING ERRCODE='55000';
 END IF;
 original:=replace(replace(actual,lista_nueva,lista),replay_nuevo,replay);
 EXECUTE original;
 IF md5(pg_get_functiondef(f))<>'2f9f7fe8b8508a6b80f95a56942cade5'
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
 THEN RAISE EXCEPTION 'AD3-113 DOWN: restauración inexacta' USING ERRCODE='55000'; END IF;
END $m$;
COMMIT;
