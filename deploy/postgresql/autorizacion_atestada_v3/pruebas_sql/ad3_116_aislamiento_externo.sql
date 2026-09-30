\set ON_ERROR_STOP on
-- Sonda sin escritura. Ejecutar tras Contexto-12, AUT-15, B-59, AD3-115/116.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL statement_timeout='10s';
DO $prueba$
DECLARE n text; t regclass; f regprocedure;
        codigo text; interno text;
BEGIN
 FOREACH n IN ARRAY ARRAY['atestacion_decision_v3_externa','consumo_decision_v3_externa',
                             'auditoria_consumo_v3_externa','control_cadena_auditoria_externa'] LOOP
  t:=pg_catalog.to_regclass('vec_autorizacion_atestada_v3.'||n);
  IF t IS NULL OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class c WHERE c.oid=t
      AND c.relowner='vec_autorizacion_atestada_v3_propietario'::regrole
      AND c.relrowsecurity AND c.relforcerowsecurity)
     OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_policies p
          WHERE p.schemaname='vec_autorizacion_atestada_v3' AND p.tablename=n
            AND p.policyname='propietario_exacto'
            AND p.roles::text[]=ARRAY['vec_autorizacion_atestada_v3_propietario'])
     OR EXISTS (SELECT 1 FROM pg_catalog.pg_class c,
           LATERAL pg_catalog.aclexplode(coalesce(c.relacl,pg_catalog.acldefault('r',c.relowner))) a
           WHERE c.oid=t AND a.grantee=0)
  THEN RAISE EXCEPTION 'AD3-116: aislamiento roto en %',n; END IF;
 END LOOP;
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_constraint
       WHERE conrelid='vec_autorizacion_atestada_v3.atestacion_decision_v3_externa'::regclass
         AND confrelid='vec_autorizacion.decision_concedida_contexto_actor_v3_externa'::regclass
         AND contype='f')
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_constraint
       WHERE conrelid='vec_autorizacion_atestada_v3.atestacion_decision_v3_externa'::regclass
         AND confrelid='vec_autorizacion.decision_concedida_contexto_actor_v3'::regclass)
    OR NOT EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.control_cadena_auditoria_externa
       WHERE control_id AND secuencia=0 AND cabeza_sha256=repeat('0',64))
 THEN RAISE EXCEPTION 'AD3-116: FK o cadena externa incorrecta'; END IF;
 f:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_externa_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 SELECT p.prosrc INTO STRICT codigo FROM pg_catalog.pg_proc p WHERE p.oid=f
   AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
   AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s'];
 IF codigo !~ 'session_user <> ''vec_externo_bolsa_desarrollo'''
    OR codigo !~ 'portal_candidato_bolsa'
    OR codigo !~ 'consulta_participaciones_propias_bolsa'
    OR codigo !~ 'registrar_y_revalidar_decision_contexto_actor_externa_v3'
    OR codigo !~ 'revalidar_decision_contexto_actor_externa_v3_viva'
    OR codigo ~ 'vec_autorizacion_atestada_v3\.(atestacion_decision_v3|consumo_decision_v3|auditoria_consumo_v3|control_cadena_auditoria)([^_a-z]|$)'
    OR codigo ~ 'registrar_y_revalidar_decision_contexto_actor_v3\('
    OR codigo ~ 'revalidar_decision_contexto_actor_v3_viva\('
 THEN RAISE EXCEPTION 'AD3-116: carril externo no aislado'; END IF;
 IF EXISTS (SELECT 1 FROM pg_catalog.pg_proc p,
     LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
     WHERE p.oid=f AND a.privilege_type='EXECUTE'
       AND (a.grantee=0 OR a.grantee='vec_bolsa_llamamientos_portal_externo'::regrole))
 THEN RAISE EXCEPTION 'AD3-116: función externa expuesta'; END IF;
 SELECT p.prosrc INTO STRICT interno FROM pg_catalog.pg_proc p
   WHERE p.oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 IF interno !~ 'session_user = ''vec_externo_bolsa_desarrollo'''
    OR interno !~ 'consumir_decision_mutacion_v3_externa_interna'
    OR interno !~ 'vec_autorizacion_atestada_v3\.atestacion_decision_v3([^_a-z]|$)'
    OR interno !~ 'registrar_y_revalidar_decision_contexto_actor_v3'
 THEN RAISE EXCEPTION 'AD3-116: despacho interno incorrecto'; END IF;
END $prueba$;
ROLLBACK;
