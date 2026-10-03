\set ON_ERROR_STOP on
-- AUT31: lectura y revalidación de cargo nominal publicado por AD160.
-- AUT30 permanece intacta y cerrada para sus consumidores históricos.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion:migracion:000031',0));
DO $preimagen$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
  OR pg_catalog.to_regprocedure('vec_autorizacion.revalidar_competencia_firmante_ct_v1(text)') IS NULL
  OR pg_catalog.to_regclass('vec_autorizacion.cargo_ct_plan') IS NULL
  OR pg_catalog.to_regclass('vec_autorizacion.cargo_ct_aprobacion') IS NULL
  OR pg_catalog.to_regclass('vec_autorizacion.cargo_ct_recibo') IS NULL
  OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_destino_cargo_ct_v1(text,text,text)') IS NULL
  OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.revalidar_vinculo_certificado_firmante_ct_v1(text,numeric,text,text,text,text)') IS NULL
  OR pg_catalog.to_regprocedure('vec_autorizacion.competencia_cargo_ct_interna_v1(text,text,text,text)') IS NOT NULL
  OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p
   WHERE p.oid=pg_catalog.to_regprocedure('vec_contexto_actor_v1.revalidar_vinculo_certificado_firmante_ct_v1(text,numeric,text,text,text,text)')
    AND p.proowner='vec_contexto_actor_v1_propietario'::regrole AND p.prosecdef AND p.provolatile='v'
    AND p.prorettype='boolean'::regtype
    AND EXISTS(SELECT 1 FROM pg_catalog.unnest(p.proconfig) c WHERE c LIKE 'search_path=pg_catalog%'))
 THEN RAISE EXCEPTION 'AUT31: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $preimagen$;
SET LOCAL ROLE vec_autorizacion_propietario;

