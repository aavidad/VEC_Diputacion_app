\set ON_ERROR_STOP on
-- AD210 + CT186: huella del recurso de la consulta y la recuperación R5 V2
-- con los ámbitos de la asignación de quien consulta; la unidad, la de
-- UnidadRef del material. Como superusuario, sobre un clon con asignaciones
-- vigentes de sólo organización y de organización y unidad. No escribe: todo
-- en ROLLBACK. La comprobación de la unidad en el plan (CC10) tiene su propio
-- vector, cc10_paso_plan_unidad_positivo_clon.sql.
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
-- Huella a través de AD210, ejecutada como el propietario CT (quien la usa).
CREATE FUNCTION pg_temp.huella(sol text,decision jsonb) RETURNS text LANGUAGE plpgsql AS $f$
DECLARE r text;
BEGIN
 SET LOCAL ROLE vec_contratacion_temporal_propietario;
 BEGIN r:=vec_autorizacion_atestada_v3.huella_recurso_consulta_firmas_r5_ct_v1(sol,convert_to(decision::text,'UTF8'));
 EXCEPTION WHEN insufficient_privilege THEN r:='denegado:'||SQLERRM;
 END;
 RESET ROLE;
 RETURN r;
END $f$;
CREATE FUNCTION pg_temp.canon(org text,uni text,sol text) RETURNS text LANGUAGE sql AS $f$
 SELECT encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||org||'"'||CASE WHEN uni IS NULL THEN '' ELSE ',"unidad_ref":"'||uni||'"' END||
  '},"atributos":{"material_sha256":"'||encode(sha256(convert_to(sol,'UTF8')),'hex')||'"}}','UTF8')),'hex')
$f$;
-- 0. Valor común con Go (firmaemisorv2.TestFirmaV2HuellaConsultaConUnidadIgualQueSQL).
SELECT pg_temp.ok('canon_comun_con_go',pg_temp.canon('org_fija','unidad:fija',
 '{"Via":"certificado_vec","OrganizacionRef":"org_fija","UnidadRef":"unidad:fija"}')
 ='c051b4e351005621ea4eeaca10696497a6d59c545eac14de9c590a7d939cefcf');
SELECT pg_temp.asignacion('organizacion_ref') AS a1, pg_temp.asignacion('organizacion_ref,unidad_ref') AS a2 \gset
SELECT pg_temp.ok('hay_asignaciones',(:'a1'::jsonb)->>'org' IS NOT NULL AND (:'a2'::jsonb)->>'uni' IS NOT NULL);
-- 1. Sólo organización y UnidadRef nula: la misma huella que CT172/CT175 v2.
SELECT jsonb_build_object('Via','certificado_vec','OrganizacionRef',(:'a1'::jsonb)->>'org','UnidadRef',NULL)::text AS s1 \gset
SELECT pg_temp.ok('solo_organizacion_igual_que_v2',pg_temp.huella(:'s1',:'a1'::jsonb)=pg_temp.canon((:'a1'::jsonb)->>'org',NULL,:'s1'));
-- 2. Asignación con unidad y UnidadRef igual, vía VEC: huella con unidad.
SELECT jsonb_build_object('Via','certificado_vec','OrganizacionRef',(:'a2'::jsonb)->>'org','UnidadRef',(:'a2'::jsonb)->>'uni')::text AS s2 \gset
SELECT pg_temp.ok('unidad_en_la_huella',pg_temp.huella(:'s2',:'a2'::jsonb)=pg_temp.canon((:'a2'::jsonb)->>'org',(:'a2'::jsonb)->>'uni',:'s2')
 AND pg_temp.huella(:'s2',:'a2'::jsonb)<>pg_temp.canon((:'a2'::jsonb)->>'org',NULL,:'s2'));
-- 3. UnidadRef distinta, ausente con asignación con unidad, o presente sin unidad en la asignación: denegado.
SELECT pg_temp.ok('unidad_distinta_o_ausente_denegada',
 pg_temp.huella(jsonb_set(:'s2'::jsonb,'{UnidadRef}','"unidad:otra"')::text,:'a2'::jsonb) LIKE 'denegado:%'
 AND pg_temp.huella(jsonb_set(:'s2'::jsonb,'{UnidadRef}','null')::text,:'a2'::jsonb) LIKE 'denegado:%'
 AND pg_temp.huella(jsonb_set(:'s1'::jsonb,'{UnidadRef}','"unidad:del:paso"')::text,:'a1'::jsonb) LIKE 'denegado:%');
-- 4. Vía externa con asignación con unidad: aún no admitida.
SELECT pg_temp.ok('externa_con_unidad_denegada',pg_temp.huella(jsonb_set(:'s2'::jsonb,'{Via}','"portafirmas_registro_rrhh"')::text,:'a2'::jsonb) LIKE 'denegado:%');
-- 5. Organización del material distinta de la de la asignación: denegado.
SELECT pg_temp.ok('otra_organizacion_denegada',pg_temp.huella(jsonb_set(:'s1'::jsonb,'{OrganizacionRef}','"org_otra_sintetica"')::text,:'a1'::jsonb) LIKE 'denegado:%');
-- 6. La decisión nombra una asignación con otra huella, otro perfil o otro principal: denegado.
SELECT pg_temp.ok('asignacion_alterada_denegada',
 pg_temp.huella(:'s1',jsonb_set(:'a1'::jsonb,'{asignacion_huella_sha256}',to_jsonb(repeat('0',64)))) LIKE 'denegado:%'
 AND pg_temp.huella(:'s1',jsonb_set(:'a1'::jsonb,'{perfil_activo_ref}','"prf_ajeno"')) LIKE 'denegado:%'
 AND pg_temp.huella(:'s1',jsonb_set(:'a1'::jsonb,'{principal_id}','"per_ajeno"')) LIKE 'denegado:%');
