\set ON_ERROR_STOP on
-- AUT42: estructura de mantenimiento aprobado. No publica rol5 ni permisos al instalar.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) THEN RAISE EXCEPTION 'AUT42: PARO clave=migrador actual=no_superusuario esperado=superusuario' USING ERRCODE='42501'; END IF;
 IF current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999 THEN RAISE EXCEPTION 'AUT42: PARO clave=PG actual=% esperado=18',current_setting('server_version_num') USING ERRCODE='55000'; END IF;
 IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_mantenimiento_perfil_fijo_admin_v1(jsonb)') IS NULL
 OR to_regprocedure('vec_autorizacion.cotejar_ambitos_bootstrap_central_admin_v3(jsonb,timestamptz)') IS NULL
 OR to_regprocedure('vec_contexto_actor_v1.bloquear_contexto_admin_v1(text,text,text,text,numeric,numeric,numeric,numeric)') IS NULL
 OR to_regclass('vec_autorizacion.sello_efecto_admin_tx_v1') IS NULL
 OR to_regclass('vec_autorizacion.mantenimiento_perfil_fijo_admin_v1') IS NOT NULL
 THEN RAISE EXCEPTION 'AUT42: PARO clave=dependencias actual=divergente esperado=AUT33_34_37_AD183_sin_AUT42' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE FUNCTION vec_autorizacion.concesiones_denominacion_persona_admin_v1()
RETURNS jsonb LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT $datos$[{"accion":"vec.persona.denominacion.publicar","modulo_id":"vec","tipo_recurso":"persona_denominacion","finalidades":["presentacion_persona"],"garantia_minima":"alto","campos_permitidos":["denominacion"],"obligaciones":["auditar"]},{"accion":"vec.persona.denominacion.leer","modulo_id":"vec","tipo_recurso":"persona_denominacion","finalidades":["presentacion_persona"],"garantia_minima":"alto","campos_permitidos":["nombre_mostrar"],"obligaciones":["auditar"]}]$datos$::jsonb
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.concesiones_denominacion_persona_admin_v1() FROM PUBLIC;
DO $fuente_acreditar_perfil_aplicacion_nominal_v1$
DECLARE actual text;
BEGIN
 SELECT encode(pg_catalog.sha256(convert_to(prosrc,'UTF8')),'hex') INTO actual FROM pg_proc
 WHERE oid=to_regprocedure('vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(text,text,text,text,text,text,text,text,jsonb,jsonb)') AND proowner='vec_autorizacion_propietario'::regrole AND prosecdef;
 IF actual IS DISTINCT FROM 'd88a2c5f24fc6f359e3230a2ed71f1f84272902d1dc268f401e97f5ce4f02ba5' THEN RAISE EXCEPTION 'AUT42: PARO clave=acreditar_perfil_aplicacion_nominal_v1.prosrc actual=% esperado=d88a2c5f24fc6f359e3230a2ed71f1f84272902d1dc268f401e97f5ce4f02ba5',COALESCE(actual,'ausente') USING ERRCODE='55000'; END IF;
