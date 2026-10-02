-- BORRADOR: depende de IS11 y CA23; no instalar ni reaplicar sobre historia.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_identidad_sesiones_v1:migracion:000012',0));
DO $preimagen$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 OR pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.revalidar_certificado_admin_v1(text,text,text,text,text,text,boolean,timestamptz,boolean,boolean)') IS NULL
 OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.consultar_perfil_admin_perfiles_v1(text)') IS NULL
 OR pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)') IS NULL
 OR pg_catalog.to_regrole('vec_autorizacion_atestada_v3_propietario') IS NULL
 OR pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.revalidar_sesion_admin_copias_v1(text,text,text,text,text,text,text,text)') IS NULL
 OR pg_catalog.to_regclass('vec_identidad_sesiones_v1.sesion_admin_perfiles_v1') IS NOT NULL
 THEN RAISE EXCEPTION 'IS12: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $preimagen$;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;

CREATE TABLE vec_identidad_sesiones_v1.preimagen_acl_admin_perfiles_v1(singleton boolean PRIMARY KEY CHECK(singleton),uso_ad3_previo boolean NOT NULL);
REVOKE ALL ON TABLE vec_identidad_sesiones_v1.preimagen_acl_admin_perfiles_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_identidad_sesiones_v1.preimagen_acl_admin_perfiles_v1 FROM PUBLIC;
INSERT INTO vec_identidad_sesiones_v1.preimagen_acl_admin_perfiles_v1 VALUES(true,pg_catalog.has_schema_privilege('vec_autorizacion_atestada_v3_propietario','vec_identidad_sesiones_v1','USAGE'));
CREATE TRIGGER preimagen_acl_admin_perfiles_historia BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_identidad_sesiones_v1.preimagen_acl_admin_perfiles_v1 FOR EACH STATEMENT EXECUTE FUNCTION vec_identidad_sesiones_v1.rechazar_mutacion();
CREATE TABLE vec_identidad_sesiones_v1.sesion_admin_perfiles_v1(
 autenticacion_ref text PRIMARY KEY,sesion_ref text NOT NULL UNIQUE,
 cuenta_ref text NOT NULL,cuenta_ordinaria_ref text NOT NULL,persona_ref text NOT NULL,perfil_ref text NOT NULL,
 seleccion_revision numeric(20,0) NOT NULL CHECK(seleccion_revision BETWEEN 1 AND 18446744073709551615),
 certificado_vinculo_ref text NOT NULL,certificado_vinculo_version numeric(20,0) NOT NULL,
 politica_ref text NOT NULL,politica_huella_sha256 text NOT NULL,
 entorno text NOT NULL,host text NOT NULL,audiencia text NOT NULL,
 autenticacion_verificada_en timestamptz(6) NOT NULL,revocacion_verificada_en timestamptz(6) NOT NULL,
 crl_vigente_hasta timestamptz(6) NOT NULL,certificado_vigente_hasta timestamptz(6) NOT NULL,
 observacion_vigente_hasta timestamptz(6) NOT NULL,
 vigente_hasta timestamptz(6) NOT NULL,
 snapshot jsonb NOT NULL,snapshot_sha256 text NOT NULL,
 registrada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
 FOREIGN KEY(autenticacion_ref) REFERENCES vec_identidad_sesiones_v1.consumo_asercion(autenticacion_ref),
 FOREIGN KEY(sesion_ref) REFERENCES vec_identidad_sesiones_v1.consumo_asercion(sesion_ref),
 FOREIGN KEY(certificado_vinculo_ref,certificado_vinculo_version) REFERENCES vec_identidad_sesiones_v1.vinculo_certificado_admin_v1(vinculo_ref,version),
 CHECK(cuenta_ref<>cuenta_ordinaria_ref),
 CHECK((snapshot->>'seleccion_revision')::numeric IS NOT DISTINCT FROM seleccion_revision),
 CHECK(pg_catalog.isfinite(crl_vigente_hasta) AND pg_catalog.isfinite(certificado_vigente_hasta)),
 CHECK(observacion_vigente_hasta<=LEAST(crl_vigente_hasta,certificado_vigente_hasta,vigente_hasta) AND observacion_vigente_hasta>GREATEST(autenticacion_verificada_en,revocacion_verificada_en)),
 CHECK(snapshot_sha256=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(snapshot::text,'UTF8')),'hex'))
);
REVOKE ALL ON TABLE vec_identidad_sesiones_v1.sesion_admin_perfiles_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_identidad_sesiones_v1.sesion_admin_perfiles_v1 FROM PUBLIC;
CREATE TRIGGER sesion_admin_perfiles_historia BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_identidad_sesiones_v1.sesion_admin_perfiles_v1 FOR EACH STATEMENT EXECUTE FUNCTION vec_identidad_sesiones_v1.rechazar_mutacion();

