\set ON_ERROR_STOP on
-- AD209 + CT185: huella del recurso exterior (plan) de firma V2 con los
-- ámbitos de la asignación de quien firma. Como superusuario, sobre un clon
-- con asignaciones vigentes de sólo organización y de organización y unidad.
-- No escribe: todo en ROLLBACK.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
CREATE FUNCTION pg_temp.ok(caso text,cond boolean) RETURNS text LANGUAGE plpgsql AS $f$
BEGIN IF cond IS NOT TRUE THEN RAISE EXCEPTION 'FALLO %',caso; END IF; RETURN 'OK '||caso; END $f$;
-- Una asignación actual, activa y vigente con exactamente las dimensiones pedidas.
CREATE FUNCTION pg_temp.asignacion(dims text) RETURNS jsonb LANGUAGE sql AS $f$
 SELECT jsonb_build_object('asignacion_ref',a.asignacion_ref,'asignacion_huella_sha256',a.huella_sha256,'principal_id',a.principal_id,
  'perfil_activo_ref',a.perfil_activo_ref,'version_rol_ref',a.version_rol_ref,
  'org',(SELECT e->'valores'->>0 FROM jsonb_array_elements(a.documento->'ambitos') e WHERE e->>'clave'='organizacion_ref'),
  'uni',(SELECT e->'valores'->>0 FROM jsonb_array_elements(a.documento->'ambitos') e WHERE e->>'clave'='unidad_ref'))
 FROM vec_autorizacion.asignacion_perfil_actual p JOIN vec_autorizacion.asignacion_perfil a USING(perfil_activo_ref,asignacion_ref)
 WHERE a.documento->>'estado'='activa' AND clock_timestamp()>=(a.documento->>'vigente_desde')::timestamptz
  AND clock_timestamp()<(a.documento->>'vigente_hasta')::timestamptz
  AND (SELECT string_agg(e->>'clave',',' ORDER BY e->>'clave') FROM jsonb_array_elements(a.documento->'ambitos') e)=dims
 ORDER BY a.asignacion_ref LIMIT 1
$f$;
-- Huella exterior a través de AD209, ejecutada como el propietario CT (quien la usa).
CREATE FUNCTION pg_temp.huella(sol text,env bytea,decision jsonb) RETURNS text LANGUAGE plpgsql AS $f$
DECLARE r text;
BEGIN
 SET LOCAL ROLE vec_contratacion_temporal_propietario;
 BEGIN r:=vec_autorizacion_atestada_v3.huella_recurso_plan_firma_ct_v1(sol,env,convert_to(decision::text,'UTF8'));
 EXCEPTION WHEN insufficient_privilege THEN r:='denegado:'||SQLERRM;
 END;
 RESET ROLE;
 RETURN r;
END $f$;
CREATE FUNCTION pg_temp.canon(org text,uni text,sol text,env bytea) RETURNS text LANGUAGE sql AS $f$
 SELECT encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||org||'"'||CASE WHEN uni IS NULL THEN '' ELSE ',"unidad_ref":"'||uni||'"' END||
  '},"atributos":{"material_sha256":"'||encode(sha256(convert_to(sol,'UTF8')),'hex')||'","plan_firma_sha256":"'||encode(sha256(env),'hex')||'"}}','UTF8')),'hex')
$f$;
-- 0. Valor común con Go (firmaemisorv2.TestFirmaV2HuellaPlanConUnidadIgualQueSQL):
-- mismas entradas, misma huella que calcula el PDP.
SELECT pg_temp.ok('canon_comun_con_go',pg_temp.canon('org_fija','unidad:fija',
 '{"Via":"certificado_vec","OrganizacionRef":"org_fija","UnidadFirmanteRef":"unidad:fija"}','\x7b2265223a317d'::bytea)
 ='14b23972f887f011031a06ba4ae7a7798548a3532211f8712f2a8d1a448df0db');
SELECT pg_temp.asignacion('organizacion_ref') AS a1, pg_temp.asignacion('organizacion_ref,unidad_ref') AS a2 \gset
SELECT pg_temp.ok('hay_asignaciones',(:'a1'::jsonb)->>'org' IS NOT NULL AND (:'a2'::jsonb)->>'uni' IS NOT NULL);
\set env '\\x7b2265223a317d'
-- 1. Sólo organización: la misma huella que calculaban AD177 v2 y CT176/CT181.
SELECT jsonb_build_object('Via','certificado_vec','OrganizacionRef',(:'a1'::jsonb)->>'org','UnidadFirmanteRef','unidad:del:paso')::text AS s1 \gset
SELECT pg_temp.ok('solo_organizacion_igual_que_v2',pg_temp.huella(:'s1',:'env'::bytea,:'a1'::jsonb)=pg_temp.canon((:'a1'::jsonb)->>'org',NULL,:'s1',:'env'::bytea));
-- 2. Organización y unidad, vía VEC, unidad del paso igual a la de la asignación: huella con unidad.
SELECT jsonb_build_object('Via','certificado_vec','OrganizacionRef',(:'a2'::jsonb)->>'org','UnidadFirmanteRef',(:'a2'::jsonb)->>'uni')::text AS s2 \gset
SELECT pg_temp.ok('unidad_del_paso_en_la_huella',pg_temp.huella(:'s2',:'env'::bytea,:'a2'::jsonb)=pg_temp.canon((:'a2'::jsonb)->>'org',(:'a2'::jsonb)->>'uni',:'s2',:'env'::bytea)
 AND pg_temp.huella(:'s2',:'env'::bytea,:'a2'::jsonb)<>pg_temp.canon((:'a2'::jsonb)->>'org',NULL,:'s2',:'env'::bytea));
