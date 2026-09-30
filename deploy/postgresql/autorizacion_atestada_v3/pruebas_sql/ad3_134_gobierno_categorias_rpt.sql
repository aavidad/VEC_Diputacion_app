\set ON_ERROR_STOP on
-- Sondas de ACL y forma, en clon con Cat4 y AD3-134. No fabrican V3.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='20s';
SET LOCAL idle_in_transaction_session_timeout='25s';
DO $prueba$
DECLARE f regprocedure;
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname='vec_catalogos_configurables_gobierno_ejecutor'
    AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls)
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members WHERE member='vec_catalogos_configurables_gobierno_ejecutor'::regrole
       OR roleid='vec_catalogos_configurables_gobierno_ejecutor'::regrole)
    OR NOT pg_catalog.has_schema_privilege('vec_catalogos_configurables_gobierno_ejecutor','vec_autorizacion_atestada_v3','USAGE') THEN
  RAISE EXCEPTION 'rol de gobierno no aislado';
 END IF;
 FOREACH f IN ARRAY ARRAY[
  'vec_autorizacion_atestada_v3.proponer_gobierno_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_autorizacion_atestada_v3.aprobar_gobierno_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_autorizacion_atestada_v3.confirmar_gobierno_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
 ] LOOP
  IF NOT pg_catalog.has_function_privilege('vec_catalogos_configurables_gobierno_ejecutor',f,'EXECUTE')
     OR pg_catalog.has_function_privilege('vec_personal_ejecutor',f,'EXECUTE')
     OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
       CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
       WHERE p.oid=f AND a.grantee=0) THEN
   RAISE EXCEPTION 'ACL de fachada incompatible';
  END IF;
 END LOOP;
 IF pg_catalog.strpos(pg_catalog.pg_get_functiondef(
   'vec_autorizacion_atestada_v3.ejecutar_gobierno_categoria_rpt_v3_interna(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),
   'revision<>2 OR consulta#>>''{propuesta,revision}'' NOT IN (''2'',''3'')')=0 THEN
  RAISE EXCEPTION 'recuperacion de confirmacion revision 3 ausente';
 END IF;
 IF pg_catalog.strpos(pg_catalog.pg_get_functiondef(
   'vec_autorizacion_atestada_v3.autorizar_gobierno_categoria_rpt_v3_interna(jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),
   'ca->>''persona_ref'' IS DISTINCT FROM d->>''principal_id''')=0
 OR pg_catalog.strpos(pg_catalog.pg_get_functiondef(
   'vec_autorizacion_atestada_v3.ejecutar_gobierno_categoria_rpt_v3_interna(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),
   'p_decision,p_motivo,p_persona_version,p_perfil_version')=0 THEN
  RAISE EXCEPTION 'persona canonica o revalidacion viva ausentes';
 END IF;
 IF pg_catalog.has_function_privilege('vec_catalogos_configurables_gobierno_ejecutor',
   'vec_autorizacion_atestada_v3.autorizar_gobierno_categoria_rpt_v3_interna(jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR pg_catalog.has_function_privilege('vec_catalogos_configurables_gobierno_ejecutor',
   'vec_autorizacion_atestada_v3.acreditar_aprobacion_historica_gobierno_categoria_rpt_v3_interna(jsonb,text,text,text,text)','EXECUTE')
    OR pg_catalog.has_function_privilege('vec_catalogos_configurables_gobierno_ejecutor',
   'vec_catalogos_configurables.confirmar_propuesta_gobierno(text,text,bigint,text,text,text,text,text)','EXECUTE') THEN
  RAISE EXCEPTION 'helper o core expuesto';
 END IF;
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.proponer_gobierno_categoria_rpt_v3_atestada(
   '{}'::jsonb,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'propuesta incompleta admitida';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.proponer_gobierno_categoria_rpt_v3_atestada(
   '{"propuesta_ref":"prop*uno","contenido":{},"huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","recibo_ref":"recibo:uno"}'::jsonb,
   NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'referencia comodin admitida';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.aprobar_gobierno_categoria_rpt_v3_atestada(
   '{"propuesta_ref":"prop:uno","huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","recibo_ref":"recibo:uno","revision_esperada":"1","catalogo_id":"rpt-demo","modulo_id":"personal"}'::jsonb,
   NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'revision como texto admitida';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.confirmar_gobierno_categoria_rpt_v3_atestada(
   '{"propuesta_ref":"prop:uno","huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","recibo_ref":"recibo:uno","revision_esperada":3,"catalogo_id":"rpt-demo","modulo_id":"personal","actor_ref":"falso"}'::jsonb,
   NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'campo actor del cliente admitido';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
END $prueba$;
ROLLBACK;