-- Solo referencias centrales y alias explícitos para el HMAC ya aprovisionado.
-- No devuelve certificado, credencial ni secreto de política.
CREATE FUNCTION vec_identidad_sesiones_v1.resolver_identidad_admin_perfiles_propietaria_v1(p_entorno text,p_host text,p_audiencia text,p_certificado text,p_ca text,p_autenticada timestamptz,p_revocada timestamptz)
RETURNS TABLE(persona_ref text,cuenta_ref text,cuenta_ordinaria_ref text,vinculo_ref text,vinculo_version numeric,politica_ref text,politica_huella_sha256 text,vigente_hasta timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE pol record;v record;c record;ahora timestamptz;
BEGIN
 IF current_setting('transaction_read_only')<>'off' OR current_setting('transaction_isolation') NOT IN('serializable','read committed')
 OR p_audiencia IS NULL OR p_audiencia !~ '^[a-z0-9][a-z0-9._:-]{3,255}$'
 OR p_autenticada IS NULL OR NOT pg_catalog.isfinite(p_autenticada)
 OR p_certificado IS NULL OR p_certificado !~ '^[0-9a-f]{64}$' OR p_certificado=pg_catalog.repeat('0',64)
 OR p_ca IS NULL OR p_ca !~ '^[0-9a-f]{64}$' OR p_ca=pg_catalog.repeat('0',64)
 THEN RETURN; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 SELECT * INTO pol FROM vec_identidad_sesiones_v1.politica_certificado_admin_v1 WHERE singleton FOR SHARE;
 ahora:=pg_catalog.clock_timestamp();
 IF NOT FOUND OR p_autenticada>ahora OR p_autenticada<pol.registrada_en OR NOT pol.activa
 OR pol.entorno IS DISTINCT FROM p_entorno OR pol.host_admin IS DISTINCT FROM p_host OR pol.ca_sha256 IS DISTINCT FROM p_ca OR ahora>=pol.vigente_hasta THEN RETURN; END IF;
 SELECT x.* INTO STRICT v FROM vec_identidad_sesiones_v1.vinculo_certificado_admin_actual_v1 a JOIN vec_identidad_sesiones_v1.vinculo_certificado_admin_v1 x USING(vinculo_ref,version)
 WHERE x.certificado_sha256=p_certificado AND x.ca_sha256=p_ca AND x.politica_ref=pol.politica_ref FOR SHARE OF a;
 -- Este corte usa únicamente certificado directo: producción sigue cerrada
 -- hasta disponer de las evidencias corporativas exigidas por IS9.
 IF vec_identidad_sesiones_v1.revalidar_certificado_admin_v1(p_entorno,p_host,v.persona_ref,v.cuenta_privilegiada_ref,p_certificado,p_ca,true,p_revocada,false,false) IS NOT TRUE
 OR p_autenticada<v.vigente_desde THEN RETURN; END IF;
 SELECT * INTO STRICT c FROM vec_identidad_sesiones_v1.cuenta WHERE cuenta.cuenta_ref=v.cuenta_privilegiada_ref;
 IF c.cuenta_ordinaria_ref=c.cuenta_ref OR NOT c.cuenta_privilegiada
 OR NOT EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.cuenta o WHERE o.cuenta_ref=c.cuenta_ordinaria_ref AND NOT o.cuenta_privilegiada AND o.cuenta_ordinaria_ref IS NULL) THEN RETURN; END IF;
 ahora:=pg_catalog.clock_timestamp();
 IF ahora>=LEAST(pol.vigente_hasta,v.vigente_hasta,p_revocada+pol.maxima_edad_revocacion) THEN RETURN; END IF;
 RETURN QUERY SELECT v.persona_ref,c.cuenta_ref,c.cuenta_ordinaria_ref,v.vinculo_ref,v.version,pol.politica_ref,pol.huella_aprobacion_sha256,LEAST(pol.vigente_hasta,v.vigente_hasta,p_revocada+pol.maxima_edad_revocacion);