-- 3. Otra unidad en el paso: denegado.
SELECT pg_temp.ok('otra_unidad_denegada',pg_temp.huella(jsonb_set(:'s2'::jsonb,'{UnidadFirmanteRef}','"unidad:otra"')::text,:'env'::bytea,:'a2'::jsonb) LIKE 'denegado:%');
-- 4. Vía externa con asignación con unidad: aún no admitida.
SELECT pg_temp.ok('externa_con_unidad_denegada',pg_temp.huella(jsonb_set(:'s2'::jsonb,'{Via}','"portafirmas_registro_rrhh"')::text,:'env'::bytea,:'a2'::jsonb) LIKE 'denegado:%');
-- 5. Organización del material distinta de la de la asignación: denegado.
SELECT pg_temp.ok('otra_organizacion_denegada',pg_temp.huella(jsonb_set(:'s1'::jsonb,'{OrganizacionRef}','"org_otra_sintetica"')::text,:'env'::bytea,:'a1'::jsonb) LIKE 'denegado:%');
-- 6. La decisión nombra una asignación con otra huella, otro perfil o otro principal: denegado.
SELECT pg_temp.ok('asignacion_alterada_denegada',
 pg_temp.huella(:'s1',:'env'::bytea,jsonb_set(:'a1'::jsonb,'{asignacion_huella_sha256}',to_jsonb(repeat('0',64)))) LIKE 'denegado:%'
 AND pg_temp.huella(:'s1',:'env'::bytea,jsonb_set(:'a1'::jsonb,'{perfil_activo_ref}','"prf_ajeno"')) LIKE 'denegado:%'
 AND pg_temp.huella(:'s1',:'env'::bytea,jsonb_set(:'a1'::jsonb,'{principal_id}','"per_ajeno"')) LIKE 'denegado:%');
-- 7. La v4 calcula las dos huellas con AD206/AD209 y consume con AD177 v3; el
-- ejecutor sólo llega a la v4; la v3 y AD177 v2 quedan cerradas.
SELECT pg_temp.ok('v4_con_ad209_y_v3_cerrada',
 strpos(pg_get_functiondef('vec_contratacion_temporal.registrar_firma_con_plan_v4(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'huella_recurso_plan_firma_ct_v1(p_solicitud,p_envoltorio,p_dec_e)')>0
 AND strpos(pg_get_functiondef('vec_contratacion_temporal.registrar_firma_con_plan_v4(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'consumir_plan_firma_ct_v3_atestada(')>0
 AND strpos(pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_plan_firma_ct_v3_atestada(text,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'huella_recurso_plan_firma_ct_v1(p_solicitud,p_envoltorio,p_decision)')>0
 AND has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.registrar_firma_con_plan_v4(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND NOT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.registrar_firma_con_plan_v3(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND NOT has_function_privilege('vec_contratacion_temporal_propietario','vec_autorizacion_atestada_v3.consumir_plan_firma_ct_v2_atestada(text,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND NOT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_autorizacion_atestada_v3.huella_recurso_plan_firma_ct_v1(text,bytea,bytea)','EXECUTE')
 AND NOT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_autorizacion_atestada_v3.consumir_plan_firma_ct_v3_atestada(text,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE'));
-- 9. Unidad no canónica en la asignación (con comilla): se deniega y no entra
-- en la cadena de la huella. Se siembra sólo dentro de este ROLLBACK.
SET LOCAL session_replication_role=replica;
UPDATE vec_autorizacion.asignacion_perfil SET documento=jsonb_set(documento,'{ambitos}',(SELECT jsonb_agg(CASE WHEN e->>'clave'='unidad_ref'
  THEN jsonb_build_object('clave','unidad_ref','valores',jsonb_build_array('unidad"x')) ELSE e END) FROM jsonb_array_elements(documento->'ambitos') e))
 WHERE asignacion_ref=(:'a2'::jsonb)->>'asignacion_ref';
SET LOCAL session_replication_role=origin;
SELECT pg_temp.ok('unidad_no_canonica_denegada',pg_temp.huella(jsonb_set(:'s2'::jsonb,'{UnidadFirmanteRef}','"unidad\"x"')::text,:'env'::bytea,:'a2'::jsonb) LIKE 'denegado:%');
ROLLBACK;