-- 7. Las v3 usan AD210 y CC10; el ejecutor sólo llega a las v3; las v2 y las
-- funciones de AD210/CC10 quedan cerradas para él.
SELECT pg_temp.ok('v3_con_ad210_cc10_y_v2_cerradas',
 strpos(pg_get_functiondef('vec_contratacion_temporal.consultar_firmas_r5_atestadas_v3(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'huella_recurso_consulta_firmas_r5_ct_v1(p_solicitud,p_decision)')>0
 AND strpos(pg_get_functiondef('vec_contratacion_temporal.recuperar_firmas_r5_atestadas_v3(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'huella_recurso_consulta_firmas_r5_ct_v1(p_solicitud,p_decision)')>0
 AND strpos(pg_get_functiondef('vec_contratacion_temporal.consultar_firmas_r5_atestadas_v3(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'paso_plan_firma_con_unidad_v1(')>0
 AND strpos(pg_get_functiondef('vec_contratacion_temporal.recuperar_firmas_r5_atestadas_v3(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'paso_plan_firma_con_unidad_v1(')>0
 AND has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.consultar_firmas_r5_atestadas_v3(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.recuperar_firmas_r5_atestadas_v3(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND NOT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.consultar_firmas_r5_atestadas_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND NOT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.recuperar_firmas_r5_atestadas_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND NOT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_autorizacion_atestada_v3.huella_recurso_consulta_firmas_r5_ct_v1(text,bytea)','EXECUTE')
 AND NOT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_catalogos_configurables.paso_plan_firma_con_unidad_v1(text,text,integer,text,text)','EXECUTE'));
-- 9. Unidad no canónica en la asignación (con comilla): se deniega y no entra
-- en la cadena de la huella. Se siembra sólo dentro de este ROLLBACK.
SET LOCAL session_replication_role=replica;
UPDATE vec_autorizacion.asignacion_perfil SET documento=jsonb_set(documento,'{ambitos}',(SELECT jsonb_agg(CASE WHEN e->>'clave'='unidad_ref'
  THEN jsonb_build_object('clave','unidad_ref','valores',jsonb_build_array('unidad"x')) ELSE e END) FROM jsonb_array_elements(documento->'ambitos') e))
 WHERE asignacion_ref=(:'a2'::jsonb)->>'asignacion_ref';
SET LOCAL session_replication_role=origin;
SELECT pg_temp.ok('unidad_no_canonica_denegada',pg_temp.huella(jsonb_set(:'s2'::jsonb,'{UnidadRef}','"unidad\"x"')::text,:'a2'::jsonb) LIKE 'denegado:%');
-- 8. Forma de UnidadRef en las v3, como el LOGIN del ejecutor: texto de
-- referencia y sólo vía VEC (22023 antes de mirar la capacidad); una válida
-- pasa esa guarda y se para después, en la decisión (42501).
CREATE ROLE prueba_ct186_ct LOGIN;
GRANT vec_contratacion_temporal_ejecutor TO prueba_ct186_ct WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
CREATE FUNCTION pg_temp.v3(funcion text,unidad jsonb,via text DEFAULT 'certificado_vec') RETURNS text LANGUAGE plpgsql AS $f$
DECLARE sol text:=jsonb_build_object('OrganizacionRef','org_ct186','ExpedienteRef','exp:ct186','VersionExpediente',1,
  'Documento','informe_definitivo','FirmantePrincipalCandidatoRef','per_ct186','ClaveIdempotencia','clave-ct186-0000001',
  'PasoOrden',1,'CatalogoHuella',repeat('a',64),'Via',via,'UnidadRef',unidad)::text;
BEGIN
 EXECUTE format('SELECT vec_contratacion_temporal.%s(%L,%L,%L,NULL,NULL,1,1,NULL,NULL,NULL,NULL)',funcion,sol,'\x7b7d'::bytea,'\x7b7d'::bytea);
 RETURN 'ok';
EXCEPTION WHEN OTHERS THEN RETURN SQLSTATE;
END $f$;
GRANT EXECUTE ON FUNCTION pg_temp.v3(text,jsonb,text) TO prueba_ct186_ct;
SET SESSION AUTHORIZATION prueba_ct186_ct;
SELECT pg_temp.ok('forma_unidad_en_las_v3',
 pg_temp.v3('consultar_firmas_r5_atestadas_v3','123')='22023' AND pg_temp.v3('recuperar_firmas_r5_atestadas_v3','123')='22023'
 AND pg_temp.v3('consultar_firmas_r5_atestadas_v3','"unidad con espacios"')='22023'
 AND pg_temp.v3('consultar_firmas_r5_atestadas_v3','"unidad:x"','portafirmas_registro_rrhh')='22023'
 AND pg_temp.v3('recuperar_firmas_r5_atestadas_v3','"unidad:x"','portafirmas_registro_rrhh')='22023'
 AND pg_temp.v3('consultar_firmas_r5_atestadas_v3','"unidad:x"')='42501' AND pg_temp.v3('recuperar_firmas_r5_atestadas_v3','"unidad:x"')='42501'
 AND pg_temp.v3('consultar_firmas_r5_atestadas_v3','null')='42501');
RESET SESSION AUTHORIZATION;
ROLLBACK;