EXCEPTION WHEN no_data_found OR too_many_rows OR data_exception THEN RETURN;
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.resolver_identidad_admin_perfiles_propietaria_v1(text,text,text,text,text,timestamptz,timestamptz) FROM PUBLIC;

CREATE FUNCTION vec_identidad_sesiones_v1.resolver_cuenta_admin_perfiles_propietaria_v1(p_entorno text,p_host text,p_audiencia text,p_certificado text,p_ca text,p_autenticada timestamptz,p_revocada timestamptz)
RETURNS TABLE(sujeto_id text,cuenta_id text,cuenta_ordinaria_id text,persona_ref text,cuenta_ref text,cuenta_ordinaria_ref text,perfil_activo_ref text,rol_id text,vinculo_ref text,vinculo_version numeric,politica_garantia_ref text,politica_garantia_huella_sha256 text,garantia_observada text,vigente_hasta timestamptz,seleccion_revision numeric)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE r record;b record;ahora timestamptz;
BEGIN
 SELECT * INTO STRICT r FROM vec_identidad_sesiones_v1.resolver_identidad_admin_perfiles_propietaria_v1(p_entorno,p_host,p_audiencia,p_certificado,p_ca,p_autenticada,p_revocada);
 SELECT * INTO STRICT b FROM vec_contexto_actor_v1.consultar_perfil_admin_perfiles_v1(r.cuenta_ref);
 ahora:=pg_catalog.clock_timestamp();
 IF b.persona_ref IS DISTINCT FROM r.persona_ref OR b.audiencia IS DISTINCT FROM p_audiencia OR ahora>=LEAST(r.vigente_hasta,b.vigente_hasta) THEN RETURN; END IF;
 RETURN QUERY SELECT 'admin-persona-v1:'||r.persona_ref,'admin-cuenta-v1:'||pg_catalog.lower(r.cuenta_ref),'admin-cuenta-v1:'||pg_catalog.lower(r.cuenta_ordinaria_ref),r.persona_ref,r.cuenta_ref,r.cuenta_ordinaria_ref,b.perfil_ref,'administracion_perfiles'::text,r.vinculo_ref,r.vinculo_version,r.politica_ref,r.politica_huella_sha256,'alto'::text,LEAST(r.vigente_hasta,b.vigente_hasta),b.seleccion_revision;
EXCEPTION WHEN no_data_found OR too_many_rows OR data_exception THEN RETURN;
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.resolver_cuenta_admin_perfiles_propietaria_v1(text,text,text,text,text,timestamptz,timestamptz) FROM PUBLIC;
CREATE FUNCTION vec_identidad_sesiones_v1.resolver_cuenta_admin_perfiles_v1(p_entorno text,p_host text,p_audiencia text,p_certificado text,p_ca text,p_autenticada timestamptz,p_revocada timestamptz)
RETURNS TABLE(sujeto_id text,cuenta_id text,cuenta_ordinaria_id text,persona_ref text,cuenta_ref text,cuenta_ordinaria_ref text,perfil_activo_ref text,rol_id text,vinculo_ref text,vinculo_version numeric,politica_garantia_ref text,politica_garantia_huella_sha256 text,garantia_observada text,vigente_hasta timestamptz,seleccion_revision numeric)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_admin_perfiles_v1();
 RETURN QUERY SELECT * FROM vec_identidad_sesiones_v1.resolver_cuenta_admin_perfiles_propietaria_v1(p_entorno,p_host,p_audiencia,p_certificado,p_ca,p_autenticada,p_revocada);
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.resolver_cuenta_admin_perfiles_v1(text,text,text,text,text,timestamptz,timestamptz) FROM PUBLIC;