END $fuente_acreditar_perfil_aplicacion_nominal_v1$;
CREATE FUNCTION vec_autorizacion.acreditar_perfil_aplicacion_nominal_v4_preservada_aut42(
 version_ref text,p_asignacion_ref text,principal_ref text,perfil_ref text,
 accion text,modulo text,tipo text,finalidad text,campos jsonb,autenticacion jsonb)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE r record;ct record;asig record;meta record;ahora timestamptz;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
  OR pg_catalog.current_setting('transaction_read_only')<>'off'
  OR version_ref IS DISTINCT FROM 'rol:administracion_perfiles:v4'
  OR accion IS NULL OR modulo IS NULL OR tipo IS NULL OR finalidad IS NULL
  OR campos IS NULL OR pg_catalog.jsonb_typeof(campos)<>'array'
  OR pg_catalog.jsonb_typeof(autenticacion)<>'object'
  OR vec_identidad_sesiones_v1.acreditar_operador_cargos_ct_v1(autenticacion) IS NOT TRUE
  THEN RETURN false; END IF;
 SELECT v.* INTO r FROM vec_autorizacion.version_rol v WHERE v.version_rol_ref=version_ref FOR SHARE;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT c.* INTO ct FROM vec_autorizacion.control_vigencia_version_rol_actual a
 JOIN vec_autorizacion.control_vigencia_version_rol c USING(version_rol_ref,revision)
 WHERE a.version_rol_ref=version_ref FOR SHARE OF a,c;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.* INTO asig FROM vec_autorizacion.asignacion_perfil_actual a
 JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref)
 WHERE a.perfil_activo_ref=perfil_ref AND a.asignacion_ref=p_asignacion_ref FOR SHARE OF a,x;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.* INTO meta FROM vec_autorizacion.perfil_fijo_categoria_nominal_v1 x WHERE x.version_rol_ref=version_ref FOR SHARE;
 IF NOT FOUND THEN RETURN false; END IF;
 ahora:=pg_catalog.clock_timestamp();
 RETURN r.rol_id='administracion_perfiles' AND r.version=4 AND r.documento->>'estado'='publicada'
  AND r.huella_sha256=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion.canon_version_rol_admin_v1(r.documento),'UTF8')),'hex')
  AND ct.estado='habilitada' AND ct.huella_sha256=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion.canon_control_rol_admin_v1(ct.documento),'UTF8')),'hex')
  AND meta.version_rol_huella_sha256=r.huella_sha256 AND meta.categoria_administrativa='aplicacion' AND meta.tipo_perfil='fijo_sistema'
  AND meta.fuente_ref=version_ref AND meta.fuente_version=4 AND meta.fuente_huella_sha256=r.huella_sha256
  AND asig.version_rol_ref=version_ref AND asig.principal_id=principal_ref AND asig.perfil_activo_ref=perfil_ref
  AND asig.documento->>'estado'='activa'
  AND ahora>=(asig.documento->>'vigente_desde')::timestamptz
  AND ahora<(asig.documento->>'vigente_hasta')::timestamptz
  AND EXISTS(SELECT 1 FROM vec_autorizacion.catalogo_accion_nominal_v1 catalogo
    WHERE catalogo.version_rol_ref=version_ref AND catalogo.accion_ref='accion:'||accion
     AND catalogo.version=1 AND catalogo.fuente_ref=version_ref AND catalogo.fuente_version=4
     AND catalogo.fuente_huella_sha256=r.huella_sha256
     AND catalogo.clase_control='administrador_aplicacion'
     AND catalogo.dimensiones_ambito=CASE WHEN modulo='administracion' THEN '["organizacion_ref"]'::jsonb
       ELSE '["organizacion_ref","unidad_ref"]'::jsonb END
     AND ahora>=catalogo.vigente_desde AND (catalogo.vigente_hasta IS NULL OR ahora<catalogo.vigente_hasta)
     AND catalogo.concesion->>'accion'=accion AND catalogo.concesion->>'modulo_id'=modulo
     AND catalogo.concesion->>'tipo_recurso'=tipo
     AND catalogo.concesion->'finalidades'=pg_catalog.jsonb_build_array(finalidad)
     AND catalogo.concesion->>'garantia_minima'='alto'
     AND COALESCE(catalogo.concesion->'campos_permitidos','[]'::jsonb)=campos
     AND COALESCE(catalogo.concesion->'obligaciones','[]'::jsonb)='[]'::jsonb)
  AND EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(r.documento->'concesiones') c
   WHERE c->>'accion'=accion AND c->>'modulo_id'=modulo AND c->>'tipo_recurso'=tipo
    AND c->'finalidades'=pg_catalog.jsonb_build_array(finalidad) AND c->>'garantia_minima'='alto'
    AND COALESCE(c->'campos_permitidos','[]'::jsonb)=campos AND COALESCE(c->'obligaciones','[]'::jsonb)='[]'::jsonb);
