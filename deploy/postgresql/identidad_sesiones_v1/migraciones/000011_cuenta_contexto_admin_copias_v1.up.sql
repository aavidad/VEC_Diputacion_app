\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_identidad_sesiones_v1:migracion:000011',0));
DO $preimagen$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 OR pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.revalidar_certificado_admin_v1(text,text,text,text,text,text,boolean,timestamptz,boolean,boolean)') IS NULL
 OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.consultar_perfil_admin_copias_v1(text)') IS NULL
 OR pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)') IS NULL
 OR pg_catalog.to_regrole('vec_autorizacion_atestada_v3_propietario') IS NULL
 OR pg_catalog.to_regclass('vec_identidad_sesiones_v1.sesion_admin_copias_v1') IS NOT NULL
 THEN RAISE EXCEPTION 'IS11: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $preimagen$;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;

CREATE TABLE vec_identidad_sesiones_v1.preimagen_acl_admin_copias_v1(singleton boolean PRIMARY KEY CHECK(singleton),uso_ad3_previo boolean NOT NULL);
INSERT INTO vec_identidad_sesiones_v1.preimagen_acl_admin_copias_v1 VALUES(true,pg_catalog.has_schema_privilege('vec_autorizacion_atestada_v3_propietario','vec_identidad_sesiones_v1','USAGE'));
CREATE TRIGGER preimagen_acl_admin_copias_historia BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_identidad_sesiones_v1.preimagen_acl_admin_copias_v1 FOR EACH STATEMENT EXECUTE FUNCTION vec_identidad_sesiones_v1.rechazar_mutacion();
CREATE TABLE vec_identidad_sesiones_v1.sesion_admin_copias_v1(
 autenticacion_ref text PRIMARY KEY,sesion_ref text NOT NULL UNIQUE,
 cuenta_ref text NOT NULL,cuenta_ordinaria_ref text NOT NULL,persona_ref text NOT NULL,perfil_ref text NOT NULL,
 certificado_vinculo_ref text NOT NULL,certificado_vinculo_version numeric(20,0) NOT NULL,
 politica_ref text NOT NULL,politica_huella_sha256 text NOT NULL,
 entorno text NOT NULL,host text NOT NULL,audiencia text NOT NULL,
 autenticacion_verificada_en timestamptz(6) NOT NULL,revocacion_verificada_en timestamptz(6) NOT NULL,
 vigente_hasta timestamptz(6) NOT NULL,
 snapshot jsonb NOT NULL,snapshot_sha256 text NOT NULL,
 registrada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
 FOREIGN KEY(autenticacion_ref) REFERENCES vec_identidad_sesiones_v1.consumo_asercion(autenticacion_ref),
 FOREIGN KEY(sesion_ref) REFERENCES vec_identidad_sesiones_v1.consumo_asercion(sesion_ref),
 FOREIGN KEY(certificado_vinculo_ref,certificado_vinculo_version) REFERENCES vec_identidad_sesiones_v1.vinculo_certificado_admin_v1(vinculo_ref,version),
 CHECK(cuenta_ref<>cuenta_ordinaria_ref),
 CHECK(snapshot_sha256=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(snapshot::text,'UTF8')),'hex'))
);
CREATE TRIGGER sesion_admin_copias_historia BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_identidad_sesiones_v1.sesion_admin_copias_v1 FOR EACH STATEMENT EXECUTE FUNCTION vec_identidad_sesiones_v1.rechazar_mutacion();