-- El listado es propio de la cuenta derivada de la observación F, sin perfil
-- solicitado como autoridad y sin credencial expuesta al cliente.
CREATE FUNCTION vec_identidad_sesiones_v1.listar_perfiles_admin_v1(p_entorno text,p_host text,p_audiencia text,p_certificado text,p_ca text,p_autenticada timestamptz,p_revocada timestamptz,p_crl_vigente_hasta timestamptz,p_certificado_vigente_hasta timestamptz)
RETURNS TABLE(persona_ref text,cuenta_ref text,perfil_ref text,vinculo_ref text,audiencia text,vigente_hasta timestamptz,seleccionado boolean,seleccion_revision numeric,rol_version_ref text,clave_i18n text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE r record;b record;sel record;hasta timestamptz;
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_admin_perfiles_v1();
 IF current_setting('transaction_isolation')<>'read committed' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'IS12: listado requiere READ COMMITTED de escritura' USING ERRCODE='25000'; END IF;
 IF p_crl_vigente_hasta IS NULL OR p_certificado_vigente_hasta IS NULL OR NOT pg_catalog.isfinite(p_crl_vigente_hasta) OR NOT pg_catalog.isfinite(p_certificado_vigente_hasta) THEN RETURN; END IF;
 SELECT * INTO STRICT r FROM vec_identidad_sesiones_v1.resolver_identidad_admin_perfiles_propietaria_v1(p_entorno,p_host,p_audiencia,p_certificado,p_ca,p_autenticada,p_revocada);
 hasta:=LEAST(r.vigente_hasta,p_crl_vigente_hasta,p_certificado_vigente_hasta);
 IF pg_catalog.clock_timestamp()>=hasta THEN RETURN; END IF;
 SELECT * INTO sel FROM vec_contexto_actor_v1.consultar_seleccion_perfil_admin_v1(r.cuenta_ref);
 IF sel.persona_ref IS NOT NULL AND sel.persona_ref IS DISTINCT FROM r.persona_ref THEN RETURN; END IF;
 FOR b IN SELECT x.* FROM vec_contexto_actor_v1.listar_perfiles_admin_propios_reconciliacion_v1(r.cuenta_ref,r.persona_ref,p_audiencia) x ORDER BY x.perfil_ref LOOP
  IF pg_catalog.clock_timestamp()>=LEAST(hasta,b.vigente_hasta) THEN CONTINUE; END IF;
  RETURN QUERY SELECT r.persona_ref,r.cuenta_ref,b.perfil_ref,b.vinculo_ref,b.audiencia,LEAST(hasta,b.vigente_hasta),COALESCE(sel.perfil_ref=b.perfil_ref AND sel.fuente=b.fuente,false),COALESCE(sel.seleccion_revision,0::numeric),b.rol_version_ref,b.clave_i18n;
 END LOOP;
EXCEPTION WHEN no_data_found OR too_many_rows OR data_exception THEN RETURN;
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.listar_perfiles_admin_v1(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz) FROM PUBLIC;

CREATE FUNCTION vec_identidad_sesiones_v1.seleccionar_perfil_admin_v1(p_entorno text,p_host text,p_audiencia text,p_certificado text,p_ca text,p_autenticada timestamptz,p_revocada timestamptz,p_crl_vigente_hasta timestamptz,p_certificado_vigente_hasta timestamptz,p_perfil_ref text,p_revision_esperada numeric)
RETURNS TABLE(perfil_ref text,seleccion_revision numeric,seleccionada_en timestamptz,auditoria_ref text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE r record;hasta timestamptz;
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_admin_perfiles_v1();
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'IS12: selección requiere SERIALIZABLE de escritura' USING ERRCODE='25000'; END IF;
 IF p_crl_vigente_hasta IS NULL OR p_certificado_vigente_hasta IS NULL OR NOT pg_catalog.isfinite(p_crl_vigente_hasta) OR NOT pg_catalog.isfinite(p_certificado_vigente_hasta) THEN RAISE EXCEPTION 'IS12: observación de selección inválida' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT r FROM vec_identidad_sesiones_v1.resolver_identidad_admin_perfiles_propietaria_v1(p_entorno,p_host,p_audiencia,p_certificado,p_ca,p_autenticada,p_revocada);
 hasta:=LEAST(r.vigente_hasta,p_crl_vigente_hasta,p_certificado_vigente_hasta);
 IF pg_catalog.clock_timestamp()>=hasta THEN RAISE EXCEPTION 'IS12: observación de selección caducada' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT * FROM vec_contexto_actor_v1.seleccionar_perfil_admin_propietaria_v1(r.cuenta_ref,r.persona_ref,p_perfil_ref,p_revision_esperada,p_audiencia,p_autenticada,hasta,'eleccion_explicita');
EXCEPTION WHEN no_data_found OR too_many_rows THEN RAISE EXCEPTION 'IS12: identidad de selección no acreditada' USING ERRCODE='42501';
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.seleccionar_perfil_admin_v1(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz,text,numeric) FROM PUBLIC;

CREATE FUNCTION vec_identidad_sesiones_v1.autoseleccionar_perfil_admin_unico_v1(p_entorno text,p_host text,p_audiencia text,p_certificado text,p_ca text,p_autenticada timestamptz,p_revocada timestamptz,p_crl_vigente_hasta timestamptz,p_certificado_vigente_hasta timestamptz)
RETURNS TABLE(perfil_ref text,seleccion_revision numeric,seleccionada_en timestamptz,auditoria_ref text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_admin_perfiles_v1();
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'IS12: selección requiere SERIALIZABLE de escritura' USING ERRCODE='25000'; END IF;
 -- La ausencia de fuente nominal de Sistemas no acredita unicidad entre
 -- perfiles ADMIN. La elección debe ser explícita hasta cerrar ese recuento.
 RETURN;
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.autoseleccionar_perfil_admin_unico_v1(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.listar_perfiles_admin_v1(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz),vec_identidad_sesiones_v1.seleccionar_perfil_admin_v1(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz,text,numeric),vec_identidad_sesiones_v1.autoseleccionar_perfil_admin_unico_v1(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz) TO vec_identidad_sesiones_v1_admin_perfiles;

CREATE FUNCTION vec_identidad_sesiones_v1.vincular_sesion_admin_perfiles_v1(p_entorno text,p_host text,p_audiencia text,p_certificado text,p_ca text,p_autenticada timestamptz,p_revocada timestamptz,p_crl_vigente_hasta timestamptz,p_certificado_vigente_hasta timestamptz,p_autenticacion_ref text,p_sesion_ref text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE r record;s record;prev record;doc jsonb;hasta timestamptz;
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_admin_perfiles_v1();
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'IS12: requiere SERIALIZABLE' USING ERRCODE='25000'; END IF;
 IF p_crl_vigente_hasta IS NULL OR p_certificado_vigente_hasta IS NULL
 OR NOT pg_catalog.isfinite(p_crl_vigente_hasta) OR NOT pg_catalog.isfinite(p_certificado_vigente_hasta)
 OR LEAST(p_crl_vigente_hasta,p_certificado_vigente_hasta)<=pg_catalog.clock_timestamp() THEN RETURN false; END IF;
 SELECT * INTO STRICT r FROM vec_identidad_sesiones_v1.resolver_cuenta_admin_perfiles_propietaria_v1(p_entorno,p_host,p_audiencia,p_certificado,p_ca,p_autenticada,p_revocada);
 SELECT * INTO STRICT s FROM vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(p_autenticacion_ref,p_sesion_ref);
 IF s.cuenta_ref IS DISTINCT FROM r.cuenta_ref OR s.cuenta_ordinaria_ref IS DISTINCT FROM r.cuenta_ordinaria_ref
 OR s.cuenta_privilegiada IS DISTINCT FROM true OR s.superficie IS DISTINCT FROM 'administracion_privilegiada'
 OR s.metodo_observado IS DISTINCT FROM 'certificado' OR s.garantia_observada IS DISTINCT FROM 'alto'
 OR s.politica_garantia_ref IS DISTINCT FROM r.politica_garantia_ref OR s.politica_garantia_huella_sha256 IS DISTINCT FROM r.politica_garantia_huella_sha256
 OR s.autenticacion_verificada_en IS DISTINCT FROM p_autenticada OR s.sesion_valida_hasta>r.vigente_hasta THEN RETURN false; END IF;
 hasta:=LEAST(r.vigente_hasta,s.sesion_valida_hasta,p_crl_vigente_hasta,p_certificado_vigente_hasta);
 IF hasta<=GREATEST(p_autenticada,p_revocada) OR hasta<=pg_catalog.clock_timestamp() THEN RETURN false; END IF;
 doc:=pg_catalog.jsonb_build_object('esquema','vec.admin-perfiles.sesion.v1','autenticacion_ref',p_autenticacion_ref,'sesion_ref',p_sesion_ref,'autenticacion_huella_sha256',s.autenticacion_huella_sha256,'cuenta_ref',r.cuenta_ref,'cuenta_ordinaria_ref',r.cuenta_ordinaria_ref,'persona_ref',r.persona_ref,'perfil_ref',r.perfil_activo_ref,'seleccion_revision',r.seleccion_revision,'certificado_vinculo_ref',r.vinculo_ref,'certificado_vinculo_version',r.vinculo_version,'politica_ref',r.politica_garantia_ref,'politica_huella_sha256',r.politica_garantia_huella_sha256,'entorno',p_entorno,'host',p_host,'audiencia',p_audiencia,'autenticacion_verificada_en',p_autenticada,'revocacion_verificada_en',p_revocada,'crl_vigente_hasta',p_crl_vigente_hasta,'certificado_vigente_hasta',p_certificado_vigente_hasta,'observacion_vigente_hasta',hasta,'vigente_hasta',hasta);
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:sesion:v1:'||p_autenticacion_ref,0));
 -- La espera por la clave de sesión no puede convertir una CRL caducada en
 -- una vinculación positiva; el consumidor vuelve a comprobarla al actuar.
 IF hasta<=pg_catalog.clock_timestamp() THEN RETURN false; END IF;
 SELECT * INTO prev FROM vec_identidad_sesiones_v1.sesion_admin_perfiles_v1 WHERE autenticacion_ref=p_autenticacion_ref;
 IF FOUND THEN RETURN prev.snapshot IS NOT DISTINCT FROM doc; END IF;
 INSERT INTO vec_identidad_sesiones_v1.sesion_admin_perfiles_v1 VALUES(p_autenticacion_ref,p_sesion_ref,r.cuenta_ref,r.cuenta_ordinaria_ref,r.persona_ref,r.perfil_activo_ref,r.seleccion_revision,r.vinculo_ref,r.vinculo_version,r.politica_garantia_ref,r.politica_garantia_huella_sha256,p_entorno,p_host,p_audiencia,p_autenticada,p_revocada,p_crl_vigente_hasta,p_certificado_vigente_hasta,hasta,hasta,doc,pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(doc::text,'UTF8')),'hex'),pg_catalog.clock_timestamp());
 RETURN true;
EXCEPTION WHEN no_data_found OR too_many_rows OR data_exception THEN RETURN false;
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.vincular_sesion_admin_perfiles_v1(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz,text,text) FROM PUBLIC;

-- Consumidor de efecto/replay: la sesión común y el certificado exacto se
-- revalidan antes del mismo COMMIT. El JSON del cliente no es autoridad.
CREATE FUNCTION vec_identidad_sesiones_v1.revalidar_sesion_admin_perfiles_v1(p_autenticacion_ref text,p_sesion_ref text,p_cuenta_ref text,p_cuenta_ordinaria_ref text,p_persona_ref text,p_perfil_ref text,p_politica_ref text,p_politica_huella_sha256 text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE b record;s record;v record;r record;ahora timestamptz;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RETURN false; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 SELECT * INTO STRICT b FROM vec_identidad_sesiones_v1.sesion_admin_perfiles_v1 WHERE autenticacion_ref=p_autenticacion_ref AND sesion_ref=p_sesion_ref FOR SHARE;
 IF (b.cuenta_ref,b.cuenta_ordinaria_ref,b.persona_ref,b.perfil_ref,b.politica_ref,b.politica_huella_sha256) IS DISTINCT FROM (p_cuenta_ref,p_cuenta_ordinaria_ref,p_persona_ref,p_perfil_ref,p_politica_ref,p_politica_huella_sha256) THEN RETURN false; END IF;
 SELECT * INTO STRICT s FROM vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(p_autenticacion_ref,p_sesion_ref);
 IF s.cuenta_privilegiada IS DISTINCT FROM true OR s.superficie IS DISTINCT FROM 'administracion_privilegiada'
 OR s.metodo_observado IS DISTINCT FROM 'certificado' OR s.garantia_observada IS DISTINCT FROM 'alto'
 OR s.cuenta_ref IS DISTINCT FROM p_cuenta_ref OR s.cuenta_ordinaria_ref IS DISTINCT FROM p_cuenta_ordinaria_ref OR s.politica_garantia_ref IS DISTINCT FROM p_politica_ref OR s.politica_garantia_huella_sha256 IS DISTINCT FROM p_politica_huella_sha256 OR s.autenticacion_huella_sha256 IS DISTINCT FROM b.snapshot->>'autenticacion_huella_sha256' THEN RETURN false; END IF;
 SELECT x.* INTO STRICT v FROM vec_identidad_sesiones_v1.vinculo_certificado_admin_actual_v1 a JOIN vec_identidad_sesiones_v1.vinculo_certificado_admin_v1 x USING(vinculo_ref,version) WHERE a.vinculo_ref=b.certificado_vinculo_ref FOR SHARE OF a;
 IF v.version IS DISTINCT FROM b.certificado_vinculo_version THEN RETURN false; END IF;
 SELECT * INTO STRICT r FROM vec_identidad_sesiones_v1.resolver_cuenta_admin_perfiles_propietaria_v1(b.entorno,b.host,b.audiencia,v.certificado_sha256,v.ca_sha256,b.autenticacion_verificada_en,b.revocacion_verificada_en);
 ahora:=pg_catalog.clock_timestamp();
 RETURN (r.cuenta_ref,r.cuenta_ordinaria_ref,r.persona_ref,r.perfil_activo_ref,r.seleccion_revision,r.vinculo_ref,r.vinculo_version,r.politica_garantia_ref,r.politica_garantia_huella_sha256) IS NOT DISTINCT FROM (b.cuenta_ref,b.cuenta_ordinaria_ref,b.persona_ref,b.perfil_ref,b.seleccion_revision,b.certificado_vinculo_ref,b.certificado_vinculo_version,b.politica_ref,b.politica_huella_sha256) AND ahora<b.vigente_hasta AND ahora<r.vigente_hasta AND ahora<b.observacion_vigente_hasta
 AND ahora<b.crl_vigente_hasta AND ahora<b.certificado_vigente_hasta;
EXCEPTION WHEN no_data_found OR too_many_rows OR data_exception THEN RETURN false;
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.revalidar_sesion_admin_perfiles_v1(text,text,text,text,text,text,text,text) FROM PUBLIC;

ALTER TABLE vec_identidad_sesiones_v1.preimagen_acl_admin_perfiles_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_identidad_sesiones_v1.preimagen_acl_admin_perfiles_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_identidad_sesiones_v1.preimagen_acl_admin_perfiles_v1 FOR ALL TO vec_identidad_sesiones_v1_propietario USING(current_user='vec_identidad_sesiones_v1_propietario') WITH CHECK(current_user='vec_identidad_sesiones_v1_propietario');
REVOKE ALL ON TABLE vec_identidad_sesiones_v1.preimagen_acl_admin_perfiles_v1 FROM PUBLIC,vec_identidad_sesiones_v1_admin_perfiles;
REVOKE ALL ON TYPE vec_identidad_sesiones_v1.preimagen_acl_admin_perfiles_v1 FROM PUBLIC,vec_identidad_sesiones_v1_admin_perfiles;
ALTER TABLE vec_identidad_sesiones_v1.sesion_admin_perfiles_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_identidad_sesiones_v1.sesion_admin_perfiles_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_identidad_sesiones_v1.sesion_admin_perfiles_v1 FOR ALL TO vec_identidad_sesiones_v1_propietario USING(current_user='vec_identidad_sesiones_v1_propietario') WITH CHECK(current_user='vec_identidad_sesiones_v1_propietario');
REVOKE ALL ON TABLE vec_identidad_sesiones_v1.sesion_admin_perfiles_v1 FROM PUBLIC,vec_identidad_sesiones_v1_admin_perfiles;
REVOKE ALL ON TYPE vec_identidad_sesiones_v1.sesion_admin_perfiles_v1 FROM PUBLIC,vec_identidad_sesiones_v1_admin_perfiles;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.resolver_cuenta_admin_perfiles_propietaria_v1(text,text,text,text,text,timestamptz,timestamptz),vec_identidad_sesiones_v1.resolver_cuenta_admin_perfiles_v1(text,text,text,text,text,timestamptz,timestamptz),vec_identidad_sesiones_v1.vincular_sesion_admin_perfiles_v1(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz,text,text),vec_identidad_sesiones_v1.revalidar_sesion_admin_perfiles_v1(text,text,text,text,text,text,text,text) FROM PUBLIC,vec_identidad_sesiones_v1_admin_perfiles;
GRANT USAGE ON SCHEMA vec_identidad_sesiones_v1 TO vec_identidad_sesiones_v1_admin_perfiles,vec_autorizacion_atestada_v3_propietario;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.resolver_cuenta_admin_perfiles_v1(text,text,text,text,text,timestamptz,timestamptz),vec_identidad_sesiones_v1.vincular_sesion_admin_perfiles_v1(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz,text,text) TO vec_identidad_sesiones_v1_admin_perfiles;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.revalidar_sesion_admin_perfiles_v1(text,text,text,text,text,text,text,text) TO vec_autorizacion_atestada_v3_propietario;
COMMIT;
