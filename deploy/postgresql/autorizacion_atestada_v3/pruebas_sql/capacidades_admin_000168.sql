\set ON_ERROR_STOP on
-- Puerta sin provisión: ACL, preimagen ampliada y rechazos anteriores al consumo.
-- Favorable/acuse y negativos de caducidad/huella requieren fixture nominal real
-- emitido para un LOGIN privado miembro del lector. No se usa un doble de V3.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
DO $frontera$
DECLARE f regprocedure:='vec_autorizacion.consultar_capacidades_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 consumidor regprocedure:='vec_autorizacion_atestada_v3.consumir_capacidades_admin_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 otra text;n integer;
BEGIN
 IF NOT pg_catalog.has_function_privilege('vec_admin_perfiles_lector',f,'EXECUTE')
 OR pg_catalog.has_function_privilege('vec_admin_perfiles_ejecutor',f,'EXECUTE')
 OR pg_catalog.has_function_privilege('vec_admin_perfiles_lector',consumidor,'EXECUTE')
 OR NOT pg_catalog.has_function_privilege('vec_autorizacion_propietario',consumidor,'EXECUTE')
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
 WHERE p.oid IN (f,consumidor) AND (a.grantee=0 OR a.is_grantable))
 THEN RAISE EXCEPTION 'AD168: ACL de lectura divergente'; END IF;
 SELECT count(*) INTO n FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace s ON s.oid=p.pronamespace
 CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
 WHERE s.nspname LIKE 'vec_%' AND a.grantee='vec_admin_perfiles_lector'::regrole;
 IF n<>1 THEN RAISE EXCEPTION 'AD168: lector tiene otras funciones concedidas'; END IF;
 FOREACH otra IN ARRAY ARRAY['buscar_personas_admin_v1','consultar_persona_admin_v1','listar_roles_admin_v1',
 'listar_propuestas_admin_v1','consultar_propuesta_admin_v1','consultar_recibo_admin_v1'] LOOP
 IF pg_catalog.to_regprocedure('vec_autorizacion.'||otra||'(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'AD168: otra consulta publicada %',otra; END IF; END LOOP;
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_propietario'::regrole
 AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s'])
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=consumidor AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
 AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
 THEN RAISE EXCEPTION 'AD168: propietario o configuración divergentes'; END IF;
END $frontera$;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $negativos_material$
DECLARE m jsonb;d jsonb;c jsonb;caso text;
BEGIN
 m:=pg_catalog.jsonb_build_object('esquema','administracion_perfiles_lectura_v1','consulta','capacidades',
 'operacion_ref','prf_'||repeat('a',22),'actor_persona_ref','per_'||repeat('b',22),
 'actor_perfil_ref','prf_'||repeat('a',22),'asignacion_perfil_ref','asignacion:sintetica:v1',
 'correlacion_ref','correlacion_'||repeat('c',32),'limite',100);
 d:=pg_catalog.jsonb_build_object('principal_id',m->>'actor_persona_ref','perfil_activo_ref',m->>'actor_perfil_ref',
 'asignacion_ref',m->>'asignacion_perfil_ref','correlacion_ref',m->>'correlacion_ref');
 c:=pg_catalog.jsonb_build_object('efecto_ref',m->>'operacion_ref','huella_efecto_sha256',
 pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(m::text,'UTF8')),'hex'));
 FOREACH caso IN ARRAY ARRAY['correlacion_ajena','correlacion_raw','actor_ajeno','huella_material','entrada_nula'] LOOP
 BEGIN
  PERFORM vec_autorizacion.consultar_capacidades_admin_v1(
   CASE WHEN caso='entrada_nula' THEN NULL ELSE m::text END,pg_catalog.convert_to(CASE WHEN caso='huella_material' THEN pg_catalog.jsonb_set(c,'{huella_efecto_sha256}',pg_catalog.to_jsonb(repeat('0',64)))::text ELSE c::text END,'UTF8'),
   pg_catalog.convert_to(CASE caso
    WHEN 'correlacion_ajena' THEN pg_catalog.jsonb_set(d,'{correlacion_ref}',pg_catalog.to_jsonb('correlacion_'||repeat('d',32)))::text
    WHEN 'correlacion_raw' THEN pg_catalog.jsonb_set(d,'{correlacion_ref}',pg_catalog.to_jsonb(repeat('c',32)))::text
    WHEN 'actor_ajeno' THEN pg_catalog.jsonb_set(d,'{principal_id}',pg_catalog.to_jsonb('per_'||repeat('e',22)))::text
    ELSE d::text END,'UTF8'),NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'AD168: material inválido aceptado %',caso;
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 END LOOP;
END $negativos_material$;
DO $audiencia$
BEGIN
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.consumir_capacidades_admin_v3_atestada(
   pg_catalog.convert_to('{"audiencia_consumo":"vec_autorizacion.administracion_perfiles.lectura.buscar_personas.v1"}','UTF8'),
   pg_catalog.convert_to('{}','UTF8'),NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'AD168: audiencia distinta aceptada';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $audiencia$;
RESET ROLE;
ROLLBACK;
