\set ON_ERROR_STOP on
-- AUT72: registrar_fallo_version_rol_bolsa_v1 devuelve también el SQLSTATE.
-- AUT63 recoge en las fachadas de propuesta y cierre B1 el código del fallo
-- (GET STACKED DIAGNOSTICS RETURNED_SQLSTATE), pero la respuesta sólo decía
-- «denegado» o «error»: un 503 con la clase sql_intento_error no permitía saber
-- qué había fallado. Desde AUT72 la respuesta lleva la clave «sqlstate» con el
-- código de cinco caracteres [0-9A-Z] o null si no tiene ese formato. Nunca el
-- mensaje, el detalle ni el contexto: el código es una clase, no una causa.
-- El asiento de auditoría NO cambia. registrar_intento_version_rol_bolsa_v1
-- (AD227) tiene una ABI cerrada de doce claves con huella encadenada y el
-- CHECK auditoria_version_rol_bolsa_formato_v1 fija motivo_ref; guardar ahí el
-- código exigiría rehacer la familia de auditoría de AD227.
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
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 OR pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 THEN RAISE EXCEPTION 'AUT72: PARO clave=postgres18_superuser actual=no esperado=si' USING ERRCODE='55000';END IF;
 IF propietario IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_version_rol_bolsa_v1(jsonb)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.proponer_version_rol_bolsa_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.cerrar_version_rol_bolsa_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
  WHERE n.nspname='vec_autorizacion' AND p.proname='registrar_fallo_version_rol_bolsa_v1')<>1 THEN
  RAISE EXCEPTION 'AUT72: PARO clave=dependencias actual=divergente esperado=AUT63+AD227' USING ERRCODE='55000';END IF;
 -- Preimagen exacta de AUT63.
 SELECT p.*,pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(p.oid),'UTF8')),'hex') AS def_sha
  INTO f FROM pg_catalog.pg_proc p WHERE p.oid=pg_catalog.to_regprocedure('vec_autorizacion.registrar_fallo_version_rol_bolsa_v1(text,text,text)');
 IF NOT FOUND OR f.def_sha IS DISTINCT FROM 'b6e33f0eec258cc2ebbc4829f677186d20da986068bfecf67e568304b29e4e55'
 OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(f.prosrc,'UTF8')),'hex') IS DISTINCT FROM 'f90048b8aea14ff7a296919fc0a679420ef1ecbf372e9bed5013fc947574c9a6'
 OR f.proowner IS DISTINCT FROM propietario OR NOT f.prosecdef OR f.provolatile<>'v' OR f.proparallel<>'u'
 OR f.prorettype<>'jsonb'::regtype OR f.proretset
 OR f.proargnames IS DISTINCT FROM ARRAY['p_material','p_accion','p_sqlstate']::text[]
 OR f.prolang<>(SELECT oid FROM pg_catalog.pg_language WHERE lanname='plpgsql')
 OR f.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','TimeZone=UTC','row_security=on']::text[] THEN
  RAISE EXCEPTION 'AUT72: PARO clave=preimagen_registrar_fallo actual=% esperado=b6e33f0e',coalesce(f.def_sha,'ausente') USING ERRCODE='55000';END IF;
 -- AUT63 la dejó sin EXECUTE para nadie más que su propietario.
 SELECT pg_catalog.count(*) INTO numero FROM pg_catalog.aclexplode(f.proacl);
 IF f.proacl IS NULL OR numero<>1
 OR EXISTS(SELECT 1 FROM pg_catalog.aclexplode(f.proacl) a
  WHERE a.grantee<>propietario OR a.grantor<>propietario OR a.privilege_type<>'EXECUTE' OR a.is_grantable) THEN
  RAISE EXCEPTION 'AUT72: PARO clave=acl_registrar_fallo actual=% esperado=1',numero USING ERRCODE='55000';END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;