END $f$;
CREATE FUNCTION vec_autorizacion.acreditar_perfil_aplicacion_nominal_v5_aut42(
 version_ref text,p_asignacion_ref text,principal_ref text,perfil_ref text,
 accion text,modulo text,tipo text,finalidad text,campos jsonb,autenticacion jsonb)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE r record;ct record;asig record;meta record;ahora timestamptz;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
  OR pg_catalog.current_setting('transaction_read_only')<>'off'
  OR version_ref IS DISTINCT FROM 'rol:administracion_perfiles:v5'
  OR accion IS NULL OR modulo IS NULL OR tipo IS NULL OR finalidad IS NULL
  OR campos IS NULL OR pg_catalog.jsonb_typeof(campos)<>'array'
  OR pg_catalog.jsonb_typeof(autenticacion)<>'object'
  OR vec_identidad_sesiones_v1.acreditar_operador_cargos_ct_v1(autenticacion) IS NOT TRUE
  THEN RETURN false; END IF;
 SELECT v.* INTO r FROM vec_autorizacion.version_rol v WHERE v.version_rol_ref=version_ref FOR SHARE;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT c.* INTO ct FROM vec_autorizacion.control_vigencia_version_rol_actual a
 JOIN vec_autorizacion.control_vigencia_version_rol c USING(version_rol_ref,revision)
 WHERE a.version_rol_ref=version_ref FOR SHARE OF a,c;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.* INTO asig FROM vec_autorizacion.asignacion_perfil_actual a
 JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref)
 WHERE a.perfil_activo_ref=perfil_ref AND a.asignacion_ref=p_asignacion_ref FOR SHARE OF a,x;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.* INTO meta FROM vec_autorizacion.perfil_fijo_categoria_nominal_v1 x WHERE x.version_rol_ref=version_ref FOR SHARE;
 IF NOT FOUND THEN RETURN false; END IF;
 ahora:=pg_catalog.clock_timestamp();
 RETURN r.rol_id='administracion_perfiles' AND r.version=5 AND r.documento->>'estado'='publicada'
  AND r.huella_sha256=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion.canon_version_rol_admin_v1(r.documento),'UTF8')),'hex')
  AND ct.estado='habilitada' AND ct.huella_sha256=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion.canon_control_rol_admin_v1(ct.documento),'UTF8')),'hex')
  AND meta.version_rol_huella_sha256=r.huella_sha256 AND meta.categoria_administrativa='aplicacion' AND meta.tipo_perfil='fijo_sistema'
  AND meta.fuente_ref=version_ref AND meta.fuente_version=5 AND meta.fuente_huella_sha256=r.huella_sha256
  AND asig.version_rol_ref=version_ref AND asig.principal_id=principal_ref AND asig.perfil_activo_ref=perfil_ref
  AND asig.documento->>'estado'='activa'
  AND ahora>=(asig.documento->>'vigente_desde')::timestamptz
  AND ahora<(asig.documento->>'vigente_hasta')::timestamptz
  AND EXISTS(SELECT 1 FROM vec_autorizacion.catalogo_accion_nominal_v1 catalogo
    WHERE catalogo.version_rol_ref=version_ref AND catalogo.accion_ref='accion:'||accion
     AND catalogo.version=2 AND catalogo.fuente_ref=version_ref AND catalogo.fuente_version=5
     AND catalogo.fuente_huella_sha256=r.huella_sha256
     AND catalogo.clase_control='administrador_aplicacion'
     AND catalogo.dimensiones_ambito=CASE WHEN modulo='administracion' THEN '["organizacion_ref"]'::jsonb
       ELSE '["organizacion_ref","unidad_ref"]'::jsonb END
     AND ahora>=catalogo.vigente_desde AND (catalogo.vigente_hasta IS NULL OR ahora<catalogo.vigente_hasta)
     AND catalogo.concesion->>'accion'=accion AND catalogo.concesion->>'modulo_id'=modulo
     AND catalogo.concesion->>'tipo_recurso'=tipo
     AND catalogo.concesion->'finalidades'=pg_catalog.jsonb_build_array(finalidad)
     AND catalogo.concesion->>'garantia_minima'='alto'
     AND COALESCE(catalogo.concesion->'campos_permitidos','[]'::jsonb)=campos
     AND COALESCE(catalogo.concesion->'obligaciones','[]'::jsonb)='[]'::jsonb)
  AND EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(r.documento->'concesiones') c
   WHERE c->>'accion'=accion AND c->>'modulo_id'=modulo AND c->>'tipo_recurso'=tipo
    AND c->'finalidades'=pg_catalog.jsonb_build_array(finalidad) AND c->>'garantia_minima'='alto'
    AND COALESCE(c->'campos_permitidos','[]'::jsonb)=campos AND COALESCE(c->'obligaciones','[]'::jsonb)='[]'::jsonb);
END $f$;
CREATE OR REPLACE FUNCTION vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(version_ref text,p_asignacion_ref text,principal_ref text,perfil_ref text,accion text,modulo text,tipo text,finalidad text,campos jsonb,autenticacion jsonb)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 IF version_ref='rol:administracion_perfiles:v4' THEN RETURN vec_autorizacion.acreditar_perfil_aplicacion_nominal_v4_preservada_aut42(version_ref,p_asignacion_ref,principal_ref,perfil_ref,accion,modulo,tipo,finalidad,campos,autenticacion);
 ELSIF version_ref='rol:administracion_perfiles:v5' THEN RETURN vec_autorizacion.acreditar_perfil_aplicacion_nominal_v5_aut42(version_ref,p_asignacion_ref,principal_ref,perfil_ref,accion,modulo,tipo,finalidad,campos,autenticacion); END IF;
 RETURN false;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.acreditar_perfil_aplicacion_nominal_v4_preservada_aut42(text,text,text,text,text,text,text,text,jsonb,jsonb),vec_autorizacion.acreditar_perfil_aplicacion_nominal_v5_aut42(text,text,text,text,text,text,text,text,jsonb,jsonb) FROM PUBLIC;