-- Solo referencias centrales y alias explícitos para el HMAC ya aprovisionado.
-- No devuelve certificado, credencial ni secreto de política.
CREATE FUNCTION vec_identidad_sesiones_v1.resolver_cuenta_admin_copias_propietaria_v1(p_entorno text,p_host text,p_audiencia text,p_certificado text,p_ca text,p_autenticada timestamptz,p_revocada timestamptz)
RETURNS TABLE(sujeto_id text,cuenta_id text,cuenta_ordinaria_id text,persona_ref text,cuenta_ref text,cuenta_ordinaria_ref text,perfil_activo_ref text,rol_id text,vinculo_ref text,vinculo_version numeric,politica_garantia_ref text,politica_garantia_huella_sha256 text,garantia_observada text,vigente_hasta timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE pol record;v record;c record;b record;ahora timestamptz;
BEGIN
 IF current_setting('transaction_read_only')<>'off' OR current_setting('transaction_isolation')='repeatable read'
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
 SELECT * INTO STRICT b FROM vec_contexto_actor_v1.consultar_perfil_admin_copias_v1(c.cuenta_ref);
 IF b.persona_ref IS DISTINCT FROM v.persona_ref OR b.audiencia IS DISTINCT FROM p_audiencia
 OR c.cuenta_ordinaria_ref=c.cuenta_ref OR NOT c.cuenta_privilegiada
 OR NOT EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.cuenta o WHERE o.cuenta_ref=c.cuenta_ordinaria_ref AND NOT o.cuenta_privilegiada AND o.cuenta_ordinaria_ref IS NULL) THEN RETURN; END IF;
 RETURN QUERY SELECT 'admin-persona-v1:'||v.persona_ref,'admin-cuenta-v1:'||c.cuenta_ref,'admin-cuenta-v1:'||c.cuenta_ordinaria_ref,v.persona_ref,c.cuenta_ref,c.cuenta_ordinaria_ref,b.perfil_ref,'operador_plataforma'::text,v.vinculo_ref,v.version,pol.politica_ref,pol.huella_aprobacion_sha256,'alto'::text,LEAST(pol.vigente_hasta,v.vigente_hasta,b.vigente_hasta,p_revocada+pol.maxima_edad_revocacion);
EXCEPTION WHEN no_data_found OR too_many_rows OR data_exception THEN RETURN;
END $f$;
CREATE FUNCTION vec_identidad_sesiones_v1.resolver_cuenta_admin_copias_v1(p_entorno text,p_host text,p_audiencia text,p_certificado text,p_ca text,p_autenticada timestamptz,p_revocada timestamptz)
RETURNS TABLE(sujeto_id text,cuenta_id text,cuenta_ordinaria_id text,persona_ref text,cuenta_ref text,cuenta_ordinaria_ref text,perfil_activo_ref text,rol_id text,vinculo_ref text,vinculo_version numeric,politica_garantia_ref text,politica_garantia_huella_sha256 text,garantia_observada text,vigente_hasta timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_admin_copias_v1();
 RETURN QUERY SELECT * FROM vec_identidad_sesiones_v1.resolver_cuenta_admin_copias_propietaria_v1(p_entorno,p_host,p_audiencia,p_certificado,p_ca,p_autenticada,p_revocada);
END $f$;

CREATE FUNCTION vec_identidad_sesiones_v1.vincular_sesion_admin_copias_v1(p_entorno text,p_host text,p_audiencia text,p_certificado text,p_ca text,p_autenticada timestamptz,p_revocada timestamptz,p_autenticacion_ref text,p_sesion_ref text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE r record;s record;prev record;doc jsonb;hasta timestamptz;
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_admin_copias_v1();
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'IS11: requiere SERIALIZABLE' USING ERRCODE='25000'; END IF;
 SELECT * INTO STRICT r FROM vec_identidad_sesiones_v1.resolver_cuenta_admin_copias_propietaria_v1(p_entorno,p_host,p_audiencia,p_certificado,p_ca,p_autenticada,p_revocada);
 SELECT * INTO STRICT s FROM vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(p_autenticacion_ref,p_sesion_ref);
 IF s.cuenta_ref IS DISTINCT FROM r.cuenta_ref OR s.cuenta_ordinaria_ref IS DISTINCT FROM r.cuenta_ordinaria_ref
 OR s.cuenta_privilegiada IS DISTINCT FROM true OR s.superficie IS DISTINCT FROM 'administracion_privilegiada'
 OR s.metodo_observado IS DISTINCT FROM 'certificado' OR s.garantia_observada IS DISTINCT FROM 'alto'
 OR s.politica_garantia_ref IS DISTINCT FROM r.politica_garantia_ref OR s.politica_garantia_huella_sha256 IS DISTINCT FROM r.politica_garantia_huella_sha256
 OR s.autenticacion_verificada_en IS DISTINCT FROM p_autenticada OR s.sesion_valida_hasta>r.vigente_hasta THEN RETURN false; END IF;
 hasta:=LEAST(r.vigente_hasta,s.sesion_valida_hasta);
 doc:=pg_catalog.jsonb_build_object('esquema','vec.admin-copias.sesion.v1','autenticacion_ref',p_autenticacion_ref,'sesion_ref',p_sesion_ref,'autenticacion_huella_sha256',s.autenticacion_huella_sha256,'cuenta_ref',r.cuenta_ref,'cuenta_ordinaria_ref',r.cuenta_ordinaria_ref,'persona_ref',r.persona_ref,'perfil_ref',r.perfil_activo_ref,'certificado_vinculo_ref',r.vinculo_ref,'certificado_vinculo_version',r.vinculo_version,'politica_ref',r.politica_garantia_ref,'politica_huella_sha256',r.politica_garantia_huella_sha256,'entorno',p_entorno,'host',p_host,'audiencia',p_audiencia,'autenticacion_verificada_en',p_autenticada,'revocacion_verificada_en',p_revocada,'vigente_hasta',hasta);
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:sesion:v1:'||p_autenticacion_ref,0));
 SELECT * INTO prev FROM vec_identidad_sesiones_v1.sesion_admin_copias_v1 WHERE autenticacion_ref=p_autenticacion_ref;
 IF FOUND THEN RETURN prev.snapshot IS NOT DISTINCT FROM doc; END IF;
 INSERT INTO vec_identidad_sesiones_v1.sesion_admin_copias_v1 VALUES(p_autenticacion_ref,p_sesion_ref,r.cuenta_ref,r.cuenta_ordinaria_ref,r.persona_ref,r.perfil_activo_ref,r.vinculo_ref,r.vinculo_version,r.politica_garantia_ref,r.politica_garantia_huella_sha256,p_entorno,p_host,p_audiencia,p_autenticada,p_revocada,hasta,doc,pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(doc::text,'UTF8')),'hex'),pg_catalog.clock_timestamp());
 RETURN true;
