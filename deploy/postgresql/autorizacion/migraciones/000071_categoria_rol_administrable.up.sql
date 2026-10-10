\set ON_ERROR_STOP on
-- AUT71: resolver_rol_administrable_v1 publica también categoria_admin.
-- El adaptador Go (administracionperfiles/postgres, rolJSON) exige esa clave
-- para un rol de clase administrador y la quiere nula para el resto; AUT24 no
-- la devolvía y toda propuesta B1 o de rol nuevo terminaba en «no disponible».
-- La categoría sale de la fuente nominal que ya existe (AUT33/AUT36:
-- perfil_fijo_categoria_nominal_v1), ligada a la versión y huella exactas del
-- rol; no se infiere de la clase ni del nombre. Un administrador sin esa fila
-- falla cerrado, igual que el Go (domain.RolAdministrable.ValidarEn).
-- Sustituye sólo el cuerpo: misma firma, OID, propietario, ACL y search_path.
-- Una sola vez; reaplicarla se rechaza por la guarda de preimagen. Sin DOWN.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$
DECLARE f record;numero integer;propietario oid:=pg_catalog.to_regrole('vec_autorizacion_propietario');
 permitidos text[]:=ARRAY['vec_autorizacion_propietario','vec_admin_perfiles_lote_ejecutor',
  'vec_admin_gobierno_roles_ejecutor','vec_admin_version_rol_bolsa_ejecutor'];
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 OR pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 THEN RAISE EXCEPTION 'AUT71: PARO clave=postgres18_superuser actual=no esperado=si' USING ERRCODE='55000';END IF;
 IF propietario IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion.rol_administrable_exacto_v1') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion.perfil_fijo_categoria_nominal_v1') IS NULL
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_constraint WHERE conrelid='vec_autorizacion.perfil_fijo_categoria_nominal_v1'::regclass
  AND contype='c' AND pg_catalog.pg_get_constraintdef(oid)='CHECK ((categoria_administrativa = ANY (ARRAY[''aplicacion''::text, ''sistemas''::text])))')
 OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
  WHERE n.nspname='vec_autorizacion' AND p.proname='resolver_rol_administrable_v1')<>1
 OR EXISTS(SELECT 1 FROM pg_catalog.unnest(permitidos) r(nombre) WHERE pg_catalog.to_regrole(r.nombre) IS NULL) THEN
  RAISE EXCEPTION 'AUT71: PARO clave=dependencias actual=divergente esperado=AUT24+AUT36+AUT44+AUT60+AUT63' USING ERRCODE='55000';END IF;
 -- Preimagen exacta de AUT24 tal como queda instalada tras AUT44, AUT60 y AUT63.
 SELECT p.*,pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(p.oid),'UTF8')),'hex') AS def_sha
  INTO f FROM pg_catalog.pg_proc p WHERE p.oid=pg_catalog.to_regprocedure('vec_autorizacion.resolver_rol_administrable_v1(text)');
 IF NOT FOUND OR f.def_sha IS DISTINCT FROM '43ead3289904852fb77ed4ed67e143a6b1c4e98ac136fddd2c06734e2ae41d0c'
 OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(f.prosrc,'UTF8')),'hex') IS DISTINCT FROM '68f57edb50bd593942b90abc51c84147fd725e3e856152e7c8f98d025a5e3748'
 OR f.proowner IS DISTINCT FROM propietario OR NOT f.prosecdef OR f.provolatile<>'v'
 OR f.prorettype<>'jsonb'::regtype OR f.proretset OR f.proargnames IS DISTINCT FROM ARRAY['p_version_ref']::text[]
 OR f.prolang<>(SELECT oid FROM pg_catalog.pg_language WHERE lanname='plpgsql')
 OR f.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp']::text[] THEN
  RAISE EXCEPTION 'AUT71: PARO clave=preimagen_resolver actual=% esperado=43ead328',coalesce(f.def_sha,'ausente') USING ERRCODE='55000';END IF;
 SELECT pg_catalog.count(*) INTO numero FROM pg_catalog.aclexplode(f.proacl);
 IF f.proacl IS NULL OR numero<>pg_catalog.cardinality(permitidos)
 OR EXISTS(SELECT 1 FROM pg_catalog.aclexplode(f.proacl) a LEFT JOIN pg_catalog.pg_roles r ON r.oid=a.grantee
  WHERE coalesce(r.rolname,'PUBLIC')<>ALL(permitidos) OR a.grantor<>propietario OR a.privilege_type<>'EXECUTE' OR a.is_grantable)
 OR (SELECT pg_catalog.count(DISTINCT a.grantee) FROM pg_catalog.aclexplode(f.proacl) a)<>pg_catalog.cardinality(permitidos) THEN
  RAISE EXCEPTION 'AUT71: PARO clave=acl_resolver actual=% esperado=%',numero,pg_catalog.cardinality(permitidos) USING ERRCODE='55000';END IF;
 -- Ningún administrador del catálogo puede quedar sin categoría: el cambio no
 -- debe cerrar un rol que hoy funciona en los consumidores SQL de AUT24/AUT44.
 IF EXISTS(SELECT 1 FROM vec_autorizacion.rol_administrable_exacto_v1 a
  LEFT JOIN vec_autorizacion.perfil_fijo_categoria_nominal_v1 m ON m.version_rol_ref=a.version_rol_ref
  WHERE (a.clase='administrador' AND (m.version_rol_ref IS NULL OR m.version_rol_huella_sha256<>a.huella_sha256
    OR m.tipo_perfil<>'fijo_sistema' OR m.categoria_administrativa NOT IN('aplicacion','sistemas')))
  OR (a.clase<>'administrador' AND m.version_rol_ref IS NOT NULL)) THEN
  RAISE EXCEPTION 'AUT71: PARO clave=categoria_catalogo actual=incoherente esperado=administrador_con_categoria' USING ERRCODE='55000';END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;