DO $fuente_acreditar_ambito_certificado_nominal_v1$
DECLARE actual text;
BEGIN
 SELECT encode(pg_catalog.sha256(convert_to(prosrc,'UTF8')),'hex') INTO actual FROM pg_proc
 WHERE oid=to_regprocedure('vec_autorizacion.acreditar_ambito_certificado_nominal_v1(text,text,text)') AND proowner='vec_autorizacion_propietario'::regrole AND prosecdef;
 IF actual IS DISTINCT FROM 'a84934dc423940c5e0e435ca4fca5ee430cfb8c8db02210d1868ac14d2ef2a6d' THEN RAISE EXCEPTION 'AUT42: PARO clave=acreditar_ambito_certificado_nominal_v1.prosrc actual=% esperado=a84934dc423940c5e0e435ca4fca5ee430cfb8c8db02210d1868ac14d2ef2a6d',COALESCE(actual,'ausente') USING ERRCODE='55000'; END IF;
END $fuente_acreditar_ambito_certificado_nominal_v1$;
CREATE FUNCTION vec_autorizacion.acreditar_ambito_certificado_nominal_v4_preservada_aut42(version_ref text,p_asignacion_ref text,org text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE a record;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
  OR pg_catalog.current_setting('transaction_read_only')<>'off'
  OR version_ref IS DISTINCT FROM 'rol:administracion_perfiles:v4'
  OR org IS NULL OR org !~ '^org_[a-z0-9]{16,80}$' THEN RETURN false; END IF;
 SELECT x.* INTO a FROM vec_autorizacion.asignacion_perfil_actual p
 JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref)
 WHERE p.asignacion_ref=p_asignacion_ref FOR SHARE OF p,x;
 IF NOT FOUND OR a.version_rol_ref<>version_ref OR a.documento->>'estado'<>'activa' THEN RETURN false; END IF;
 RETURN a.documento->'ambitos' @> pg_catalog.jsonb_build_array(
  pg_catalog.jsonb_build_object('clave','organizacion_ref','valores',pg_catalog.jsonb_build_array(org)));
END $f$;
CREATE FUNCTION vec_autorizacion.acreditar_ambito_certificado_nominal_v5_aut42(version_ref text,p_asignacion_ref text,org text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE a record;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
  OR pg_catalog.current_setting('transaction_read_only')<>'off'
  OR version_ref IS DISTINCT FROM 'rol:administracion_perfiles:v5'
  OR org IS NULL OR org !~ '^org_[a-z0-9]{16,80}$' THEN RETURN false; END IF;
 SELECT x.* INTO a FROM vec_autorizacion.asignacion_perfil_actual p
 JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref)
 WHERE p.asignacion_ref=p_asignacion_ref FOR SHARE OF p,x;
 IF NOT FOUND OR a.version_rol_ref<>version_ref OR a.documento->>'estado'<>'activa' THEN RETURN false; END IF;
 RETURN a.documento->'ambitos' @> pg_catalog.jsonb_build_array(
  pg_catalog.jsonb_build_object('clave','organizacion_ref','valores',pg_catalog.jsonb_build_array(org)));
END $f$;
CREATE OR REPLACE FUNCTION vec_autorizacion.acreditar_ambito_certificado_nominal_v1(version_ref text,p_asignacion_ref text,org text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 IF version_ref='rol:administracion_perfiles:v4' THEN RETURN vec_autorizacion.acreditar_ambito_certificado_nominal_v4_preservada_aut42(version_ref,p_asignacion_ref,org);
 ELSIF version_ref='rol:administracion_perfiles:v5' THEN RETURN vec_autorizacion.acreditar_ambito_certificado_nominal_v5_aut42(version_ref,p_asignacion_ref,org); END IF;
 RETURN false;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.acreditar_ambito_certificado_nominal_v4_preservada_aut42(text,text,text),vec_autorizacion.acreditar_ambito_certificado_nominal_v5_aut42(text,text,text) FROM PUBLIC;