EXCEPTION WHEN no_data_found OR too_many_rows OR data_exception THEN RETURN false;
END $f$;

-- Consumidor de efecto/replay: la sesión común y el certificado exacto se
-- revalidan antes del mismo COMMIT. El JSON del cliente no es autoridad.
CREATE FUNCTION vec_identidad_sesiones_v1.revalidar_sesion_admin_copias_v1(p_autenticacion_ref text,p_sesion_ref text,p_cuenta_ref text,p_cuenta_ordinaria_ref text,p_persona_ref text,p_perfil_ref text,p_politica_ref text,p_politica_huella_sha256 text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE b record;s record;v record;r record;ahora timestamptz;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RETURN false; END IF;
 -- El mismo orden global que la vinculación y la revocación del certificado:
 -- continuidad antes del control de sesión, las cuentas y el vínculo.
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 SELECT * INTO STRICT b FROM vec_identidad_sesiones_v1.sesion_admin_copias_v1 WHERE autenticacion_ref=p_autenticacion_ref AND sesion_ref=p_sesion_ref FOR SHARE;
 IF (b.cuenta_ref,b.cuenta_ordinaria_ref,b.persona_ref,b.perfil_ref,b.politica_ref,b.politica_huella_sha256) IS DISTINCT FROM (p_cuenta_ref,p_cuenta_ordinaria_ref,p_persona_ref,p_perfil_ref,p_politica_ref,p_politica_huella_sha256) THEN RETURN false; END IF;
 SELECT * INTO STRICT s FROM vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(p_autenticacion_ref,p_sesion_ref);
 IF s.cuenta_ref IS DISTINCT FROM p_cuenta_ref OR s.cuenta_ordinaria_ref IS DISTINCT FROM p_cuenta_ordinaria_ref OR s.politica_garantia_ref IS DISTINCT FROM p_politica_ref OR s.politica_garantia_huella_sha256 IS DISTINCT FROM p_politica_huella_sha256 OR s.autenticacion_huella_sha256 IS DISTINCT FROM b.snapshot->>'autenticacion_huella_sha256' THEN RETURN false; END IF;
 SELECT x.* INTO STRICT v FROM vec_identidad_sesiones_v1.vinculo_certificado_admin_actual_v1 a JOIN vec_identidad_sesiones_v1.vinculo_certificado_admin_v1 x USING(vinculo_ref,version) WHERE a.vinculo_ref=b.certificado_vinculo_ref FOR SHARE OF a;
 IF v.version IS DISTINCT FROM b.certificado_vinculo_version THEN RETURN false; END IF;
 SELECT * INTO STRICT r FROM vec_identidad_sesiones_v1.resolver_cuenta_admin_copias_propietaria_v1(b.entorno,b.host,b.audiencia,v.certificado_sha256,v.ca_sha256,b.autenticacion_verificada_en,b.revocacion_verificada_en);
 ahora:=pg_catalog.clock_timestamp();
 RETURN (r.cuenta_ref,r.cuenta_ordinaria_ref,r.persona_ref,r.perfil_activo_ref,r.vinculo_ref,r.vinculo_version,r.politica_garantia_ref,r.politica_garantia_huella_sha256) IS NOT DISTINCT FROM (b.cuenta_ref,b.cuenta_ordinaria_ref,b.persona_ref,b.perfil_ref,b.certificado_vinculo_ref,b.certificado_vinculo_version,b.politica_ref,b.politica_huella_sha256) AND ahora<b.vigente_hasta AND ahora<r.vigente_hasta;
EXCEPTION WHEN no_data_found OR too_many_rows OR data_exception THEN RETURN false;
END $f$;

ALTER TABLE vec_identidad_sesiones_v1.preimagen_acl_admin_copias_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_identidad_sesiones_v1.preimagen_acl_admin_copias_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_identidad_sesiones_v1.preimagen_acl_admin_copias_v1 FOR ALL TO vec_identidad_sesiones_v1_propietario USING(current_user='vec_identidad_sesiones_v1_propietario') WITH CHECK(current_user='vec_identidad_sesiones_v1_propietario');
REVOKE ALL ON TABLE vec_identidad_sesiones_v1.preimagen_acl_admin_copias_v1 FROM PUBLIC,vec_identidad_sesiones_v1_admin_copias;
REVOKE ALL ON TYPE vec_identidad_sesiones_v1.preimagen_acl_admin_copias_v1 FROM PUBLIC,vec_identidad_sesiones_v1_admin_copias;
ALTER TABLE vec_identidad_sesiones_v1.sesion_admin_copias_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_identidad_sesiones_v1.sesion_admin_copias_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_identidad_sesiones_v1.sesion_admin_copias_v1 FOR ALL TO vec_identidad_sesiones_v1_propietario USING(current_user='vec_identidad_sesiones_v1_propietario') WITH CHECK(current_user='vec_identidad_sesiones_v1_propietario');
REVOKE ALL ON TABLE vec_identidad_sesiones_v1.sesion_admin_copias_v1 FROM PUBLIC,vec_identidad_sesiones_v1_admin_copias;
REVOKE ALL ON TYPE vec_identidad_sesiones_v1.sesion_admin_copias_v1 FROM PUBLIC,vec_identidad_sesiones_v1_admin_copias;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.resolver_cuenta_admin_copias_propietaria_v1(text,text,text,text,text,timestamptz,timestamptz),vec_identidad_sesiones_v1.resolver_cuenta_admin_copias_v1(text,text,text,text,text,timestamptz,timestamptz),vec_identidad_sesiones_v1.vincular_sesion_admin_copias_v1(text,text,text,text,text,timestamptz,timestamptz,text,text),vec_identidad_sesiones_v1.revalidar_sesion_admin_copias_v1(text,text,text,text,text,text,text,text) FROM PUBLIC,vec_identidad_sesiones_v1_admin_copias;
GRANT USAGE ON SCHEMA vec_identidad_sesiones_v1 TO vec_identidad_sesiones_v1_admin_copias,vec_autorizacion_atestada_v3_propietario;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.resolver_cuenta_admin_copias_v1(text,text,text,text,text,timestamptz,timestamptz),vec_identidad_sesiones_v1.vincular_sesion_admin_copias_v1(text,text,text,text,text,timestamptz,timestamptz,text,text) TO vec_identidad_sesiones_v1_admin_copias;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.revalidar_sesion_admin_copias_v1(text,text,text,text,text,text,text,text) TO vec_autorizacion_atestada_v3_propietario;
COMMIT;