-- El subbloque de cada fachada revierte consumo y escrituras al fallar. El
-- intento se asienta después, en la misma transacción que confirma el caller.
-- No convierte un fallo de auditoría en una respuesta denegada sin recibo.
-- «sqlstate» sólo admite el formato de un código SQLSTATE; lo demás es null.
CREATE OR REPLACE FUNCTION vec_autorizacion.registrar_fallo_version_rol_bolsa_v1(
 p_material text,p_accion text,p_sqlstate text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET row_security=on AS $f$
DECLARE m jsonb;h text;corr text;estado text;codigo text;codigo_sql text;a record;
BEGIN
 h:=encode(sha256(convert_to(coalesce(p_material,''),'UTF8')),'hex');
 corr:='correlacion_'||substr(h,1,32);
 BEGIN
  m:=p_material::jsonb;
  IF jsonb_typeof(m)='object' AND m->>'correlacion_ref' ~ '^correlacion_[0-9a-f]{32}$'
  THEN corr:=m->>'correlacion_ref';END IF;
 EXCEPTION WHEN OTHERS THEN NULL;
 END;
 estado:=CASE WHEN p_sqlstate IN('42501','22023','23505','P0002') THEN 'denegado' ELSE 'error' END;
 codigo:='version_rol_bolsa_'||estado;
 codigo_sql:=CASE WHEN p_sqlstate COLLATE "C" ~ '^[0-9A-Z]{5}$' THEN p_sqlstate END;
 SELECT * INTO STRICT a FROM vec_autorizacion_atestada_v3.registrar_intento_version_rol_bolsa_v1(
  jsonb_build_object('tipo_registro','intento_version_rol_bolsa',
   'evento_ref','evento_'||replace(gen_random_uuid()::text,'-',''),
   'operador_login',session_user::text,'solicitud_sha256',h,
   'accion',p_accion,'recurso_ref','solicitud_version_rol_bolsa:'||substr(h,1,32),
   'resultado',estado,'motivo_ref',codigo,'proceso','postgresql',
   'canal','operacion_tecnica_privada','finalidad_ref','gobierno_definiciones_perfiles',
   'correlacion_ref',corr));
 RETURN jsonb_build_object('estado',estado,'codigo',codigo,'sqlstate',codigo_sql,'auditoria_intento',
  jsonb_build_object('auditoria_ref',a.auditoria_ref,'secuencia',a.secuencia,
   'huella_sha256',a.huella_sha256,'correlacion_ref',a.correlacion_ref,
   'registrada_en',a.registrada_en));
END $f$;

RESET ROLE;
DO $post$
DECLARE f record;numero integer;propietario oid:=pg_catalog.to_regrole('vec_autorizacion_propietario');
BEGIN
 SELECT p.*,pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(p.oid),'UTF8')),'hex') AS def_sha
  INTO f FROM pg_catalog.pg_proc p WHERE p.oid=pg_catalog.to_regprocedure('vec_autorizacion.registrar_fallo_version_rol_bolsa_v1(text,text,text)');
 IF NOT FOUND OR f.def_sha IS DISTINCT FROM '52c78d9a6ad88b305e1ce74feb945f3c4f99055798339d6028003dd9a0623ed1'
 OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(f.prosrc,'UTF8')),'hex') IS DISTINCT FROM '79c932633b9021001717e0c921d88dcb4288c0ced24dcabaa24c999072a01bfe'
 OR f.proowner IS DISTINCT FROM propietario OR NOT f.prosecdef OR f.provolatile<>'v' OR f.proparallel<>'u'
 OR f.prorettype<>'jsonb'::regtype OR f.proretset
 OR f.proargnames IS DISTINCT FROM ARRAY['p_material','p_accion','p_sqlstate']::text[]
 OR f.prolang<>(SELECT oid FROM pg_catalog.pg_language WHERE lanname='plpgsql')
 OR f.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','TimeZone=UTC','row_security=on']::text[] THEN
  RAISE EXCEPTION 'AUT72: PARO clave=postimagen_registrar_fallo actual=% esperado=52c78d9a',coalesce(f.def_sha,'ausente') USING ERRCODE='55000';END IF;
 SELECT pg_catalog.count(*) INTO numero FROM pg_catalog.aclexplode(f.proacl);
 IF f.proacl IS NULL OR numero<>1
 OR EXISTS(SELECT 1 FROM pg_catalog.aclexplode(f.proacl) a
  WHERE a.grantee<>propietario OR a.grantor<>propietario OR a.privilege_type<>'EXECUTE' OR a.is_grantable) THEN
  RAISE EXCEPTION 'AUT72: PARO clave=acl_postimagen actual=% esperado=1',numero USING ERRCODE='55000';END IF;
END $post$;
COMMIT;