-- Contrato del adaptador Go (rolJSON.CategoriaAdmin *string):
--  - clase administrador: 'aplicacion' o 'sistemas' de la fila nominal de esa
--    versión y huella; sin fila coherente, 42501 (fallo cerrado);
--  - cualquier otra clase: null, y una fila nominal para ella es incoherente.
CREATE OR REPLACE FUNCTION vec_autorizacion.resolver_rol_administrable_v1(p_version_ref text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE r record; meta record; con_meta boolean; categoria text:=null; instante timestamptz:=clock_timestamp();
BEGIN
 SELECT a.* INTO r FROM vec_autorizacion.rol_administrable_exacto_v1 a
 JOIN vec_autorizacion.version_rol v USING(version_rol_ref)
 JOIN vec_autorizacion.control_vigencia_version_rol_actual ca USING(version_rol_ref)
 JOIN vec_autorizacion.control_vigencia_version_rol c ON c.version_rol_ref=ca.version_rol_ref AND c.revision=ca.revision
 WHERE a.version_rol_ref=p_version_ref AND a.huella_sha256=v.huella_sha256
 AND v.documento->>'estado'='publicada' AND c.estado='habilitada'
 AND instante>=a.vigente_desde AND instante<a.vigente_hasta;
 IF NOT FOUND THEN RAISE EXCEPTION 'AUT24: rol no administrable' USING ERRCODE='42501'; END IF;
 SELECT m.* INTO meta FROM vec_autorizacion.perfil_fijo_categoria_nominal_v1 m WHERE m.version_rol_ref=r.version_rol_ref;
 con_meta:=FOUND;
 IF r.clase='administrador' THEN
  IF NOT con_meta OR meta.version_rol_huella_sha256 IS DISTINCT FROM r.huella_sha256
  OR meta.tipo_perfil IS DISTINCT FROM 'fijo_sistema' OR meta.categoria_administrativa IS NULL
  OR meta.categoria_administrativa NOT IN ('aplicacion','sistemas') THEN
   RAISE EXCEPTION 'AUT71: administrador sin categoria nominal' USING ERRCODE='42501'; END IF;
  categoria:=meta.categoria_administrativa;
 ELSIF con_meta THEN
  RAISE EXCEPTION 'AUT71: categoria nominal en rol no administrador' USING ERRCODE='42501';
 END IF;
 RETURN jsonb_build_object('version_ref',r.version_rol_ref,'clase',r.clase,'huella_sha256',r.huella_sha256,
  'vigente_desde',r.vigente_desde,'vigente_hasta',r.vigente_hasta,'unidad_requerida',r.unidad_requerida,
  'categoria_admin',categoria);
END $f$;

RESET ROLE;
DO $post$
DECLARE f record;numero integer;resueltos integer:=0;fila record;resultado jsonb;propietario oid:=pg_catalog.to_regrole('vec_autorizacion_propietario');
 permitidos text[]:=ARRAY['vec_autorizacion_propietario','vec_admin_perfiles_lote_ejecutor',
  'vec_admin_gobierno_roles_ejecutor','vec_admin_version_rol_bolsa_ejecutor'];
BEGIN
 SELECT p.*,pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(p.oid),'UTF8')),'hex') AS def_sha
  INTO f FROM pg_catalog.pg_proc p WHERE p.oid=pg_catalog.to_regprocedure('vec_autorizacion.resolver_rol_administrable_v1(text)');
 IF NOT FOUND OR f.def_sha IS DISTINCT FROM '9f2ca6af30c7040eee2b3905ce1ca667261254c8ba110bdb9a0703c638c0af39'
 OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(f.prosrc,'UTF8')),'hex') IS DISTINCT FROM '2bff713c504b8b0b29d719ede504e3dda6676fd477a081afcd8ee460650a12e7'
 OR f.proowner IS DISTINCT FROM propietario OR NOT f.prosecdef OR f.provolatile<>'v'
 OR f.prorettype<>'jsonb'::regtype OR f.proretset
 OR f.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp']::text[] THEN
  RAISE EXCEPTION 'AUT71: PARO clave=postimagen_resolver actual=% esperado=9f2ca6af',coalesce(f.def_sha,'ausente') USING ERRCODE='55000';END IF;
 SELECT pg_catalog.count(*) INTO numero FROM pg_catalog.aclexplode(f.proacl);
 IF f.proacl IS NULL OR numero<>pg_catalog.cardinality(permitidos)
 OR EXISTS(SELECT 1 FROM pg_catalog.aclexplode(f.proacl) a LEFT JOIN pg_catalog.pg_roles r ON r.oid=a.grantee
  WHERE coalesce(r.rolname,'PUBLIC')<>ALL(permitidos) OR a.grantor<>propietario OR a.privilege_type<>'EXECUTE' OR a.is_grantable)
 OR (SELECT pg_catalog.count(DISTINCT a.grantee) FROM pg_catalog.aclexplode(f.proacl) a)<>pg_catalog.cardinality(permitidos) THEN
  RAISE EXCEPTION 'AUT71: PARO clave=acl_postimagen actual=% esperado=%',numero,pg_catalog.cardinality(permitidos) USING ERRCODE='55000';END IF;
 -- Cada administrador resoluble por AUT24 devuelve la categoría de su fila
 -- nominal; sin capturar excepciones, para que un fallo cerrado no se oculte.
 FOR fila IN SELECT a.version_rol_ref,m.categoria_administrativa FROM vec_autorizacion.rol_administrable_exacto_v1 a
  JOIN vec_autorizacion.version_rol v ON v.version_rol_ref=a.version_rol_ref AND v.huella_sha256=a.huella_sha256
  JOIN vec_autorizacion.control_vigencia_version_rol_actual ca ON ca.version_rol_ref=a.version_rol_ref
  JOIN vec_autorizacion.control_vigencia_version_rol c ON c.version_rol_ref=ca.version_rol_ref AND c.revision=ca.revision
  JOIN vec_autorizacion.perfil_fijo_categoria_nominal_v1 m ON m.version_rol_ref=a.version_rol_ref
  WHERE a.clase='administrador' AND v.documento->>'estado'='publicada' AND c.estado='habilitada'
  AND pg_catalog.clock_timestamp()>=a.vigente_desde AND pg_catalog.clock_timestamp()+interval '1 minute'<a.vigente_hasta LOOP
  resultado:=vec_autorizacion.resolver_rol_administrable_v1(fila.version_rol_ref);
  IF resultado->>'categoria_admin' IS DISTINCT FROM fila.categoria_administrativa
  OR resultado->>'clase' IS DISTINCT FROM 'administrador' THEN
   RAISE EXCEPTION 'AUT71: PARO clave=categoria_% actual=% esperado=%',fila.version_rol_ref,resultado->>'categoria_admin',fila.categoria_administrativa USING ERRCODE='55000';END IF;
  resueltos:=resueltos+1;
 END LOOP;
 IF resueltos=0 THEN
  RAISE EXCEPTION 'AUT71: PARO clave=administradores_resueltos actual=0 esperado=>0' USING ERRCODE='55000';END IF;
END $post$;
COMMIT;
