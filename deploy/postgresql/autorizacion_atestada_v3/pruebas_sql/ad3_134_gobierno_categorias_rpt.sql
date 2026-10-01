\set ON_ERROR_STOP on
-- Solo clon desechable con postA, CA21, AUT25 final, IS10, Cat4 y AD3-134.
-- Sondas negativas del helper real y del orden antes del efecto; no fabrican V3.
-- Pendiente: positivo emitido, revocacion concurrente y ausencia de efectos por
-- denegacion nominal atravesando toda AD134. Requiere asignaciones nominales
-- y el arnes real de D (AUT25@a69a73a6 es candidato; propietario aun pendiente).
BEGIN ISOLATION LEVEL SERIALIZABLE;
-- Métrica mínima de efectos: no contiene documentos ni material de sesiones.
CREATE TEMP TABLE preimagen_ad134_nominal AS
 SELECT (SELECT count(*) FROM vec_catalogos_configurables.propuesta_gobierno) AS propuestas,
 (SELECT count(*) FROM vec_catalogos_configurables.aprobacion_gobierno) AS aprobaciones,
 (SELECT count(*) FROM vec_catalogos_configurables.confirmacion_gobierno) AS confirmaciones,
 (SELECT count(*) FROM vec_catalogos_configurables.outbox_gobierno) AS eventos,
 (SELECT count(*) FROM vec_catalogos_configurables.publicacion) AS publicaciones,
 (SELECT count(*) FROM vec_catalogos_configurables.historia) AS historias,
 (SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3) AS consumos,
 (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3) AS auditorias;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='20s';
SET LOCAL idle_in_transaction_session_timeout='25s';
DO $prueba$
DECLARE f regprocedure;
 nominal oid:=pg_catalog.to_regprocedure('vec_autorizacion.revalidar_gobierno_rpt_v1(jsonb,text,text,jsonb)');
 autorizar text; ejecutar text; inicio_nominal integer; resultado boolean;
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname='vec_catalogos_configurables_gobierno_ejecutor'
    AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls)
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members WHERE member='vec_catalogos_configurables_gobierno_ejecutor'::regrole
       OR roleid='vec_catalogos_configurables_gobierno_ejecutor'::regrole)
    OR NOT pg_catalog.has_schema_privilege('vec_catalogos_configurables_gobierno_ejecutor','vec_autorizacion_atestada_v3','USAGE') THEN
  RAISE EXCEPTION 'rol de gobierno no aislado';
 END IF;
 -- Ausencia, firma distinta o permisos amplios deben cerrar el preflight.
 IF nominal IS NULL
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p
   WHERE p.oid=nominal AND p.proowner='vec_autorizacion_propietario'::regrole
     AND p.prokind='f' AND p.prorettype='boolean'::regtype AND NOT p.proretset
     AND p.prosecdef AND p.provolatile='v' AND p.proconfig=ARRAY['search_path=pg_catalog'])
 OR pg_catalog.has_function_privilege('vec_autorizacion_atestada_v3_propietario',nominal,'EXECUTE') IS NOT TRUE
 OR pg_catalog.has_function_privilege('vec_catalogos_configurables_gobierno_ejecutor',nominal,'EXECUTE') IS TRUE
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p
   CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
   WHERE p.oid=nominal AND (a.grantee=0 OR a.privilege_type<>'EXECUTE' OR a.is_grantable
     OR a.grantee NOT IN(p.proowner,'vec_autorizacion_atestada_v3_propietario'::regrole))) THEN
  RAISE EXCEPTION 'contrato AUT25 ausente o incompatible';
 END IF;
 -- Helper AUT25 real: material ausente/incompatible nunca acredita nominalmente.
 IF vec_autorizacion.revalidar_gobierno_rpt_v1(NULL,'rpt-demo','personal',NULL) IS NOT FALSE
 OR vec_autorizacion.revalidar_gobierno_rpt_v1('{}'::jsonb,'rpt-demo','personal','{}'::jsonb) IS NOT FALSE THEN
  RAISE EXCEPTION 'helper real concede con material nominal ausente';
 END IF;
 -- Sonda del criterio booleano, no una decisión V3 ni una autoridad sustituta.
 FOREACH resultado IN ARRAY ARRAY[false,NULL::boolean] LOOP
  IF resultado IS TRUE THEN RAISE EXCEPTION 'criterio nominal concede FALSE o NULL'; END IF;
 END LOOP;
 autorizar:=pg_catalog.pg_get_functiondef(
  'vec_autorizacion_atestada_v3.autorizar_gobierno_categoria_rpt_v3_interna(jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure);
 ejecutar:=pg_catalog.pg_get_functiondef(
  'vec_autorizacion_atestada_v3.ejecutar_gobierno_categoria_rpt_v3_interna(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure);
 inicio_nominal:=pg_catalog.strpos(autorizar,'IF vec_autorizacion.revalidar_gobierno_rpt_v1(');
 IF inicio_nominal=0
 OR pg_catalog.strpos(autorizar,'d,alcance->>''catalogo_id'',alcance->>''modulo_id'',m) IS NOT TRUE THEN')=0
 OR inicio_nominal<pg_catalog.strpos(autorizar,'consumir_decision_mutacion_v3_interna(')
 OR inicio_nominal<pg_catalog.strpos(autorizar,'motivo_canonico:=')
 OR inicio_nominal>pg_catalog.strpos(autorizar,'RETURN QUERY SELECT x.decision_ref')
 OR pg_catalog.strpos(ejecutar,'autorizar_gobierno_categoria_rpt_v3_interna(')=0
 OR pg_catalog.strpos(ejecutar,'autorizar_gobierno_categoria_rpt_v3_interna(')>
    pg_catalog.strpos(ejecutar,'registrar_propuesta_gobierno(')
 OR pg_catalog.strpos(ejecutar,'autorizar_gobierno_categoria_rpt_v3_interna(')>
    pg_catalog.strpos(ejecutar,'consultar_aprobaciones_gobierno(')
 OR pg_catalog.strpos(ejecutar,'IF vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(')=0
 OR pg_catalog.strpos(ejecutar,'IF vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(')<
    pg_catalog.strpos(ejecutar,'confirmar_propuesta_gobierno(') THEN
  RAISE EXCEPTION 'nominal no precede todos los efectos y replay, o se perdio revalidacion final';
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
RESET ROLE;
DO $sin_efectos$
BEGIN
 IF EXISTS(SELECT 1 FROM pg_temp.preimagen_ad134_nominal p
  WHERE p.propuestas<>(SELECT count(*) FROM vec_catalogos_configurables.propuesta_gobierno)
   OR p.aprobaciones<>(SELECT count(*) FROM vec_catalogos_configurables.aprobacion_gobierno)
   OR p.confirmaciones<>(SELECT count(*) FROM vec_catalogos_configurables.confirmacion_gobierno)
   OR p.eventos<>(SELECT count(*) FROM vec_catalogos_configurables.outbox_gobierno)
   OR p.publicaciones<>(SELECT count(*) FROM vec_catalogos_configurables.publicacion)
   OR p.historias<>(SELECT count(*) FROM vec_catalogos_configurables.historia)
   OR p.consumos<>(SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3)
   OR p.auditorias<>(SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3)) THEN
  RAISE EXCEPTION 'las sondas negativas nominales dejaron efectos';
 END IF;
END $sin_efectos$;
ROLLBACK;