-- Solo las fachadas nominales de este propietario llaman a esta función.
-- La unidad procede de la asignación central, nunca de un cargo textual.
CREATE FUNCTION vec_autorizacion.competencia_cargo_ct_interna_v1(
 p_persona text,p_perfil text,p_rol text,p_organizacion text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog SET row_security=on SET timezone='UTC' AS $f$
DECLARE a record; r record; t record; plan jsonb; unidad text; ahora timestamptz(6); destino text;
BEGIN
 IF current_user<>'vec_autorizacion_propietario'
  OR current_setting('transaction_isolation')<>'serializable'
  OR current_setting('transaction_read_only')<>'off'
  OR p_rol NOT IN ('ct_cargo_tecnico_solicitante','ct_cargo_delegacion_solicitante',
   'ct_cargo_direccion_rrhh','ct_cargo_jefatura_servicio_rrhh','ct_cargo_diputacion_delegada_rrhh')
  OR p_rol IS NULL THEN RETURN NULL; END IF;
 SELECT x.asignacion_ref,x.acto_ref,asignacion.version,asignacion.principal_id,asignacion.perfil_activo_ref,
  asignacion.version_rol_ref,asignacion.huella_sha256,asignacion.documento
 INTO a FROM vec_autorizacion.asignacion_perfil_actual x
 JOIN vec_autorizacion.asignacion_perfil asignacion USING(perfil_activo_ref,asignacion_ref)
 WHERE x.perfil_activo_ref=p_perfil FOR UPDATE OF x;
 IF NOT FOUND OR a.principal_id IS DISTINCT FROM p_persona
  OR a.documento->>'estado' IS DISTINCT FROM 'activa'
  OR a.acto_ref !~ '^cargo_ct:[0-9a-f]{32}$'
  OR pg_catalog.jsonb_array_length(a.documento->'ambitos')<>2 THEN RETURN NULL; END IF;
 SELECT b.valor->'valores'->>0 INTO STRICT unidad
 FROM pg_catalog.jsonb_array_elements(a.documento->'ambitos') b(valor)
 WHERE b.valor->>'clave'='unidad_ref' AND pg_catalog.jsonb_array_length(b.valor->'valores')=1;
 IF a.documento->'ambitos' IS DISTINCT FROM pg_catalog.jsonb_build_array(
   pg_catalog.jsonb_build_object('clave','organizacion_ref','valores',pg_catalog.jsonb_build_array(p_organizacion)),
   pg_catalog.jsonb_build_object('clave','unidad_ref','valores',pg_catalog.jsonb_build_array(unidad)))
 THEN RETURN NULL; END IF;
 SELECT rol.rol_id,rol.huella_sha256,rol.documento,c.revision,c.estado,c.huella_sha256 control_huella
 INTO r FROM vec_autorizacion.version_rol rol
 JOIN vec_autorizacion.control_vigencia_version_rol_actual x USING(version_rol_ref)
 JOIN vec_autorizacion.control_vigencia_version_rol c USING(version_rol_ref,revision)
 WHERE rol.version_rol_ref=a.version_rol_ref FOR UPDATE OF x;
 IF NOT FOUND OR r.rol_id IS DISTINCT FROM p_rol OR r.estado IS DISTINCT FROM 'habilitada'
  OR r.documento->>'estado' IS DISTINCT FROM 'publicada' THEN RETURN NULL; END IF;
 SELECT p.plan_canonico,p.plan_sha256,b.aprobador_ref,b.plan_sha256 aprobacion_huella,
  c.plan_sha256 recibo_huella,c.asignacion_ref
 INTO t FROM vec_autorizacion.cargo_ct_plan p
 JOIN vec_autorizacion.cargo_ct_aprobacion b USING(clave)
 JOIN vec_autorizacion.cargo_ct_recibo c USING(clave)
 WHERE p.clave=pg_catalog.substr(a.acto_ref,10);
 IF NOT FOUND OR t.asignacion_ref IS DISTINCT FROM a.asignacion_ref
  OR t.plan_sha256 IS DISTINCT FROM t.aprobacion_huella OR t.plan_sha256 IS DISTINCT FROM t.recibo_huella
  OR t.plan_sha256 IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(t.plan_canonico),'hex')
 THEN RETURN NULL; END IF;
 plan:=pg_catalog.convert_from(t.plan_canonico,'UTF8')::jsonb;
 IF plan->>'rol_id' IS DISTINCT FROM r.rol_id
  OR plan->>'version_rol_ref' IS DISTINCT FROM a.version_rol_ref
  OR plan->>'version_rol_sha256' IS DISTINCT FROM r.huella_sha256
  OR plan->>'control_rol_sha256' IS DISTINCT FROM r.control_huella
  OR plan->>'organizacion_ref' IS DISTINCT FROM p_organizacion OR plan->>'unidad_ref' IS DISTINCT FROM unidad
  OR a.huella_sha256 IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(
    pg_catalog.decode(plan->>'asignacion_canonica','base64')),'hex')
  OR pg_catalog.convert_from(pg_catalog.decode(plan->>'asignacion_canonica','base64'),'UTF8')::jsonb
    IS DISTINCT FROM a.documento
 THEN RETURN NULL; END IF;
 destino:=vec_contexto_actor_v1.acreditar_destino_cargo_ct_v1(plan->>'registro_destino_ref',p_persona,p_perfil);
 IF destino IS NULL OR destino IS DISTINCT FROM plan->>'destino_sha256' THEN RETURN NULL; END IF;
 ahora:=pg_catalog.clock_timestamp();
 IF ahora<(a.documento->>'vigente_desde')::timestamptz OR ahora>=(a.documento->>'vigente_hasta')::timestamptz
 THEN RETURN NULL; END IF;
 RETURN pg_catalog.jsonb_build_object(
  'FirmantePrincipalRef',p_persona,'RolIDFirmante',r.rol_id,'CargoFirmante',r.rol_id,'UnidadFirmanteRef',unidad,
  'PerfilActivoFirmanteRef',a.perfil_activo_ref,'AsignacionFirmanteRef',a.asignacion_ref,
  'AsignacionFirmanteVersion',a.version,'AsignacionFirmanteHuella',a.huella_sha256,
  'VersionRolFirmanteRef',a.version_rol_ref,'VersionRolFirmanteHuella',r.huella_sha256,
  'ControlVigenciaFirmanteRef',a.version_rol_ref,'ControlVigenciaFirmanteRevision',r.revision,
  'ControlVigenciaFirmanteHuella',r.control_huella,'AsignacionVigenteDesde',a.documento->>'vigente_desde',
  'AsignacionVigenteHasta',a.documento->>'vigente_hasta','ActoCompetenciaRef',a.acto_ref,
  'RegistroContextoFirmanteRef',plan->>'registro_destino_ref','ContextoFirmanteHuella',destino,
  'CompetenciaComprobadaEn',pg_catalog.regexp_replace(pg_catalog.to_char(ahora AT TIME ZONE 'UTC',
   'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'[.]?0+Z$','Z'));
EXCEPTION WHEN no_data_found OR too_many_rows OR data_exception THEN RETURN NULL;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.competencia_cargo_ct_interna_v1(text,text,text,text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.leer_competencia_cargo_ct_v1(p_persona text,p_rol text,p_organizacion text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE perfil text;
BEGIN
 IF current_user<>'vec_autorizacion_propietario' OR session_user=current_user
  OR current_setting('role')<>'none'
  OR current_setting('transaction_isolation')<>'serializable'
  OR current_setting('transaction_read_only')<>'off'
  OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=session_user AND rolcanlogin AND rolinherit
   AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
  OR (SELECT count(*) FROM pg_catalog.pg_auth_members WHERE member=session_user::regrole)<>1
  OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=session_user::regrole
   AND roleid='vec_autorizacion_fuente'::regrole AND inherit_option AND NOT set_option AND NOT admin_option)
 THEN RAISE EXCEPTION 'AUT31: fuente incompatible' USING ERRCODE='42501'; END IF;
 -- Impide que una publicación concurrente cree otra asignación que cambie
 -- la unicidad observada. Coopera con los publicadores centrales existentes.
 LOCK TABLE vec_autorizacion.asignacion_perfil_actual IN SHARE MODE;
 SELECT a.perfil_activo_ref INTO STRICT perfil
 FROM vec_autorizacion.asignacion_perfil_actual x
 JOIN vec_autorizacion.asignacion_perfil a USING(perfil_activo_ref,asignacion_ref)
 JOIN vec_autorizacion.version_rol r USING(version_rol_ref)
 WHERE a.principal_id=p_persona AND r.rol_id=p_rol AND a.documento->>'estado'='activa'
  AND a.documento->'ambitos' @> pg_catalog.jsonb_build_array(pg_catalog.jsonb_build_object(
   'clave','organizacion_ref','valores',pg_catalog.jsonb_build_array(p_organizacion)))
  AND pg_catalog.clock_timestamp()>=(a.documento->>'vigente_desde')::timestamptz
  AND pg_catalog.clock_timestamp()<(a.documento->>'vigente_hasta')::timestamptz;
 RETURN vec_autorizacion.competencia_cargo_ct_interna_v1(p_persona,perfil,p_rol,p_organizacion);
EXCEPTION WHEN no_data_found OR too_many_rows OR data_exception THEN RETURN NULL;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.leer_competencia_cargo_ct_v1(text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.leer_competencia_cargo_ct_v1(text,text,text) TO vec_autorizacion_fuente;

-- CT conserva la autoridad del circuito. Su emisor V3 ha ligado catálogo,
-- paso y perfil lógico al RolID; aquí se revalida únicamente la competencia
-- central. ActorRef se añade desde la decisión consumida, nunca desde HTTP.
CREATE FUNCTION vec_autorizacion.revalidar_competencia_firmante_ct_v2(p_solicitud text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE s jsonb; e jsonb; clave text;
BEGIN
 IF current_user<>'vec_autorizacion_propietario' OR session_user=current_user
  OR current_setting('role')<>'none'
  OR NOT pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
  OR pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
  OR current_setting('transaction_isolation')<>'serializable'
  OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'AUT31: consumidor incompatible' USING ERRCODE='42501'; END IF;
 IF p_solicitud IS NULL OR pg_catalog.octet_length(p_solicitud) NOT BETWEEN 2 AND 65536 THEN RETURN false; END IF;
 s:=p_solicitud::jsonb;
 IF pg_catalog.jsonb_typeof(s) IS DISTINCT FROM 'object'
  OR (SELECT count(*) FROM pg_catalog.json_each(p_solicitud::json))<>(SELECT count(*) FROM pg_catalog.jsonb_each(s))
 THEN RETURN false; END IF;
 FOREACH clave IN ARRAY ARRAY['ActorRef','FirmantePrincipalRef','RolIDFirmante','CargoFirmante','PerfilFirmanteRef',
  'PerfilActivoFirmanteRef','OrganizacionRef','UnidadFirmanteRef','ActoCompetenciaRef',
  'CuentaFirmanteRef','VinculoCredencialFirmanteRef','VinculoCredencialFirmanteHuella',
  'AsignacionFirmanteRef','AsignacionFirmanteHuella','VersionRolFirmanteRef','VersionRolFirmanteHuella',
  'ControlVigenciaFirmanteRef','ControlVigenciaFirmanteHuella','AsignacionVigenteDesde','AsignacionVigenteHasta',
  'CatalogoRef','CatalogoHuella','PasoRef','CertificadoHuella','FirmanteRef','Via'] LOOP
  IF pg_catalog.jsonb_typeof(s->clave) IS DISTINCT FROM 'string'
   OR pg_catalog.octet_length(s->>clave) NOT BETWEEN 1 AND 512 THEN RETURN false; END IF;
 END LOOP;
 FOREACH clave IN ARRAY ARRAY['AsignacionFirmanteVersion','ControlVigenciaFirmanteRevision','VinculoCredencialFirmanteRevision','CatalogoVersion','PasoOrden'] LOOP
  IF pg_catalog.jsonb_typeof(s->clave) IS DISTINCT FROM 'number'
   OR (s->>clave)!~'^[1-9][0-9]{0,15}$' THEN RETURN false; END IF;
 END LOOP;
 FOREACH clave IN ARRAY ARRAY['AsignacionFirmanteHuella','VersionRolFirmanteHuella','ControlVigenciaFirmanteHuella',
  'CatalogoHuella','CertificadoHuella','VinculoCredencialFirmanteHuella'] LOOP
  IF (s->>clave)!~'^[0-9a-f]{64}$' OR s->>clave=pg_catalog.repeat('0',64) THEN RETURN false; END IF;
 END LOOP;
 IF s->>'FirmanteRef' IS DISTINCT FROM 'ref:'||(s->>'CertificadoHuella')
  OR s->>'Via' NOT IN ('certificado_vec','portafirmas_registro_rrhh')
  OR s->>'PuestoFirmanteRef' IS NOT NULL OR s->>'AmbitoFirmanteRef' IS NOT NULL
  OR s->>'DelegacionRef' IS NOT NULL
  OR (s->>'Via'='certificado_vec' AND s->>'ActorRef' IS DISTINCT FROM s->>'FirmantePrincipalRef')
  OR (s->>'Via'='portafirmas_registro_rrhh' AND s->>'ActorRef'=s->>'FirmantePrincipalRef')
 THEN RETURN false; END IF;
 IF vec_contexto_actor_v1.revalidar_vinculo_certificado_firmante_ct_v1(
  s->>'VinculoCredencialFirmanteRef',(s->>'VinculoCredencialFirmanteRevision')::numeric,
  s->>'VinculoCredencialFirmanteHuella',s->>'CertificadoHuella',
  s->>'FirmantePrincipalRef',s->>'CuentaFirmanteRef') IS NOT TRUE THEN RETURN false; END IF;
 e:=vec_autorizacion.competencia_cargo_ct_interna_v1(s->>'FirmantePrincipalRef',s->>'PerfilActivoFirmanteRef',
  s->>'RolIDFirmante',s->>'OrganizacionRef');
 IF e IS NULL THEN RETURN false; END IF;
 e:=e-ARRAY['RegistroContextoFirmanteRef','ContextoFirmanteHuella','CompetenciaComprobadaEn'];
 RETURN s @> e;
EXCEPTION WHEN data_exception THEN RETURN false;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.revalidar_competencia_firmante_ct_v2(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.revalidar_competencia_firmante_ct_v2(text) TO vec_contratacion_temporal_propietario;
COMMIT;
